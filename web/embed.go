package web

import "embed"

//go:embed static templates
var EmbeddedFS embed.FS

// StaticFS provides the static directory for file serving
var StaticFS = EmbeddedFS

// TemplateFS provides the templates directory for template rendering
var TemplateFS = EmbeddedFS
