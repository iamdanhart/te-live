//go:build !production

package router

import "net/http"

func staticHandler() http.Handler {
	return http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
}
