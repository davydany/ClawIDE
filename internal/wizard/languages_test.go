package wizard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// SupportedLanguages
// ---------------------------------------------------------------------------

func TestSupportedLanguages_ReturnsAllExpectedLanguages(t *testing.T) {
	langs := SupportedLanguages()

	expectedIDs := []string{
		"python", "javascript", "go", "java", "csharp", "php", "ruby", "rust",
	}
	gotIDs := make([]string, len(langs))
	for i, l := range langs {
		gotIDs[i] = l.ID
	}
	assert.ElementsMatch(t, expectedIDs, gotIDs, "supported language IDs should match expected set")
}

func TestSupportedLanguages_EachLanguageHasFrameworks(t *testing.T) {
	for _, lang := range SupportedLanguages() {
		t.Run(lang.ID, func(t *testing.T) {
			assert.NotEmpty(t, lang.Frameworks, "language %q must have at least one framework", lang.ID)
		})
	}
}

func TestSupportedLanguages_NoDuplicateLanguageIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, lang := range SupportedLanguages() {
		assert.False(t, seen[lang.ID], "duplicate language ID: %s", lang.ID)
		seen[lang.ID] = true
	}
}

func TestSupportedLanguages_NoDuplicateFrameworkIDs(t *testing.T) {
	for _, lang := range SupportedLanguages() {
		t.Run(lang.ID, func(t *testing.T) {
			seen := make(map[string]bool)
			for _, fw := range lang.Frameworks {
				assert.False(t, seen[fw.ID], "duplicate framework ID %q in language %q", fw.ID, lang.ID)
				seen[fw.ID] = true
			}
		})
	}
}

func TestSupportedLanguages_AllFieldsPopulated(t *testing.T) {
	for _, lang := range SupportedLanguages() {
		t.Run(lang.ID, func(t *testing.T) {
			assert.NotEmpty(t, lang.ID, "language ID must not be empty")
			assert.NotEmpty(t, lang.Name, "language Name must not be empty")

			for _, fw := range lang.Frameworks {
				t.Run(fw.ID, func(t *testing.T) {
					assert.NotEmpty(t, fw.ID, "framework ID must not be empty")
					assert.NotEmpty(t, fw.Name, "framework Name must not be empty")
					assert.NotEmpty(t, fw.Description, "framework Description must not be empty")
				})
			}
		})
	}
}

func TestSupportedLanguages_StableOrder(t *testing.T) {
	first := SupportedLanguages()
	second := SupportedLanguages()

	require.Equal(t, len(first), len(second), "length must be stable across calls")
	for i := range first {
		assert.Equal(t, first[i].ID, second[i].ID, "language order must be stable at index %d", i)
	}
}

// ---------------------------------------------------------------------------
// Specific language/framework combos (regression guards)
// ---------------------------------------------------------------------------

func TestSupportedLanguages_PythonFrameworks(t *testing.T) {
	lang, ok := FindLanguage("python")
	require.True(t, ok)

	expected := []string{"django", "fastapi", "flask"}
	got := make([]string, len(lang.Frameworks))
	for i, fw := range lang.Frameworks {
		got[i] = fw.ID
	}
	assert.ElementsMatch(t, expected, got)
}

func TestSupportedLanguages_JavaScriptFrameworks(t *testing.T) {
	lang, ok := FindLanguage("javascript")
	require.True(t, ok)

	expected := []string{"node-express", "react", "nextjs", "vue"}
	got := make([]string, len(lang.Frameworks))
	for i, fw := range lang.Frameworks {
		got[i] = fw.ID
	}
	assert.ElementsMatch(t, expected, got)
}

func TestSupportedLanguages_GoFrameworks(t *testing.T) {
	lang, ok := FindLanguage("go")
	require.True(t, ok)

	expected := []string{"gin", "echo", "chi"}
	got := make([]string, len(lang.Frameworks))
	for i, fw := range lang.Frameworks {
		got[i] = fw.ID
	}
	assert.ElementsMatch(t, expected, got)
}

func TestSupportedLanguages_JavaFrameworks(t *testing.T) {
	lang, ok := FindLanguage("java")
	require.True(t, ok)

	expected := []string{"spring-boot", "quarkus"}
	got := make([]string, len(lang.Frameworks))
	for i, fw := range lang.Frameworks {
		got[i] = fw.ID
	}
	assert.ElementsMatch(t, expected, got)
}

