package aicli

import (
	"context"
	"os"
	"strings"
	"testing"
)

// fakeProvider lets us exercise the registry without depending on any real CLI binary.
type fakeProvider struct {
	id      string
	binary  string
	models  []ModelInfo
	runFn   func(ctx context.Context, req Request) (Response, error)
}

func (f *fakeProvider) ID() string                 { return f.id }
func (f *fakeProvider) DisplayName() string        { return f.id }
func (f *fakeProvider) Binary() string             { return f.binary }
func (f *fakeProvider) AvailableModels() []ModelInfo {
	if f.models == nil {
		return []ModelInfo{{ID: "default-model", DisplayName: "Default"}}
	}
	return f.models
}
func (f *fakeProvider) Run(ctx context.Context, req Request) (Response, error) {
	if f.runFn != nil {
		return f.runFn(ctx, req)
	}
	return Response{Text: "ok"}, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry(nil)
	p := &fakeProvider{id: "fake", binary: "/definitely/not/on/path/zzz"}
	r.Register(p)

	got, ok := r.Get("fake")
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.ID() != "fake" {
		t.Errorf("Get returned wrong provider: %q", got.ID())
	}
	if r.IsInstalled("fake") {
		t.Error("fake provider should not be installed — binary path is bogus")
	}
	if _, ok := r.Get("missing"); ok {
		t.Error("Get missing should return false")
	}
}

func TestRegistry_List_Sorted(t *testing.T) {
	r := NewRegistry(nil)
	r.Register(&fakeProvider{id: "charlie", binary: "nope"})
	r.Register(&fakeProvider{id: "alpha", binary: "nope"})
	r.Register(&fakeProvider{id: "bravo", binary: "nope"})
	list := r.List()
	if len(list) != 3 {
		t.Fatalf("len: %d", len(list))
	}
	for i, want := range []string{"alpha", "bravo", "charlie"} {
		if list[i].ID() != want {
			t.Errorf("index %d: got %q, want %q", i, list[i].ID(), want)
		}
	}
}

func TestRegistry_IsInstalled_RealBinary(t *testing.T) {
	// Use /bin/sh as a binary that's guaranteed to exist on macOS/Linux. It's not an AI CLI,
	// but it's good enough to verify IsInstalled() correctly uses exec.LookPath.
	r := NewRegistry(nil)
	r.Register(&fakeProvider{id: "sh", binary: "sh"})
	if !r.IsInstalled("sh") {
		t.Error("sh should be installed")
	}
}

func TestValidateModel(t *testing.T) {
	p := &fakeProvider{id: "fp", models: []ModelInfo{{ID: "m1"}, {ID: "m2"}}}
	if err := ValidateModel(p, "m1"); err != nil {
		t.Errorf("valid model rejected: %v", err)
	}
	if err := ValidateModel(p, "unknown"); err == nil {
		t.Error("unknown model accepted")
	}
	if err := ValidateModel(p, ""); err == nil {
		t.Error("empty model accepted")
	}
}

func TestParseClaudeOutput_Success(t *testing.T) {
	// Use the real claude output sample captured during plan phase.
	data, err := os.ReadFile("testdata/claude_sample.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	text, err := parseClaudeOutput(string(data))
	if err != nil {
		t.Fatalf("parseClaudeOutput: %v", err)
	}
	if text != "Ready to build some code?" {
		t.Errorf("result text: got %q", text)
	}
}

func TestParseClaudeOutput_Error(t *testing.T) {
	data, err := os.ReadFile("testdata/claude_error.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	_, err = parseClaudeOutput(string(data))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Rate limit") {
		t.Errorf("error message should include the upstream detail: %v", err)
	}
}

func TestParseClaudeOutput_Malformed(t *testing.T) {
	cases := map[string]string{
		"empty":    "",
		"not-json": "hello world",
		"no-result": `{"type":"result","is_error":false}`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseClaudeOutput(input); err == nil {
				t.Errorf("expected error for %s", name)
			}
		})
	}
}

func TestClaudeProvider_Metadata(t *testing.T) {
	p := NewClaudeProvider(NewRegistry(nil))
	if p.ID() != "claude" {
		t.Errorf("ID: %q", p.ID())
	}
	if p.Binary() != "claude" {
		t.Errorf("Binary: %q", p.Binary())
	}
	models := p.AvailableModels()
	if len(models) == 0 {
		t.Error("no models listed")
	}
	// Recommended default is sonnet.
	if models[0].ID != "sonnet" {
		t.Errorf("first model should be sonnet, got %q", models[0].ID)
	}
}

func TestRegisterDefaults(t *testing.T) {
	r := NewRegistry(nil)
	RegisterDefaults(r)
	for _, id := range []string{"claude", "codex", "ollama", "gemini"} {
		if _, ok := r.Get(id); !ok {
			t.Errorf("provider %q not registered", id)
		}
	}
}
