package wizard

import "embed"

// TemplatesFS embeds the project scaffold templates used by the wizard.
// Directory structure:
//
//	templates/
//	  common/              shared files (CLAUDE.md, .gitignore, README.md)
//	  python/django/       Python+Django specific files
//	  python/fastapi/      Python+FastAPI specific files
//	  ...
//
//go:embed templates
var TemplatesFS embed.FS
