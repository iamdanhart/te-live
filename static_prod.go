//go:build production

package main

import (
	"embed"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

func staticHandler() http.Handler {
	return http.FileServerFS(staticFiles)
}
