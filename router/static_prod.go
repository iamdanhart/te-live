//go:build production

package router

import (
	"embed"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

func staticHandler() http.Handler {
	return http.FileServerFS(staticFiles)
}