func TestSupportedLanguages_CSharpFrameworks(t *testing.T) {
	lang, ok := FindLanguage("csharp")
	require.True(t, ok)

	expected := []string{"aspnet", "minimal-api"}
	got := make([]string, len(lang.Frameworks))
	for i, fw := range lang.Frameworks {
		got[i] = fw.ID
	}
	assert.ElementsMatch(t, expected, got)
}

func TestSupportedLanguages_PHPFrameworks(t *testing.T) {
	lang, ok := FindLanguage("php")
	require.True(t, ok)

	expected := []string{"laravel", "symfony"}
	got := make([]string, len(lang.Frameworks))
	for i, fw := range lang.Frameworks {
		got[i] = fw.ID
	}
	assert.ElementsMatch(t, expected, got)
}

func TestSupportedLanguages_RubyFrameworks(t *testing.T) {
	lang, ok := FindLanguage("ruby")
	require.True(t, ok)

	expected := []string{"rails", "sinatra"}
	got := make([]string, len(lang.Frameworks))
	for i, fw := range lang.Frameworks {
		got[i] = fw.ID
	}
	assert.ElementsMatch(t, expected, got)
}

func TestSupportedLanguages_RustFrameworks(t *testing.T) {
	lang, ok := FindLanguage("rust")
	require.True(t, ok)

	expected := []string{"actix", "axum"}
	got := make([]string, len(lang.Frameworks))
	for i, fw := range lang.Frameworks {
		got[i] = fw.ID
	}
	assert.ElementsMatch(t, expected, got)
}

// ---------------------------------------------------------------------------
// FindLanguage
// ---------------------------------------------------------------------------

func TestFindLanguage_Found(t *testing.T) {
	tests := []struct {
		id           string
		expectedName string
	}{
		{"python", "Python"},
		{"javascript", "JavaScript"},
		{"go", "Go"},
		{"java", "Java"},
		{"csharp", "C#"},
		{"php", "PHP"},
		{"ruby", "Ruby"},
		{"rust", "Rust"},
	}

	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			lang, ok := FindLanguage(tc.id)
			require.True(t, ok, "FindLanguage(%q) should return true", tc.id)
			assert.Equal(t, tc.id, lang.ID)
			assert.Equal(t, tc.expectedName, lang.Name)
		})
	}
}

func TestFindLanguage_NotFound(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"empty string", ""},
		{"wrong case Python", "Python"},
		{"upper case PYTHON", "PYTHON"},
		{"typescript not separate", "typescript"},
		{"c++ not supported", "c++"},
		{"swift not supported", "swift"},
		{"kotlin not supported", "kotlin"},
		{"random string", "nonexistent"},
		{"leading space", " python"},
		{"trailing space", "python "},
		{"with newline", "python\n"},
		{"with tab", "python\t"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lang, ok := FindLanguage(tc.id)
			assert.False(t, ok, "FindLanguage(%q) should return false", tc.id)
			assert.Equal(t, Language{}, lang, "FindLanguage(%q) should return zero Language", tc.id)
		})
	}
}

func TestFindLanguage_ReturnsCopy(t *testing.T) {
	// Verify returned value is independent (since we return by value)
	lang1, ok := FindLanguage("python")
	require.True(t, ok)

	lang2, ok := FindLanguage("python")
	require.True(t, ok)

	// Modify lang1's frameworks slice should not affect lang2
	lang1.Frameworks = nil
	assert.NotEmpty(t, lang2.Frameworks, "modifications to returned Language should not affect subsequent calls")
}

// ---------------------------------------------------------------------------
// FindFramework
// ---------------------------------------------------------------------------

