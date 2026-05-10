package router

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iamdanhart/te-live/config"
	"github.com/iamdanhart/te-live/grab_templates"
	"github.com/iamdanhart/te-live/middleware"
	"github.com/iamdanhart/te-live/queue"
)

func NewRouter(cfg config.Props) http.Handler {
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
		handleSignupPage(w, r, q)
	})
	mux.HandleFunc("GET /signup/check-name", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name != "" && q.HasName(name) {
			if err := grab_templates.GetTemplates().ExecuteTemplate(w, "name_warning.html", nil); err != nil {
				slog.Error("template error", "err", err)
			}
		}
	})
	mux.Handle("POST /signup", rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSignup(w, r, q)
	})))
	mux.HandleFunc("GET /queue-status", func(w http.ResponseWriter, r *http.Request) {
		handleQueueStatus(w, r, q)
	})
	registerHostRoutes(mux, cfg, q)
	mux.HandleFunc("GET /catalog", func(w http.ResponseWriter, r *http.Request) {
		handleCatalog(w, r, q)
	})
	mux.Handle("GET /static/", staticHandler())
	return middleware.RequestLogger([]string{"/health", "/queue-status", "/host/queue"}, mux)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	// Implicit 200 signals healthy to Fly. No DB ping — a DB blip shouldn't trigger a machine restart.
}

func queueStatusData(q queue.Queue) struct {
	Current     *queue.Entry
	Next        *queue.Entry
	SignupsOpen bool
	Count       int
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
		Count       int
	}{current, next, q.SignupsOpen(), len(entries)}
}

func handleQueueStatus(w http.ResponseWriter, r *http.Request, q queue.Queue) {
	if err := grab_templates.GetTemplates().ExecuteTemplate(w, "index_queue.html", queueStatusData(q)); err != nil {
		slog.Error("template error", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request, q queue.Queue) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if err := grab_templates.GetTemplates().ExecuteTemplate(w, "index.html", queueStatusData(q)); err != nil {
		slog.Error("template error", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func handleSignupPage(w http.ResponseWriter, r *http.Request, q queue.Queue) {
	data := struct{ Songs []queue.Song }{q.Songs()}
	if err := grab_templates.GetTemplates().ExecuteTemplate(w, "signup.html", data); err != nil {
		slog.Error("template error", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func handleSignup(w http.ResponseWriter, r *http.Request, q queue.Queue) {
	name := r.FormValue("name")
	var songIDs []int
	for _, s := range r.Form["song"] {
		var id int
		if _, err := fmt.Sscan(s, &id); err != nil {
			slog.Error("failed to parse song id", "err", err)
			http.Error(w, "invalid song id", http.StatusBadRequest)
			return
		}
		songIDs = append(songIDs, id)
	}
	if err := q.Add(name, songIDs); err != nil {
		slog.Error("failed to add signup", "name", name, "err", err)
		http.Error(w, "failed to save signup", http.StatusInternalServerError)
		return
	}
	slog.Info("signup", "name", name, "songs", len(songIDs))
	if err := grab_templates.GetTemplates().ExecuteTemplate(w, "signup_success.html", name); err != nil {
		slog.Error("template error", "err", err)
	}
}

func handleCatalog(w http.ResponseWriter, r *http.Request, q queue.Queue) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Songs []queue.Song `json:"songs"`
	}{q.Songs()}); err != nil {
		slog.Error("catalog encode error", "err", err)
	}
}
