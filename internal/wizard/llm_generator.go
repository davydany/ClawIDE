package wizard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// DirectoryPlan represents the structure to be created
type DirectoryPlan struct {
	Directories []string
	Files       []FileSpec
}

// FileSpec represents a file to be created
type FileSpec struct {
	Path     string
	Type     string // "config", "source", "test", "doc"
	Priority string // "essential", "recommended", "optional"
	Content  string
}

// LLMGenerator generates project structure and files using an LLM
type LLMGenerator struct {
	client *LLMClient
}

// NewLLMGenerator creates a new LLM-based generator
func NewLLMGenerator(client *LLMClient) *LLMGenerator {
	return &LLMGenerator{
		client: client,
	}
}

// GenerateStructure generates a directory structure based on project requirements
func (g *LLMGenerator) GenerateStructure(ctx context.Context, projectName, language, framework, description string, docs map[string]string) (*DirectoryPlan, error) {
	systemPrompt := fmt.Sprintf(`You are an expert software architect. Generate a project directory structure for a %s %s project.
Return ONLY a valid JSON object (no markdown, no explanation) with exactly this structure:
{
  "directories": ["/path/to/dir1", "/path/to/dir2", ...],
  "files": [
    {"path": "file.ext", "type": "config|source|test|doc", "priority": "essential|recommended|optional"}
  ]
}
Rules:
- directories must start with / (e.g., "/src")
- Use language-appropriate structure (e.g., src/ for Rust, lib/ for Python)
- Include test directories appropriate for the language
- Don't include structure.yaml or template.yaml
- Prioritize essential directories: src, tests/test
- Optional directories: examples, benchmarks, scripts
- Include doc directory if project has documentation
- Min 5 dirs, max 15 dirs for a good structure`, language, framework)

	userPrompt := g.buildStructurePrompt(projectName, language, framework, description, docs)

	resp, err := g.client.Generate(ctx, &LLMRequest{
		SystemRole:  systemPrompt,
		Prompt:      userPrompt,
		Temperature: 0.3, // Lower temperature for consistent structure
		MaxTokens:   1024,
	})

	if err != nil {
		return nil, fmt.Errorf("LLM structure generation failed: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("LLM error: %s", resp.Error)
	}

	// Parse JSON response
	plan := &DirectoryPlan{}

	// Clean up response (remove markdown code blocks if present)
	content := resp.Content
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json\n")
	content = strings.TrimPrefix(content, "```\n")
	content = strings.TrimSuffix(content, "\n```")
	content = strings.TrimSuffix(content, "```")

	if err := json.Unmarshal([]byte(content), plan); err != nil {
		log.Printf("Failed to parse LLM structure response: %s\nError: %v", content, err)
		return nil, fmt.Errorf("invalid structure JSON from LLM: %w", err)
	}

	// Filter out template metadata files
	filteredFiles := []FileSpec{}
	for _, f := range plan.Files {
		if !strings.Contains(f.Path, "structure.yaml") && !strings.Contains(f.Path, "template.yaml") {
			filteredFiles = append(filteredFiles, f)
		}
	}
	plan.Files = filteredFiles

	return plan, nil
}

// GenerateFiles generates the content for essential files
func (g *LLMGenerator) GenerateFiles(ctx context.Context, projectName, language, framework, description string, plan *DirectoryPlan, docs map[string]string) (map[string]string, error) {
	files := make(map[string]string)

	// Filter to essential files only (avoid generating too many files)
	essentialFiles := []FileSpec{}
	for _, f := range plan.Files {
		if f.Priority == "essential" || f.Priority == "" {
			essentialFiles = append(essentialFiles, f)
		}
		// Limit to reasonable number
		if len(essentialFiles) >= 8 {
			break
		}
	}

	systemPrompt := fmt.Sprintf(`You are an expert %s developer. Generate starter code for project files.
For each file, return valid, working code appropriate for the file extension.
Follow %s best practices and conventions.`, language, framework)

	for i, fileSpec := range essentialFiles {
		if i >= 5 { // Limit concurrent generation to 5 files
			break
		}

		userPrompt := g.buildFilePrompt(projectName, language, framework, description, fileSpec, docs)

		resp, err := g.client.Generate(ctx, &LLMRequest{
			SystemRole:  systemPrompt,
			Prompt:      userPrompt,
			Temperature: 0.5,
			MaxTokens:   2048,
		})

		if err != nil || resp.Error != "" {
			log.Printf("Warning: failed to generate %s: %v", fileSpec.Path, err)
			// Continue with other files instead of failing completely
			continue
		}

		files[fileSpec.Path] = resp.Content
	}

	// Generate basic files that don't need LLM
	files = g.addBasicFiles(projectName, language, framework, files, docs)

	return files, nil
}

// buildStructurePrompt constructs a prompt for directory structure generation
func (g *LLMGenerator) buildStructurePrompt(projectName, language, framework, description string, docs map[string]string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Project Name: %s\n", projectName))
	sb.WriteString(fmt.Sprintf("Language: %s\n", language))
	sb.WriteString(fmt.Sprintf("Framework: %s\n", framework))
	sb.WriteString(fmt.Sprintf("Description: %s\n\n", description))

	if len(docs) > 0 {
		sb.WriteString("Supporting Documentation:\n")
		for docType, content := range docs {
			if content != "" {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", docType, truncate(content, 200)))
			}
		}
	}

	sb.WriteString("\nGenerate a suitable directory structure for this project.")
	return sb.String()
}

// buildFilePrompt constructs a prompt for file generation
func (g *LLMGenerator) buildFilePrompt(projectName, language, framework, description string, file FileSpec, docs map[string]string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Project: %s (%s/%s)\n", projectName, language, framework))
	sb.WriteString(fmt.Sprintf("File: %s (type: %s)\n\n", file.Path, file.Type))
	sb.WriteString(fmt.Sprintf("Description: %s\n\n", description))

	if docContent, ok := docs["prd"]; ok && docContent != "" {
		sb.WriteString(fmt.Sprintf("PRD: %s\n\n", truncate(docContent, 300)))
	}

	sb.WriteString(fmt.Sprintf("Generate the content for %s following %s best practices.\n", file.Path, language))
	sb.WriteString("Return only the file content, no explanations.\n")

	return sb.String()
}

// addBasicFiles adds template-based files that don't need LLM generation
func (g *LLMGenerator) addBasicFiles(projectName, language, framework string, files map[string]string, docs map[string]string) map[string]string {
	// Add .gitignore
	gitignore := g.getGitignore(language)
	files[".gitignore"] = gitignore

	// Add README.md
	readme := fmt.Sprintf(`# %s

%s project using %s

## Setup

See documentation in /docs/supporting/ for more details.

## Development

To get started with development, follow the setup instructions above.

`, projectName, language, framework)

	files["README.md"] = readme

	// Add CLAUDE.md if docs are provided
	if len(docs) > 0 {
		claude := g.generateCLAUDEMD(projectName, docs)
		files["CLAUDE.md"] = claude
	}

	return files
}

// generateCLAUDEMD generates the CLAUDE.md file with references to supporting docs
func (g *LLMGenerator) generateCLAUDEMD(projectName string, docs map[string]string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", projectName))
	sb.WriteString("## Project Overview\n\n")
	sb.WriteString("This project was created using ClawIDE's AI-powered project wizard.\n\n")

	hasDocs := false
	for _, content := range docs {
		if content != "" {
			hasDocs = true
			break
		}
	}

	if hasDocs {
		sb.WriteString("## Supporting Documentation\n\n")
		sb.WriteString("The following documents provide context and specifications:\n\n")

		if content, ok := docs["prd"]; ok && content != "" {
			sb.WriteString("- [Product Requirements](docs/supporting/prd.md) - Feature specifications and requirements\n")
		}
		if content, ok := docs["uiux"]; ok && content != "" {
			sb.WriteString("- [UI/UX Design](docs/supporting/uiux.md) - Design specifications and mockups\n")
		}
		if content, ok := docs["architecture"]; ok && content != "" {
			sb.WriteString("- [System Architecture](docs/supporting/architecture.md) - Technical architecture and design decisions\n")
		}
		if content, ok := docs["other"]; ok && content != "" {
			sb.WriteString("- [Additional Documentation](docs/supporting/other.md) - Other relevant documentation\n")
		}

		sb.WriteString("\n**Review these documents before making changes to ensure alignment with project goals and design decisions.**\n\n")
	}

	sb.WriteString("## Getting Started\n\n")
	sb.WriteString("See README.md for development setup instructions.\n\n")
	sb.WriteString("## Development Notes\n\n")
	sb.WriteString("This project structure was auto-generated. Feel free to reorganize as needed for your workflow.\n")

	return sb.String()
}

// getGitignore returns a language-appropriate .gitignore
func (g *LLMGenerator) getGitignore(language string) string {
	ignoreMap := map[string]string{
		"python": `# Python
__pycache__/
*.py[cod]
*$py.class
*.so
.Python
build/
develop-eggs/
dist/
downloads/
eggs/
.eggs/
lib/
lib64/
parts/
sdist/
var/
wheels/
*.egg-info/
.installed.cfg
*.egg
.venv
venv/
ENV/
env/
.vscode
.idea
*.swp
*.swo
.DS_Store
`,
		"javascript": `# Node
node_modules/
npm-debug.log
yarn-error.log
.npm
dist/
build/
*.tsbuildinfo
.next
.env.local
.env.*.local
.vscode
.idea
*.swp
*.swo
.DS_Store
`,
		"go": `# Go
bin/
obj/
pkg/
*.o
*.a
*.so
.DS_Store
*.out
vendor/
.idea
.vscode
*.swp
*.swo
`,
		"rust": `# Rust
/target/
Cargo.lock
**/*.rs.bk
.DS_Store
.idea
.vscode
*.swp
*.swo
`,
		"java": `# Java
*.class
*.jar
*.war
target/
.classpath
.project
.settings
.idea
*.iml
.DS_Store
.vscode
*.swp
*.swo
`,
	}

	if ignore, ok := ignoreMap[language]; ok {
		return ignore
	}

	// Default gitignore
	return `# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Dependencies
node_modules/
vendor/
.venv
venv/

# Build outputs
dist/
build/
*.out

# Logs
*.log
`
}

// truncate limits string to max characters
func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