func TestFindFramework_Found(t *testing.T) {
	tests := []struct {
		langID string
		fwID   string
		fwName string
	}{
		{"python", "django", "Django"},
		{"python", "fastapi", "FastAPI"},
		{"python", "flask", "Flask"},
		{"javascript", "node-express", "Node.js + Express"},
		{"javascript", "react", "React (Vite)"},
		{"javascript", "nextjs", "Next.js"},
		{"javascript", "vue", "Vue.js (Vite)"},
		{"go", "gin", "Gin"},
		{"go", "echo", "Echo"},
		{"go", "chi", "Chi"},
		{"java", "spring-boot", "Spring Boot"},
		{"java", "quarkus", "Quarkus"},
		{"csharp", "aspnet", "ASP.NET Core"},
		{"csharp", "minimal-api", "Minimal API"},
		{"php", "laravel", "Laravel"},
		{"php", "symfony", "Symfony"},
		{"ruby", "rails", "Ruby on Rails"},
		{"ruby", "sinatra", "Sinatra"},
		{"rust", "actix", "Actix Web"},
		{"rust", "axum", "Axum"},
	}

	for _, tc := range tests {
		t.Run(tc.langID+"/"+tc.fwID, func(t *testing.T) {
			fw, ok := FindFramework(tc.langID, tc.fwID)
			require.True(t, ok, "FindFramework(%q, %q) should return true", tc.langID, tc.fwID)
			assert.Equal(t, tc.fwID, fw.ID)
			assert.Equal(t, tc.fwName, fw.Name)
			assert.NotEmpty(t, fw.Description, "framework %q description should not be empty", tc.fwID)
		})
	}
}

func TestFindFramework_InvalidLanguage(t *testing.T) {
	fw, ok := FindFramework("nonexistent", "django")
	assert.False(t, ok)
	assert.Equal(t, Framework{}, fw)
}

func TestFindFramework_InvalidFramework(t *testing.T) {
	fw, ok := FindFramework("python", "nonexistent")
	assert.False(t, ok)
	assert.Equal(t, Framework{}, fw)
}

func TestFindFramework_EmptyLanguageID(t *testing.T) {
	fw, ok := FindFramework("", "django")
	assert.False(t, ok)
	assert.Equal(t, Framework{}, fw)
}

func TestFindFramework_EmptyFrameworkID(t *testing.T) {
	fw, ok := FindFramework("python", "")
	assert.False(t, ok)
	assert.Equal(t, Framework{}, fw)
}

func TestFindFramework_BothEmpty(t *testing.T) {
	fw, ok := FindFramework("", "")
	assert.False(t, ok)
	assert.Equal(t, Framework{}, fw)
}

func TestFindFramework_WrongLanguageForFramework(t *testing.T) {
	// Each framework should only be found under its correct language.
	crossLanguage := []struct {
		langID string
		fwID   string
	}{
		{"javascript", "django"},       // django is python
		{"go", "flask"},                // flask is python
		{"python", "node-express"},     // node-express is javascript
		{"python", "gin"},              // gin is go
		{"javascript", "spring-boot"},  // spring-boot is java
		{"go", "laravel"},              // laravel is php
		{"php", "rails"},              // rails is ruby
		{"ruby", "axum"},              // axum is rust
	}

	for _, tc := range crossLanguage {
		t.Run(tc.langID+"/"+tc.fwID, func(t *testing.T) {
			fw, ok := FindFramework(tc.langID, tc.fwID)
			assert.False(t, ok, "%q should not be found under %q", tc.fwID, tc.langID)
			assert.Equal(t, Framework{}, fw)
		})
	}
}

func TestFindFramework_CaseSensitive(t *testing.T) {
	tests := []struct {
		name   string
		langID string
		fwID   string
	}{
		{"wrong lang case", "Python", "django"},
		{"wrong fw case", "python", "Django"},
		{"all caps", "PYTHON", "DJANGO"},
		{"mixed case", "Python", "Django"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fw, ok := FindFramework(tc.langID, tc.fwID)
			assert.False(t, ok)
			assert.Equal(t, Framework{}, fw)
		})
	}
}

// ---------------------------------------------------------------------------
// Cross-validation: every framework is reachable via FindFramework
// ---------------------------------------------------------------------------

func TestFindFramework_AllSupportedCombosReachable(t *testing.T) {
	for _, lang := range SupportedLanguages() {
		for _, fw := range lang.Frameworks {
			t.Run(lang.ID+"/"+fw.ID, func(t *testing.T) {
				found, ok := FindFramework(lang.ID, fw.ID)
				require.True(t, ok)
				assert.Equal(t, fw.ID, found.ID)
				assert.Equal(t, fw.Name, found.Name)
				assert.Equal(t, fw.Description, found.Description)
			})
		}
	}
}
