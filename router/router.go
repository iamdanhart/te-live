package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/iamdanhart/te-live/catalog"
	"github.com/iamdanhart/te-live/config"
	"github.com/iamdanhart/te-live/grab_templates"
	"github.com/iamdanhart/te-live/middleware"
	"github.com/iamdanhart/te-live/queue"
)

func NewRouter(cfg config.Props) *http.ServeMux {
	rl := middleware.NewRateLimiter(2*time.Minute, cfg.EnforceSignupLimit)
	q, err := queue.NewPgQueue(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		handleIndex(w, r, q)
	})
	mux.HandleFunc("GET /signup", func(w http.ResponseWriter, r *http.Request) {
		handleSignupPage(w, r)
	})
	mux.Handle("POST /signup", rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSignup(w, r, q)
	})))
	mux.HandleFunc("GET /queue-status", func(w http.ResponseWriter, r *http.Request) {
		handleQueueStatus(w, r, q)
	})
	registerHostRoutes(mux, cfg, q)
	mux.HandleFunc("GET /catalog", handleCatalog)
	mux.Handle("GET /static/", staticHandler())
	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	slog.Info("ok")
}

func queueStatusData(q queue.Queue) struct {
	Current     *queue.Entry
	Next        *queue.Entry
	SignupsOpen bool
} {
	entries := q.Entries()
	var current, next *queue.Entry
	if len(entries) > 0 {
		current = &entries[0]
	}
	if len(entries) > 1 {
		next = &entries[1]
	}
	return struct {
		Current     *queue.Entry
		Next        *queue.Entry
		SignupsOpen bool
	}{current, next, q.SignupsOpen()}
}

func handleQueueStatus(w http.ResponseWriter, r *http.Request, q queue.Queue) {
	if err := grab_templates.GetTemplates().ExecuteTemplate(w, "index_queue.html", queueStatusData(q)); err != nil {
		slog.Error("template error", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request, q queue.Queue) {
	if err := grab_templates.GetTemplates().ExecuteTemplate(w, "index.html", queueStatusData(q)); err != nil {
		slog.Error("template error", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func handleSignupPage(w http.ResponseWriter, r *http.Request) {
	if err := grab_templates.GetTemplates().ExecuteTemplate(w, "signup.html", catalog.FullCatalog); err != nil {
		slog.Error("template error", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func handleSignup(w http.ResponseWriter, r *http.Request, q queue.Queue) {
	name := r.FormValue("name")
	var songs []catalog.Song
	for _, s := range r.Form["song"] {
		var id int
		if _, err := fmt.Sscan(s, &id); err != nil {
			slog.Error("failed to parse song id", "err", err)
			http.Error(w, "invalid song id", http.StatusBadRequest)
			return
		}
		song, ok := catalog.FindByID(id)
		if !ok {
			slog.Error("song not found", "id", id)
			http.Error(w, "song not found", http.StatusBadRequest)
			return
		}
		songs = append(songs, song)
	}
	if err := q.Add(name, songs); err != nil {
		slog.Error("failed to add signup", "name", name, "err", err)
		http.Error(w, "failed to save signup", http.StatusInternalServerError)
		return
	}
	slog.Info("signup", "name", name, "songs", songs)
	fmt.Fprintf(w, `<p>You're on the list, %s! See you up there.</p>`, name)
}

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(catalog.JSONBytes); err != nil {
		slog.Error("write error", "err", err)
	}
}
