package wizard

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateData holds the variables available to project scaffold templates.
type TemplateData struct {
	ProjectName string
	Language    string
	Framework   string
	Description string

	// Has* flags indicate whether supporting docs were provided,
	// used for conditional sections in CLAUDE.md templates.
	HasPRD          bool
	HasUIUX         bool
	HasArchitecture bool
	HasOther        bool
}

// TemplateFile represents a single file within a project scaffold template.
type TemplateFile struct {
	// RelPath is the relative path within the generated project
	// (may contain Go template expressions like {{.ProjectName}}).
	RelPath string

	// Content is the raw template content to be rendered.
	Content string

	// IsTemplate indicates whether the content should be processed
	// through Go's text/template engine. If false, content is written as-is.
	IsTemplate bool
}

// TemplateSet is a collection of template files for a language/framework combo.
type TemplateSet struct {
	Language  string
	Framework string
	Files     []TemplateFile
}

// TemplateRegistry holds all loaded template sets indexed by language/framework.
type TemplateRegistry struct {
	sets    map[string]*TemplateSet // key: "language/framework"
	common  *TemplateSet           // shared templates (CLAUDE.md, .gitignore, etc.)
	funcMap template.FuncMap
}

// NewTemplateRegistry creates a registry and loads templates from the given
// filesystem. The filesystem should have the structure:
//
//	templates/
//	  common/          <- shared across all projects
//	  python/
//	    django/        <- language/framework specific
//	    fastapi/
//	  javascript/
//	    ...
func NewTemplateRegistry(fsys fs.FS) (*TemplateRegistry, error) {
	reg := &TemplateRegistry{
		sets: make(map[string]*TemplateSet),
		funcMap: template.FuncMap{
			"lower":    strings.ToLower,
			"upper":    strings.ToUpper,
			"contains": strings.Contains,
			"join":     strings.Join,
		},
	}

	if err := reg.loadFromFS(fsys); err != nil {
		return nil, fmt.Errorf("loading templates: %w", err)
	}

	return reg, nil
}

// loadFromFS walks the template filesystem and loads all template sets.
func (r *TemplateRegistry) loadFromFS(fsys fs.FS) error {
	// Load common templates
	common, err := r.loadSet(fsys, "templates/common", "", "")
	if err != nil {
		return fmt.Errorf("loading common templates: %w", err)
	}
	r.common = common

	// Walk language directories
	entries, err := fs.ReadDir(fsys, "templates")
	if err != nil {
		return fmt.Errorf("reading templates directory: %w", err)
	}

	for _, langEntry := range entries {
		if !langEntry.IsDir() || langEntry.Name() == "common" {
			continue
		}

		langID := langEntry.Name()

		// Walk framework directories within each language
		fwEntries, err := fs.ReadDir(fsys, filepath.Join("templates", langID))
		if err != nil {
			return fmt.Errorf("reading %s templates: %w", langID, err)
		}

		for _, fwEntry := range fwEntries {
			if !fwEntry.IsDir() {
				continue
			}
			fwID := fwEntry.Name()
			dir := filepath.Join("templates", langID, fwID)

			set, err := r.loadSet(fsys, dir, langID, fwID)
			if err != nil {
				return fmt.Errorf("loading template %s/%s: %w", langID, fwID, err)
			}

			key := langID + "/" + fwID
			r.sets[key] = set
		}
	}

	return nil
}

// loadSet reads all files within a directory and creates a TemplateSet.
func (r *TemplateRegistry) loadSet(fsys fs.FS, dir, language, framework string) (*TemplateSet, error) {
	set := &TemplateSet{
		Language:  language,
		Framework: framework,
	}

	err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		// Determine relative path within the generated project.
		// Strip the "templates/{language}/{framework}/" or "templates/common/" prefix.
		relPath := strings.TrimPrefix(path, dir+"/")

		// Files ending in .tmpl are Go templates — strip the suffix for the output path.
		isTemplate := strings.HasSuffix(relPath, ".tmpl")
		if isTemplate {
			relPath = strings.TrimSuffix(relPath, ".tmpl")
		}

		set.Files = append(set.Files, TemplateFile{
			RelPath:    relPath,
			Content:    string(content),
			IsTemplate: isTemplate,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return set, nil
}

// Get returns the merged template set for a given language/framework.
// Common templates are included first, then framework-specific templates
// override or extend them.
func (r *TemplateRegistry) Get(language, framework string) (*TemplateSet, error) {
	key := language + "/" + framework
	fwSet, ok := r.sets[key]
	if !ok {
		return nil, fmt.Errorf("no templates found for %s", key)
	}

	// Merge common + framework-specific. Framework files override common files
	// at the same relative path.
	merged := &TemplateSet{
		Language:  language,
		Framework: framework,
	}

	// Index framework files by path for override detection
	fwIndex := make(map[string]struct{}, len(fwSet.Files))
	for _, f := range fwSet.Files {
		fwIndex[f.RelPath] = struct{}{}
	}

	// Add common files that aren't overridden
	if r.common != nil {
		for _, f := range r.common.Files {
			if _, overridden := fwIndex[f.RelPath]; !overridden {
				merged.Files = append(merged.Files, f)
			}
		}
	}

	// Add all framework files
	merged.Files = append(merged.Files, fwSet.Files...)

	return merged, nil
}

// RenderFile processes a single template file with the given data.
// Returns the rendered content and the resolved output path.
func (r *TemplateRegistry) RenderFile(tf TemplateFile, data TemplateData) (content string, outPath string, err error) {
	// Resolve the output path (may contain template expressions)
	pathTmpl, err := template.New("path").Funcs(r.funcMap).Parse(tf.RelPath)
	if err != nil {
		return "", "", fmt.Errorf("parsing path template %q: %w", tf.RelPath, err)
	}
	var pathBuf strings.Builder
	if err := pathTmpl.Execute(&pathBuf, data); err != nil {
		return "", "", fmt.Errorf("rendering path %q: %w", tf.RelPath, err)
	}
	outPath = pathBuf.String()

	// Render content if it's a template
	if tf.IsTemplate {
		contentTmpl, err := template.New("content").Funcs(r.funcMap).Parse(tf.Content)
		if err != nil {
			return "", "", fmt.Errorf("parsing content template for %q: %w", tf.RelPath, err)
		}
		var contentBuf strings.Builder
		if err := contentTmpl.Execute(&contentBuf, data); err != nil {
			return "", "", fmt.Errorf("rendering content for %q: %w", tf.RelPath, err)
		}
		content = contentBuf.String()
	} else {
		content = tf.Content
	}

	return content, outPath, nil
}

// HasTemplateSet returns whether templates exist for a language/framework combo.
func (r *TemplateRegistry) HasTemplateSet(language, framework string) bool {
	key := language + "/" + framework
	_, ok := r.sets[key]
	return ok
}

// ListAvailable returns a list of all registered language/framework keys.
func (r *TemplateRegistry) ListAvailable() []string {
	keys := make([]string, 0, len(r.sets))
	for k := range r.sets {
		keys = append(keys, k)
	}
	return keys
}
