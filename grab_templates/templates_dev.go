//go:build !production

package grab_templates

import "html/template"

// GetTemplates re-parses templates from disk on every call so edits are
// reflected without restarting the server.
func GetTemplates() *template.Template {
	return template.Must(template.ParseGlob("templates/*"))
}
