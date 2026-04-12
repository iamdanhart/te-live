//go:build production

package main

import (
	"embed"
	"html/template"
)

//go:embed templates/*
var templateFiles embed.FS

var parsedTemplates = template.Must(template.ParseFS(templateFiles, "templates/*"))

// getTemplates returns the pre-compiled embedded templates.
func getTemplates() *template.Template {
	return parsedTemplates
}