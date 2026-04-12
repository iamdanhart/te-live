package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"te-live/catalog"
)

//go:embed templates/*
var templateFiles embed.FS

var templates = template.Must(template.ParseFS(templateFiles, "templates/*"))

func getTemplates() *template.Template {
	if os.Getenv("ENV") == "production" {
		return templates
	}
	return template.Must(template.ParseGlob("templates/*"))
}

func newRouter(rl *rateLimiter) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("GET /catalog", handleCatalog)
	mux.Handle("POST /signup", rl.limit(http.HandlerFunc(handleSignup)))
	mux.Handle("GET /static/", staticHandler())
	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	slog.Info("ok")
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := getTemplates().ExecuteTemplate(w, "index.html", catalog.FullCatalog); err != nil {
		slog.Error("template error", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func handleSignup(w http.ResponseWriter, r *http.Request) {
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
	slog.Info("signup", "name", name, "songs", songs)
	if _, err := fmt.Fprintf(w, "<p>Thanks, %s! You've been added to the queue.</p>", name); err != nil {
		slog.Error("write error", "err", err)
	}
}

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(catalog.JSONBytes); err != nil {
		slog.Error("write error", "err", err)
	}
}
