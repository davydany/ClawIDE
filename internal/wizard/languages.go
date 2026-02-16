package wizard

// Language represents a programming language with its available frameworks.
type Language struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Frameworks []Framework `json:"frameworks"`
}

// Framework represents a framework within a language ecosystem.
type Framework struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SupportedLanguages returns the full list of languages and frameworks
// supported by the wizard. This is the single source of truth for
// language/framework combinations.
func SupportedLanguages() []Language {
	return []Language{
		{
			ID:   "python",
			Name: "Python",
			Frameworks: []Framework{
				{ID: "django", Name: "Django", Description: "Full-stack web framework with ORM, admin, and batteries included"},
				{ID: "fastapi", Name: "FastAPI", Description: "Modern async API framework with automatic OpenAPI docs"},
				{ID: "flask", Name: "Flask", Description: "Lightweight micro-framework for web applications"},
			},
		},
		{
			ID:   "javascript",
			Name: "JavaScript",
			Frameworks: []Framework{
				{ID: "node-express", Name: "Node.js + Express", Description: "Minimal Node.js web framework for APIs and servers"},
				{ID: "react", Name: "React (Vite)", Description: "Component-based UI library with Vite build tooling"},
				{ID: "nextjs", Name: "Next.js", Description: "React framework with SSR, routing, and API routes"},
				{ID: "vue", Name: "Vue.js (Vite)", Description: "Progressive JavaScript framework with Vite"},
			},
		},
		{
			ID:   "go",
			Name: "Go",
			Frameworks: []Framework{
				{ID: "gin", Name: "Gin", Description: "High-performance HTTP web framework"},
				{ID: "echo", Name: "Echo", Description: "Minimalist Go web framework"},
				{ID: "chi", Name: "Chi", Description: "Lightweight, composable router for Go HTTP services"},
			},
		},
		{
			ID:   "java",
			Name: "Java",
			Frameworks: []Framework{
				{ID: "spring-boot", Name: "Spring Boot", Description: "Opinionated Java framework for production-grade apps"},
				{ID: "quarkus", Name: "Quarkus", Description: "Kubernetes-native Java framework with fast startup"},
			},
		},
		{
			ID:   "csharp",
			Name: "C#",
			Frameworks: []Framework{
				{ID: "aspnet", Name: "ASP.NET Core", Description: "Cross-platform web framework for .NET"},
				{ID: "minimal-api", Name: "Minimal API", Description: "Lightweight ASP.NET Core with minimal boilerplate"},
			},
		},
		{
			ID:   "php",
			Name: "PHP",
			Frameworks: []Framework{
				{ID: "laravel", Name: "Laravel", Description: "Expressive PHP framework with elegant syntax"},
				{ID: "symfony", Name: "Symfony", Description: "Flexible PHP framework for complex applications"},
			},
		},
		{
			ID:   "ruby",
			Name: "Ruby",
			Frameworks: []Framework{
				{ID: "rails", Name: "Ruby on Rails", Description: "Full-stack framework emphasizing convention over configuration"},
				{ID: "sinatra", Name: "Sinatra", Description: "Lightweight DSL for creating web applications"},
			},
		},
		{
			ID:   "rust",
			Name: "Rust",
			Frameworks: []Framework{
				{ID: "actix", Name: "Actix Web", Description: "Powerful, pragmatic Rust web framework"},
				{ID: "axum", Name: "Axum", Description: "Ergonomic web framework built on Tokio and Tower"},
			},
		},
	}
}

// FindLanguage looks up a language by its ID.
func FindLanguage(id string) (Language, bool) {
	for _, lang := range SupportedLanguages() {
		if lang.ID == id {
			return lang, true
		}
	}
	return Language{}, false
}

// FindFramework looks up a framework within a language by both IDs.
func FindFramework(languageID, frameworkID string) (Framework, bool) {
	lang, ok := FindLanguage(languageID)
	if !ok {
		return Framework{}, false
	}
	for _, fw := range lang.Frameworks {
		if fw.ID == frameworkID {
			return fw, true
		}
	}
	return Framework{}, false
}
