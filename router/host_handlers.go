package router

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/iamdanhart/te-live/config"
	"github.com/iamdanhart/te-live/grab_templates"
	"github.com/iamdanhart/te-live/middleware"
	"github.com/iamdanhart/te-live/queue"
)

func renderQueue(w http.ResponseWriter, r *http.Request, q queue.Queue) {
	var buf bytes.Buffer
	if err := grab_templates.GetTemplates().ExecuteTemplate(&buf, "host_queue.html", q.Entries(r.Context())); err != nil {
		slog.Error("template error", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := buf.WriteTo(w); err != nil {
		slog.Error("failed to write queue response", "err", err)
	}
}

func registerHostRoutes(mux *http.ServeMux, cfg config.Props, q queue.Queue, rl middleware.Limiter, csrf func(http.Handler) http.Handler) {
	auth := func(h http.HandlerFunc) http.Handler {
		return middleware.AdminAuth(cfg.EnforceAdminAuth, q.AuthenticateHost, h)
	}
	authPost := func(h http.HandlerFunc) http.Handler {
		return csrf(withHostPostMiddleware(rl, auth(h)))
	}

	mux.Handle("GET /host", auth(func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Entries     []queue.Entry
			Performed   []queue.PerformedSong
			SignupsOpen bool
		}{q.Entries(r.Context()), q.Performed(r.Context()), q.SignupsOpen(r.Context())}
		var buf bytes.Buffer
		if err := grab_templates.GetTemplates().ExecuteTemplate(&buf, "host.html", data); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if _, err := buf.WriteTo(w); err != nil {
			slog.Error("failed to write host response", "err", err)
		}
	}))

	mux.Handle("GET /host/queue", auth(func(w http.ResponseWriter, r *http.Request) {
		renderQueue(w, r, q)
	}))

	mux.Handle("POST /host/performed", authPost(func(w http.ResponseWriter, r *http.Request) {
		songID, err := strconv.Atoi(r.FormValue("song_id"))
		if err != nil {
			http.Error(w, "invalid song_id", http.StatusBadRequest)
			return
		}
		if err := q.CompleteCurrentSong(r.Context(), r.FormValue("singer"), songID); err != nil {
			http.Error(w, "failed to complete song", http.StatusInternalServerError)
			return
		}
		if err := q.MoveCurrentToBottom(r.Context()); err != nil {
			http.Error(w, "failed to move entry", http.StatusInternalServerError)
			return
		}
		tmpl := grab_templates.GetTemplates()
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "host_performed.html", q.Performed(r.Context())); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(&buf, "host_queue_oob.html", q.Entries(r.Context())); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if _, err := buf.WriteTo(w); err != nil {
			slog.Error("failed to write performed response", "err", err)
		}
	}))

	mux.Handle("POST /host/add-song", authPost(func(w http.ResponseWriter, r *http.Request) {
		songID, err := strconv.Atoi(r.FormValue("song_id"))
		if err != nil {
			http.Error(w, "invalid song_id", http.StatusBadRequest)
			return
		}
		if err := q.AddSongToFirst(r.Context(), songID); err != nil {
			if errors.Is(err, queue.ErrInvalidSongID) {
				http.Error(w, "invalid song id", http.StatusBadRequest)
				return
			}
			http.Error(w, "failed to add song", http.StatusInternalServerError)
			return
		}
		renderQueue(w, r, q)
	}))

	mux.Handle("POST /host/remove", authPost(func(w http.ResponseWriter, r *http.Request) {
		if err := q.RemoveCurrent(r.Context()); err != nil {
			http.Error(w, "failed to remove entry", http.StatusInternalServerError)
			return
		}
		renderQueue(w, r, q)
	}))

	mux.Handle("POST /host/skip", authPost(func(w http.ResponseWriter, r *http.Request) {
		if err := q.MoveCurrentToBottom(r.Context()); err != nil {
			http.Error(w, "failed to skip entry", http.StatusInternalServerError)
			return
		}
		renderQueue(w, r, q)
	}))

	mux.Handle("POST /host/move", authPost(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.FormValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		afterID, err := strconv.Atoi(r.FormValue("after_id"))
		if err != nil {
			http.Error(w, "invalid after_id", http.StatusBadRequest)
			return
		}
		if err := q.MoveEntry(r.Context(), id, afterID); err != nil {
			http.Error(w, "failed to move entry", http.StatusInternalServerError)
			return
		}
		renderQueue(w, r, q)
	}))

	mux.Handle("POST /signups/toggle", authPost(func(w http.ResponseWriter, r *http.Request) {
		open, err := q.ToggleSignups(r.Context())
		if err != nil {
			http.Error(w, "failed to toggle signups", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"signups_open":%t}`, open)
	}))
}

func withHostPostMiddleware(l middleware.Limiter, h http.Handler) http.Handler {
	return l.Limit(h)
}
