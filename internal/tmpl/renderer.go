package tmpl

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Renderer struct {
	templates map[string]*template.Template
	fs        fs.FS
}

func New(fsys fs.FS) (*Renderer, error) {
	r := &Renderer{
		templates: make(map[string]*template.Template),
		fs:        fsys,
	}
	if err := r.parseTemplates(); err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}
	return r, nil
}

func (r *Renderer) parseTemplates() error {
	titleCaser := cases.Title(language.English)

	funcMap := template.FuncMap{
		"lower":    strings.ToLower,
		"upper":    strings.ToUpper,
		"title":    titleCaser.String,
		"contains": strings.Contains,
		"join":     strings.Join,
		"dict":     dictFunc,
	}

	base, err := template.New("").Funcs(funcMap).ParseFS(r.fs, "templates/base.html")
	if err != nil {
		return fmt.Errorf("parsing base template: %w", err)
	}


	componentFiles, err := fs.Glob(r.fs, "templates/components/*.html")
	if err != nil {
		return fmt.Errorf("globbing components: %w", err)
	}

	partialFiles, err := fs.Glob(r.fs, "templates/partials/*.html")
	if err != nil {
		return fmt.Errorf("globbing partials: %w", err)
	}

	pageFiles, err := fs.Glob(r.fs, "templates/pages/*.html")
	if err != nil {
		return fmt.Errorf("globbing pages: %w", err)
	}

	for _, page := range pageFiles {
		name := strings.TrimSuffix(filepath.Base(page), ".html")

		t, err := base.Clone()
		if err != nil {
			return fmt.Errorf("cloning base for %s: %w", name, err)
		}

		// Build list of files to parse: page + components
		files := []string{page}
		files = append(files, componentFiles...)

		t, err = t.ParseFS(r.fs, files...)
		if err != nil {
			return fmt.Errorf("parsing page %s: %w", name, err)
		}

		r.templates[name] = t
		log.Printf("Registered template: %s", name)
	}

	// Register partials standalone for htmx requests
	for _, partial := range partialFiles {
		name := "partial:" + strings.TrimSuffix(filepath.Base(partial), ".html")

		files := []string{partial}
		files = append(files, componentFiles...)

		t, err := template.New("").Funcs(funcMap).ParseFS(r.fs, files...)
		if err != nil {
			return fmt.Errorf("parsing partial %s: %w", name, err)
		}

		r.templates[name] = t
		log.Printf("Registered partial: %s", name)
	}

	return nil
}

func (r *Renderer) Render(w io.Writer, name string, data any) error {
	t, ok := r.templates[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}
	return t.ExecuteTemplate(w, "base.html", data)
}

func (r *Renderer) RenderHTMX(w http.ResponseWriter, req *http.Request, page string, partial string, data any) error {
	isHTMX := req.Header.Get("HX-Request") == "true"

	if isHTMX && partial != "" {
		partialName := "partial:" + partial
		if t, ok := r.templates[partialName]; ok {
			// For partials, execute the partial's own root template
			return t.Execute(w, data)
		}
	}

	return r.Render(w, page, data)
}

func dictFunc(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict requires even number of arguments")
	}
	m := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict key must be string, got %T", pairs[i])
		}
		m[key] = pairs[i+1]
	}
	return m, nil
}
