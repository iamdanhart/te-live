//go:build !production

package main

import "html/template"

// getTemplates re-parses templates from disk on every call so edits are
// reflected without restarting the server.
func getTemplates() *template.Template {
	return template.Must(template.ParseGlob("templates/*"))
}