package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/iamdanhart/te-live/catalog"
	"github.com/iamdanhart/te-live/config"
	"github.com/iamdanhart/te-live/grab_templates"
	"github.com/iamdanhart/te-live/middleware"
	"github.com/iamdanhart/te-live/queue"
)

func registerHostRoutes(mux *http.ServeMux, cfg config.Props, q queue.Queue) {
	auth := func(h http.HandlerFunc) http.Handler {
		return middleware.AdminAuth(cfg.EnforceAdminAuth, h)
	}

	mux.Handle("GET /host", auth(func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Entries     []queue.Entry
			Performed   []queue.PerformedSong
			SignupsOpen bool
		}{q.Entries(), q.Performed(), q.SignupsOpen()}
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
		song := catalog.Song{Title: r.FormValue("title"), Artist: r.FormValue("artist")}
		q.CompleteCurrentSong(r.FormValue("singer"), song)
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

	mux.Handle("POST /host/remove", auth(func(w http.ResponseWriter, r *http.Request) {
		q.RemoveCurrent()
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

	mux.Handle("POST /host/move", auth(func(w http.ResponseWriter, r *http.Request) {
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
		q.MoveEntry(id, afterID)
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
