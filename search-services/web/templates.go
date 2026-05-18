package main

import (
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templatesFS embed.FS

func parseTemplates() (*template.Template, error) {
	return template.New("layout").Funcs(template.FuncMap{
		"addSpace": func(n int) string {
			// lightweight thousands separator
			s := fmt.Sprintf("%d", n)
			for i := len(s) - 3; i > 0; i -= 3 {
				s = s[:i] + " " + s[i:]
			}
			return s
		},
	}).ParseFS(templatesFS, "templates/*.html")
}
