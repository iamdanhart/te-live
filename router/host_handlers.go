package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/iamdanhart/te-live/catalog"
	"github.com/iamdanhart/te-live/config"
	"github.com/iamdanhart/te-live/grab_templates"
	"github.com/iamdanhart/te-live/middleware"
	"github.com/iamdanhart/te-live/queue"
)

func registerHostRoutes(mux *http.ServeMux, cfg config.Props, q *queue.Queue) {
	auth := func(h http.HandlerFunc) http.Handler {
		return middleware.AdminAuth(cfg.EnforceAdminAuth, h)
	}

	mux.Handle("GET /host", auth(func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Entries   []queue.Entry
			Performed []queue.PerformedSong
		}{q.Entries(), q.Performed()}
		if err := grab_templates.GetTemplates().ExecuteTemplate(w, "host.html", data); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}))

	mux.Handle("GET /host/queue", auth(func(w http.ResponseWriter, r *http.Request) {
		data := struct{ Entries []queue.Entry }{q.Entries()}
		if err := grab_templates.GetTemplates().ExecuteTemplate(w, "host_queue.html", data); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}))

	mux.Handle("POST /host/performed", auth(func(w http.ResponseWriter, r *http.Request) {
		title, artist := r.FormValue("title"), r.FormValue("artist")
		q.MarkSongPerformed(title, artist)
		q.RecordPerformed(r.FormValue("singer"), catalog.Song{Title: title, Artist: artist})
		q.MoveCurrentToBottom()
		tmpl := grab_templates.GetTemplates()
		if err := tmpl.ExecuteTemplate(w, "host_performed.html", struct{ Performed []queue.PerformedSong }{q.Performed()}); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "host_queue_oob.html", struct{ Entries []queue.Entry }{q.Entries()}); err != nil {
			slog.Error("template error", "err", err)
		}
	}))

	mux.Handle("POST /host/add-song", auth(func(w http.ResponseWriter, r *http.Request) {
		q.AddSongToFirst(catalog.Song{
			Title:  r.FormValue("title"),
			Artist: r.FormValue("artist"),
		})
		data := struct{ Entries []queue.Entry }{q.Entries()}
		if err := grab_templates.GetTemplates().ExecuteTemplate(w, "host_queue.html", data); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}))

	mux.Handle("POST /host/skip", auth(func(w http.ResponseWriter, r *http.Request) {
		q.MoveCurrentToBottom()
		data := struct{ Entries []queue.Entry }{q.Entries()}
		if err := grab_templates.GetTemplates().ExecuteTemplate(w, "host_queue.html", data); err != nil {
			slog.Error("template error", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}))

	mux.Handle("POST /signups/toggle", auth(func(w http.ResponseWriter, r *http.Request) {
		open := q.ToggleSignups()
		fmt.Fprintf(w, `{"signups_open":%t}`, open)
	}))
}
