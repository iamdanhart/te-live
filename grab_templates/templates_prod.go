//go:build production

package grab_templates

import (
	"embed"
	"html/template"
)

//go:embed templates/*
var templateFiles embed.FS

var parsedTemplates = template.Must(template.ParseFS(templateFiles, "templates/*"))

// GetTemplates returns the pre-compiled embedded templates.
func GetTemplates() *template.Template {
	return parsedTemplates
}
