package wizard

// WizardRequest contains all the data submitted by the project wizard form.
type WizardRequest struct {
	// Required fields
	ProjectName string `json:"project_name"`
	Language    string `json:"language"`
	Framework   string `json:"framework"`
	OutputDir   string `json:"output_dir"`

	// Optional fields
	Description  string `json:"description"`
	EmptyProject bool   `json:"empty_project"` // If true, skip language/framework/template generation

	// Supporting docs (PRD, UI/UX, Architecture, Other)
	// These are file paths to uploaded documents that will be
	// copied into the generated project's docs/supporting/ directory.
	DocPRD          string `json:"doc_prd"`
	DocUIUX         string `json:"doc_uiux"`
	DocArchitecture string `json:"doc_architecture"`
	DocOther        string `json:"doc_other"`

	// Git Clone Configuration (optional)
	// Used when cloning an existing repository instead of generating a new project
	CloneProject    bool   `json:"clone_project"`     // If true, clone from GitCloneURL
	GitCloneURL     string `json:"git_clone_url"`     // SSH or HTTPS git URL
	GitCloneBranch  string `json:"git_clone_branch"`  // Optional: branch to clone
	GitCloneDepth   int    `json:"git_clone_depth"`   // 0 = full clone, >0 = shallow
	GitCloneDirName string `json:"git_clone_dir_name"` // Directory name (derived from URL if empty)

	// AI Configuration (optional)
	// Controls whether AI code generation is enabled and which provider/model to use
	AIEnabled    bool       `json:"ai_enabled"`
	AIProvider   AIProvider `json:"ai_provider"`
	AIModel      string     `json:"ai_model"` // Model ID (e.g., "claude-opus", "gpt-4o")
	AIAPIKey     string     `json:"ai_api_key"`     // Sensitive: API key for the provider
	AIBaseURL    string     `json:"ai_base_url"`    // Optional: For self-hosted providers like Ollama
	AITemperature float32   `json:"ai_temperature"` // 0.0-1.0, controls randomness
}
