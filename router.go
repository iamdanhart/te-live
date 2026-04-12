package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/iamdanhart/te-live/catalog"
	"github.com/iamdanhart/te-live/config"
	"github.com/iamdanhart/te-live/middleware"
	"github.com/iamdanhart/te-live/queue"
)

func newRouter(cfg config.Props) *http.ServeMux {
	rl := middleware.NewRateLimiter(2*time.Minute, cfg.EnforceSignupLimit)
	q := queue.New()
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
	mux.Handle("GET /host", middleware.AdminAuth(cfg.EnforceAdminAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Entries   []queue.Entry
			Performed []queue.PerformedSong
		}{q.Entries(), q.Performed()}
		if err := getTemplates().ExecuteTemplate(w, "host.html", data); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	})))
	mux.Handle("GET /host/queue", middleware.AdminAuth(cfg.EnforceAdminAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := struct{ Entries []queue.Entry }{q.Entries()}
		if err := getTemplates().ExecuteTemplate(w, "host_queue.html", data); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	})))
	mux.Handle("POST /host/performed", middleware.AdminAuth(cfg.EnforceAdminAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		title, artist := r.FormValue("title"), r.FormValue("artist")
		q.MarkSongPerformed(title, artist)
		q.RecordPerformed(r.FormValue("singer"), catalog.Song{Title: title, Artist: artist})
		q.MoveCurrentToBottom()
		tmpl := getTemplates()
		if err := tmpl.ExecuteTemplate(w, "host_performed.html", struct{ Performed []queue.PerformedSong }{q.Performed()}); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "host_queue_oob.html", struct{ Entries []queue.Entry }{q.Entries()}); err != nil {
			slog.Error("template error", "err", err)
		}
	})))
	mux.Handle("POST /host/add-song", middleware.AdminAuth(cfg.EnforceAdminAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q.AddSongToFirst(catalog.Song{
			Title:  r.FormValue("title"),
			Artist: r.FormValue("artist"),
		})
		data := struct{ Entries []queue.Entry }{q.Entries()}
		if err := getTemplates().ExecuteTemplate(w, "host_queue.html", data); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	})))
	mux.Handle("POST /host/skip", middleware.AdminAuth(cfg.EnforceAdminAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q.MoveCurrentToBottom()
		data := struct{ Entries []queue.Entry }{q.Entries()}
		if err := getTemplates().ExecuteTemplate(w, "host_queue.html", data); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	})))
	mux.Handle("POST /signups/toggle", middleware.AdminAuth(cfg.EnforceAdminAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		open := q.ToggleSignups()
		fmt.Fprintf(w, `{"signups_open":%t}`, open)
	})))
	mux.HandleFunc("GET /catalog", handleCatalog)
	mux.Handle("GET /static/", staticHandler())
	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	slog.Info("ok")
}

func handleIndex(w http.ResponseWriter, r *http.Request, q *queue.Queue) {
	data := struct {
		Current     *queue.Entry
		Next        *queue.Entry
		SignupsOpen bool
	}{q.Current(), q.Next(), q.SignupsOpen}
	if err := getTemplates().ExecuteTemplate(w, "index.html", data); err != nil {
		slog.Error("template error", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func handleSignupPage(w http.ResponseWriter, r *http.Request) {
	if err := getTemplates().ExecuteTemplate(w, "signup.html", catalog.FullCatalog); err != nil {
		slog.Error("template error", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func handleSignup(w http.ResponseWriter, r *http.Request, q *queue.Queue) {
	name := r.FormValue("name")
	var songs []catalog.Song
	for _, s := range r.Form["song"] {
		var song catalog.Song
		if err := json.Unmarshal([]byte(s), &song); err != nil {
			slog.Error("failed to unmarshal song", "err", err)
			http.Error(w, "invalid song data", http.StatusBadRequest)
			return
		}
		songs = append(songs, song)
	}
	q.Add(name, songs)
	slog.Info("signup", "name", name, "songs", songs)
	fmt.Fprintf(w, `<p>You're on the list, %s! See you up there.</p>`, name)
}

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(catalog.JSONBytes); err != nil {
		slog.Error("write error", "err", err)
	}
}
