package wizard

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Generator orchestrates the full project generation process.
type Generator struct {
	registry *TemplateRegistry
	executor *Executor
	tracker  *JobTracker
}

// NewGenerator creates a Generator with the given dependencies.
func NewGenerator(registry *TemplateRegistry, tracker *JobTracker) *Generator {
	return &Generator{
		registry: registry,
		executor: NewExecutor(5 * time.Minute),
		tracker:  tracker,
	}
}

// Generate runs the full project generation pipeline for a wizard request.
// It tracks progress through the job system and handles rollback on failure.
func (g *Generator) Generate(ctx context.Context, job *Job) error {
	req := job.Request
	projectDir := filepath.Join(expandHomePath(req.OutputDir), strings.TrimSpace(req.ProjectName))

	// Step 1: Validate
	job.StartStep("validate")
	result := Validate(req)
	if !result.IsValid() {
		err := fmt.Errorf("validation failed: %v", result.Errors)
		job.FailStep("validate", err)
		return err
	}
	job.CompleteStep("validate", "all checks passed")

	// Step 2: Create directory
	job.StartStep("create_directory")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		err = fmt.Errorf("creating project directory: %w", err)
		job.FailStep("create_directory", err)
		return err
	}
	job.CompleteStep("create_directory", projectDir)

	// Step 3: Generate project files (LLM or templates)
	job.StartStep("generate_files")

	// Try LLM generation if enabled
	usedLLM := false
	if req.AIEnabled && req.AIAPIKey != "" && req.AIModel != "" {
		if err := g.generateWithLLM(ctx, req, projectDir); err != nil {
			log.Printf("LLM generation failed, falling back to templates: %v", err)
			// Fall back to templates
		} else {
			usedLLM = true
		}
	}

	// Fall back to templates if LLM wasn't used
	if !usedLLM {
		if err := g.copyTemplates(req, projectDir); err != nil {
			job.FailStep("generate_files", err)
			g.rollback(projectDir, job)
			return err
		}
	}

	if usedLLM {
		job.CompleteStep("generate_files", "files generated using AI")
	} else {
		job.CompleteStep("generate_files", "files generated from templates")
	}

	// Step 4: Copy supporting docs
	job.StartStep("copy_docs")
	if err := g.copyDocs(req, projectDir); err != nil {
		job.FailStep("copy_docs", err)
		g.rollback(projectDir, job)
		return err
	}
	job.CompleteStep("copy_docs", "supporting docs copied")

	// Step 5: Generate CLAUDE.md (already handled by common template, mark done)
	job.StartStep("generate_claude_md")
	job.CompleteStep("generate_claude_md", "included in template output")

	// Step 6: Initialize git
	job.StartStep("init_git")
	if err := g.initGit(ctx, projectDir); err != nil {
		job.FailStep("init_git", err)
		// Don't rollback for git init failure — files are still useful
		log.Printf("Warning: git init failed for %s: %v", projectDir, err)
	} else {
		job.CompleteStep("init_git", "git repository initialized")
	}

	// Step 7: Install dependencies (best-effort, non-blocking)
	job.StartStep("install_deps")
	if err := g.installDeps(ctx, req, projectDir); err != nil {
		// Log but don't fail the job for dependency installation
		log.Printf("Warning: dependency installation failed for %s: %v", projectDir, err)
		job.CompleteStep("install_deps", fmt.Sprintf("skipped: %v", err))
	} else {
		job.CompleteStep("install_deps", "dependencies installed")
	}

	job.Complete(projectDir)
	return nil
}

