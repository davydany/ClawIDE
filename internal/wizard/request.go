package wizard

// WizardRequest contains all the data submitted by the project wizard form.
type WizardRequest struct {
	// Required fields
	ProjectName string `json:"project_name"`
	Language    string `json:"language"`
	Framework   string `json:"framework"`
	OutputDir   string `json:"output_dir"`

	// Optional fields
	Description string `json:"description"`

	// Supporting docs (PRD, UI/UX, Architecture, Other)
	// These are file paths to uploaded documents that will be
	// copied into the generated project's docs/supporting/ directory.
	DocPRD          string `json:"doc_prd"`
	DocUIUX         string `json:"doc_uiux"`
	DocArchitecture string `json:"doc_architecture"`
	DocOther        string `json:"doc_other"`
}
