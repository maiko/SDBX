package web

import "embed"

//go:embed static/css static/js
var staticFS embed.FS

//go:embed templates/layouts templates/pages
var templatesFS embed.FS