// copyTemplates renders and writes all template files for the selected
// language/framework into the project directory.
func (g *Generator) copyTemplates(req WizardRequest, projectDir string) error {
	set, err := g.registry.Get(req.Language, req.Framework)
	if err != nil {
		return fmt.Errorf("loading template set: %w", err)
	}

	lang, _ := FindLanguage(req.Language)
	fw, _ := FindFramework(req.Language, req.Framework)

	data := TemplateData{
		ProjectName:        strings.TrimSpace(req.ProjectName),
		Language:           lang.Name,
		Framework:          fw.Name,
		Description:        req.Description,
		HasPRD:          strings.TrimSpace(req.DocPRD) != "",
		HasUIUX:         strings.TrimSpace(req.DocUIUX) != "",
		HasArchitecture: strings.TrimSpace(req.DocArchitecture) != "",
		HasOther:        strings.TrimSpace(req.DocOther) != "",
	}

	for _, tf := range set.Files {
		content, outPath, err := g.registry.RenderFile(tf, data)
		if err != nil {
			return fmt.Errorf("rendering %s: %w", tf.RelPath, err)
		}

		fullPath := filepath.Join(projectDir, outPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", outPath, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
	}

	return nil
}

// generateWithLLM generates project structure and files using the LLM
func (g *Generator) generateWithLLM(ctx context.Context, req WizardRequest, projectDir string) error {
	// Create LLM client
	client := NewLLMClient(req.AIProvider, req.AIAPIKey, req.AIModel, req.AIBaseURL)
	generator := NewLLMGenerator(client)

	// Prepare docs for LLM context
	docs := map[string]string{
		"prd":          req.DocPRD,
		"uiux":         req.DocUIUX,
		"architecture": req.DocArchitecture,
		"other":        req.DocOther,
	}

	// Generate directory structure
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	plan, err := generator.GenerateStructure(ctx, req.ProjectName, req.Language, req.Framework, req.Description, docs)
	if err != nil {
		return fmt.Errorf("structure generation failed: %w", err)
	}

	// Create directories
	for _, dir := range plan.Directories {
		dirPath := filepath.Join(projectDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Generate files
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	files, err := generator.GenerateFiles(ctx, req.ProjectName, req.Language, req.Framework, req.Description, plan, docs)
	if err != nil {
		return fmt.Errorf("file generation failed: %w", err)
	}

	// Write generated files
	for filePath, content := range files {
		fullPath := filepath.Join(projectDir, filePath)

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", filePath, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filePath, err)
		}
	}

	return nil
}

// copyDocs copies supporting documentation files into docs/supporting/.
// Supports both file paths and direct content pasted into textareas.
func (g *Generator) copyDocs(req WizardRequest, projectDir string) error {
	docsDir := filepath.Join(projectDir, "docs", "supporting")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("creating docs directory: %w", err)
	}

	docMap := map[string]string{
		"prd.md":          req.DocPRD,
		"uiux.md":         req.DocUIUX,
		"architecture.md": req.DocArchitecture,
		"other.md":        req.DocOther,
	}

	for destName, srcPath := range docMap {
		srcPath = strings.TrimSpace(srcPath)
		if srcPath == "" {
			continue
		}

		outputPath := filepath.Join(docsDir, destName)

		// Check if srcPath is a file path or direct content
		expanded := expandHomePath(srcPath)
		if _, err := os.Stat(expanded); err == nil {
			// File exists, copy it
			if err := copyFile(expanded, outputPath); err != nil {
				return fmt.Errorf("copying %s: %w", destName, err)
			}
		} else {
			// Path doesn't exist, treat as direct content and write it
			if err := os.WriteFile(outputPath, []byte(srcPath), 0644); err != nil {
				return fmt.Errorf("writing %s: %w", destName, err)
			}
		}
	}

	return nil
}

// initGit initializes a git repository in the project directory.
func (g *Generator) initGit(ctx context.Context, projectDir string) error {
	result := g.executor.Run(ctx, projectDir, "git", "init")
	if result.Err != nil {
		return fmt.Errorf("git init: %w (stderr: %s)", result.Err, result.Stderr)
	}

	result = g.executor.Run(ctx, projectDir, "git", "add", ".")
	if result.Err != nil {
		return fmt.Errorf("git add: %w (stderr: %s)", result.Err, result.Stderr)
	}

	result = g.executor.Run(ctx, projectDir, "git", "commit", "-m", "Initial project scaffold from ClawIDE wizard")
	if result.Err != nil {
		return fmt.Errorf("git commit: %w (stderr: %s)", result.Err, result.Stderr)
	}

	return nil
}

// installDeps runs the appropriate package manager for the language.
// This is best-effort and non-fatal.
func (g *Generator) installDeps(ctx context.Context, req WizardRequest, projectDir string) error {
	switch req.Language {
	case "python":
		// Check if uv is available, fall back to pip
		result := g.executor.RunWithTimeout(ctx, 2*time.Minute, projectDir, "uv", "pip", "compile", "requirements.in", "-o", "requirements.txt")
		if result.Err != nil {
			// Fall back to pip-compile
			result = g.executor.RunWithTimeout(ctx, 2*time.Minute, projectDir, "pip-compile", "requirements.in", "-o", "requirements.txt")
			if result.Err != nil {
				return fmt.Errorf("pip-compile: %w", result.Err)
			}
		}
	case "javascript":
		result := g.executor.RunWithTimeout(ctx, 3*time.Minute, projectDir, "npm", "install")
		if result.Err != nil {
			return fmt.Errorf("npm install: %w", result.Err)
		}
	case "go":
		result := g.executor.RunWithTimeout(ctx, 2*time.Minute, projectDir, "go", "mod", "tidy")
		if result.Err != nil {
			return fmt.Errorf("go mod tidy: %w", result.Err)
		}
	case "rust":
		// Just check that Cargo.toml exists and is valid
		result := g.executor.RunWithTimeout(ctx, 2*time.Minute, projectDir, "cargo", "check")
		if result.Err != nil {
			return fmt.Errorf("cargo check: %w", result.Err)
		}
	case "java":
		// Maven or Gradle — check which build file exists
		if _, err := os.Stat(filepath.Join(projectDir, "pom.xml")); err == nil {
			result := g.executor.RunWithTimeout(ctx, 5*time.Minute, projectDir, "mvn", "dependency:resolve")
			if result.Err != nil {
				return fmt.Errorf("mvn: %w", result.Err)
			}
		}
	case "ruby":
		result := g.executor.RunWithTimeout(ctx, 2*time.Minute, projectDir, "bundle", "install")
		if result.Err != nil {
			return fmt.Errorf("bundle install: %w", result.Err)
		}
	case "php":
		result := g.executor.RunWithTimeout(ctx, 2*time.Minute, projectDir, "composer", "install")
		if result.Err != nil {
			return fmt.Errorf("composer install: %w", result.Err)
		}
	case "csharp":
		result := g.executor.RunWithTimeout(ctx, 2*time.Minute, projectDir, "dotnet", "restore")
		if result.Err != nil {
			return fmt.Errorf("dotnet restore: %w", result.Err)
		}
	}
	return nil
}

// rollback removes the project directory after a failure.
func (g *Generator) rollback(projectDir string, job *Job) {
	log.Printf("Rolling back project directory: %s", projectDir)
	if err := os.RemoveAll(projectDir); err != nil {
		log.Printf("Warning: rollback failed: %v", err)
	}
	job.MarkRolledBack()
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Close()
}
