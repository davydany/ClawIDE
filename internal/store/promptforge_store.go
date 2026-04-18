package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/davydany/ClawIDE/internal/model"
	"gopkg.in/yaml.v3"
)

// promptForgeIndex is persisted in <baseDir>/index.yaml and holds folder metadata.
type promptForgeIndex struct {
	Folders []model.Folder `yaml:"folders"`
}

// PromptForgeStore is a global, file-based store for PromptForge prompts.
// Disk layout under <baseDir>:
//
//	index.yaml                           folder tree
//	<Prompt-Title>.md                    root-level prompt
//	<folder-name>/<Prompt-Title>.md      nested prompt
//	<folder-name>/<Prompt-Title>.compiled/<version-id>.md
//
// File format is YAML frontmatter (--- block) followed by the markdown body.
type PromptForgeStore struct {
	mu      sync.RWMutex
	baseDir string
	index   promptForgeIndex
	prompts []model.Prompt
}

// NewPromptForgeStore initializes (or creates) the PromptForge data directory
// and loads any existing folders + prompts into memory.
func NewPromptForgeStore(baseDir string) (*PromptForgeStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("creating promptforge dir: %w", err)
	}

	s := &PromptForgeStore{baseDir: baseDir}
	if err := s.loadIndex(); err != nil {
		s.index = promptForgeIndex{}
	}
	if err := s.loadPrompts(); err != nil {
		return nil, fmt.Errorf("loading prompts: %w", err)
	}
	return s, nil
}

// --------------- Folder CRUD ---------------

func (s *PromptForgeStore) GetFolders() []model.Folder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Folder, len(s.index.Folders))
	copy(out, s.index.Folders)
	sort.Slice(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

func (s *PromptForgeStore) GetFolder(id string) (model.Folder, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, f := range s.index.Folders {
		if f.ID == id {
			return f, true
		}
	}
	return model.Folder{}, false
}

func (s *PromptForgeStore) CreateFolder(f model.Folder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := model.ValidateFolderDepth(f.ParentID, s.index.Folders); err != nil {
		return err
	}
	if s.folderExistsAt(f.Name, f.ParentID, "") {
		return fmt.Errorf("a folder named %q already exists at this level", f.Name)
	}
	s.index.Folders = append(s.index.Folders, f)
	if err := s.saveIndex(); err != nil {
		return err
	}
	dirPath := filepath.Join(s.baseDir, s.folderDirPath(f.ID))
	return os.MkdirAll(dirPath, 0755)
}

func (s *PromptForgeStore) UpdateFolder(f model.Folder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.index.Folders {
		if existing.ID != f.ID {
			continue
		}
		if f.ParentID != existing.ParentID {
			if err := model.ValidateFolderDepth(f.ParentID, s.index.Folders); err != nil {
				return err
			}
		}
		if f.Name != existing.Name || f.ParentID != existing.ParentID {
			if s.folderExistsAt(f.Name, f.ParentID, f.ID) {
				return fmt.Errorf("a folder named %q already exists at this level", f.Name)
			}
		}
		oldDirPath := filepath.Join(s.baseDir, s.folderDirPath(f.ID))
		s.index.Folders[i] = f
		newDirPath := filepath.Join(s.baseDir, s.folderDirPath(f.ID))
		if err := s.saveIndex(); err != nil {
			return err
		}
		if oldDirPath != newDirPath {
			_ = os.MkdirAll(filepath.Dir(newDirPath), 0755)
			if _, err := os.Stat(oldDirPath); err == nil {
				if err := os.Rename(oldDirPath, newDirPath); err != nil {
					return fmt.Errorf("moving folder directory: %w", err)
				}
			}
			s.cleanEmptyDirs(filepath.Dir(oldDirPath))
		}
		return nil
	}
	return fmt.Errorf("folder %s not found", f.ID)
}

// DeleteFolder removes a folder and everything inside it (prompts and
// compiled version directories). Descendant folders are also removed.
func (s *PromptForgeStore) DeleteFolder(id string, cascade bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.folderExists(id) {
		return fmt.Errorf("folder %s not found", id)
	}

	descendantIDs := s.collectDescendantFolderIDs(id)

	if !cascade {
		if s.folderHasContent(descendantIDs) {
			return fmt.Errorf("folder is not empty; pass cascade=true to delete it and its contents")
		}
	}

	folderDir := filepath.Join(s.baseDir, s.folderDirPath(id))

	// Drop prompts whose folder is any of the descendants from the in-memory slice.
	descendantSet := make(map[string]bool, len(descendantIDs))
	for _, did := range descendantIDs {
		descendantSet[did] = true
	}
	kept := s.prompts[:0]
	for _, p := range s.prompts {
		if descendantSet[p.FolderID] {
			continue
		}
		kept = append(kept, p)
	}
	s.prompts = kept

	// Drop the descendant folders from the index.
	keptFolders := s.index.Folders[:0]
	for _, f := range s.index.Folders {
		if descendantSet[f.ID] {
			continue
		}
		keptFolders = append(keptFolders, f)
	}
	s.index.Folders = keptFolders

	// Remove the directory tree on disk.
	if err := os.RemoveAll(folderDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing folder directory: %w", err)
	}
	s.cleanEmptyDirs(filepath.Dir(folderDir))

	return s.saveIndex()
}

// --------------- Prompt CRUD ---------------

func (s *PromptForgeStore) GetAllPrompts() []model.Prompt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Prompt, len(s.prompts))
	copy(out, s.prompts)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title)
	})
	return out
}

func (s *PromptForgeStore) GetPromptsByFolder(folderID string) []model.Prompt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Prompt
	for _, p := range s.prompts {
		if p.FolderID == folderID {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title)
	})
	return out
}

func (s *PromptForgeStore) SearchPrompts(query string) []model.Prompt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		out := make([]model.Prompt, len(s.prompts))
		copy(out, s.prompts)
		return out
	}
	var out []model.Prompt
	for _, p := range s.prompts {
		if strings.Contains(strings.ToLower(p.Title), q) || strings.Contains(strings.ToLower(p.Content), q) {
			out = append(out, p)
		}
	}
	return out
}

func (s *PromptForgeStore) GetPrompt(id string) (model.Prompt, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.prompts {
		if p.ID == id {
			return p, true
		}
	}
	return model.Prompt{}, false
}

func (s *PromptForgeStore) AddPrompt(p model.Prompt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasPromptWithTitle(p.Title, p.FolderID, "") {
		return fmt.Errorf("a prompt titled %q already exists in this folder", p.Title)
	}
	s.prompts = append(s.prompts, p)
	return s.savePromptFile(p)
}

func (s *PromptForgeStore) UpdatePrompt(p model.Prompt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.prompts {
		if existing.ID != p.ID {
			continue
		}
		if p.Title != existing.Title || p.FolderID != existing.FolderID {
			if s.hasPromptWithTitle(p.Title, p.FolderID, p.ID) {
				return fmt.Errorf("a prompt titled %q already exists in this folder", p.Title)
			}
		}
		oldPath := s.promptFilePath(existing)
		oldCompiledDir := s.compiledDirPath(existing)
		s.prompts[i] = p
		newPath := s.promptFilePath(p)
		newCompiledDir := s.compiledDirPath(p)

		if oldPath != newPath {
			_ = os.MkdirAll(filepath.Dir(newPath), 0755)
			_ = os.Remove(oldPath)
			// Move the compiled-versions directory alongside the prompt.
			if _, err := os.Stat(oldCompiledDir); err == nil {
				_ = os.MkdirAll(filepath.Dir(newCompiledDir), 0755)
				if err := os.Rename(oldCompiledDir, newCompiledDir); err != nil {
					return fmt.Errorf("moving compiled-versions directory: %w", err)
				}
			}
			s.cleanEmptyDirs(filepath.Dir(oldPath))
		}

		return s.savePromptFile(p)
	}
	return fmt.Errorf("prompt %s not found", p.ID)
}

func (s *PromptForgeStore) DeletePrompt(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.prompts {
		if p.ID != id {
			continue
		}
		path := s.promptFilePath(p)
		compiledDir := s.compiledDirPath(p)
		s.prompts = append(s.prompts[:i], s.prompts[i+1:]...)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.RemoveAll(compiledDir); err != nil && !os.IsNotExist(err) {
			return err
		}
		s.cleanEmptyDirs(filepath.Dir(path))
		return nil
	}
	return fmt.Errorf("prompt %s not found", id)
}

// --------------- Compiled Version CRUD ---------------

func (s *PromptForgeStore) GetVersions(promptID string) ([]model.CompiledVersion, error) {
	s.mu.RLock()
	p, ok := s.findPromptLocked(promptID)
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("prompt %s not found", promptID)
	}
	dir := s.compiledDirPath(p)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.CompiledVersion{}, nil
		}
		return nil, err
	}
	var out []model.CompiledVersion
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		v, err := parseCompiledVersion(data)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CompiledAt.After(out[j].CompiledAt)
	})
	return out, nil
}

func (s *PromptForgeStore) GetVersion(promptID, versionID string) (model.CompiledVersion, error) {
	s.mu.RLock()
	p, ok := s.findPromptLocked(promptID)
	s.mu.RUnlock()
	if !ok {
		return model.CompiledVersion{}, fmt.Errorf("prompt %s not found", promptID)
	}
	path := filepath.Join(s.compiledDirPath(p), versionID+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return model.CompiledVersion{}, fmt.Errorf("version %s not found", versionID)
	}
	return parseCompiledVersion(data)
}

func (s *PromptForgeStore) AddVersion(v model.CompiledVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.findPromptLocked(v.PromptID)
	if !ok {
		return fmt.Errorf("prompt %s not found", v.PromptID)
	}
	dir := s.compiledDirPath(p)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating compiled dir: %w", err)
	}
	return writeCompiledVersion(filepath.Join(dir, v.ID+".md"), v)
}

func (s *PromptForgeStore) UpdateVersion(v model.CompiledVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.findPromptLocked(v.PromptID)
	if !ok {
		return fmt.Errorf("prompt %s not found", v.PromptID)
	}
	path := filepath.Join(s.compiledDirPath(p), v.ID+".md")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("version %s not found", v.ID)
	}
	return writeCompiledVersion(path, v)
}

func (s *PromptForgeStore) DeleteVersion(promptID, versionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.findPromptLocked(promptID)
	if !ok {
		return fmt.Errorf("prompt %s not found", promptID)
	}
	path := filepath.Join(s.compiledDirPath(p), versionID+".md")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("version %s not found", versionID)
		}
		return err
	}
	return nil
}

// --------------- Internal helpers ---------------

func (s *PromptForgeStore) findPromptLocked(id string) (model.Prompt, bool) {
	for _, p := range s.prompts {
		if p.ID == id {
			return p, true
		}
	}
	return model.Prompt{}, false
}

func (s *PromptForgeStore) folderExists(id string) bool {
	for _, f := range s.index.Folders {
		if f.ID == id {
			return true
		}
	}
	return false
}

func (s *PromptForgeStore) folderExistsAt(name, parentID, excludeID string) bool {
	for _, f := range s.index.Folders {
		if f.ID == excludeID {
			continue
		}
		if f.Name == name && f.ParentID == parentID {
			return true
		}
	}
	return false
}

func (s *PromptForgeStore) hasPromptWithTitle(title, folderID, excludeID string) bool {
	for _, p := range s.prompts {
		if p.ID == excludeID {
			continue
		}
		if p.Title == title && p.FolderID == folderID {
			return true
		}
	}
	return false
}

// collectDescendantFolderIDs returns the folder itself and every folder
// nested (recursively) beneath it.
func (s *PromptForgeStore) collectDescendantFolderIDs(rootID string) []string {
	result := []string{rootID}
	frontier := []string{rootID}
	for len(frontier) > 0 {
		var next []string
		for _, id := range frontier {
			for _, f := range s.index.Folders {
				if f.ParentID == id {
					result = append(result, f.ID)
					next = append(next, f.ID)
				}
			}
		}
		frontier = next
	}
	return result
}

func (s *PromptForgeStore) folderHasContent(folderIDs []string) bool {
	set := make(map[string]bool, len(folderIDs))
	for _, id := range folderIDs {
		set[id] = true
	}
	if len(set) > 1 {
		// has nested folders besides the root
		return true
	}
	for _, p := range s.prompts {
		if set[p.FolderID] {
			return true
		}
	}
	return false
}

// --------------- Path helpers ---------------

func (s *PromptForgeStore) folderDirPath(folderID string) string {
	if folderID == "" {
		return ""
	}
	var parts []string
	current := folderID
	seen := map[string]bool{}
	for current != "" {
		if seen[current] {
			break
		}
		seen[current] = true
		var found bool
		for _, f := range s.index.Folders {
			if f.ID == current {
				parts = append([]string{f.Name}, parts...)
				current = f.ParentID
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return filepath.Join(parts...)
}

func (s *PromptForgeStore) promptFilePath(p model.Prompt) string {
	return filepath.Join(s.baseDir, s.folderDirPath(p.FolderID), p.Title+".md")
}

func (s *PromptForgeStore) compiledDirPath(p model.Prompt) string {
	return filepath.Join(s.baseDir, s.folderDirPath(p.FolderID), p.Title+".compiled")
}

func (s *PromptForgeStore) dirPathToFolderID(relDir string) string {
	if relDir == "" || relDir == "." {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(relDir), "/")
	parentID := ""
	for _, part := range parts {
		var found bool
		for _, f := range s.index.Folders {
			if f.Name == part && f.ParentID == parentID {
				parentID = f.ID
				found = true
				break
			}
		}
		if !found {
			return ""
		}
	}
	return parentID
}

// cleanEmptyDirs removes empty directories walking up toward baseDir, stopping
// at baseDir itself.
func (s *PromptForgeStore) cleanEmptyDirs(dir string) {
	for dir != s.baseDir && strings.HasPrefix(dir, s.baseDir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		_ = os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// --------------- Persistence ---------------

const pfFrontmatterSep = "---\n"

func (s *PromptForgeStore) indexPath() string {
	return filepath.Join(s.baseDir, "index.yaml")
}

func (s *PromptForgeStore) loadIndex() error {
	data, err := os.ReadFile(s.indexPath())
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &s.index)
}

func (s *PromptForgeStore) saveIndex() error {
	data, err := yaml.Marshal(&s.index)
	if err != nil {
		return fmt.Errorf("encoding promptforge index: %w", err)
	}
	return os.WriteFile(s.indexPath(), data, 0644)
}

func (s *PromptForgeStore) savePromptFile(p model.Prompt) error {
	path := s.promptFilePath(p)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating prompt directory: %w", err)
	}
	return writePromptFile(path, p)
}

func writePromptFile(path string, p model.Prompt) error {
	var buf bytes.Buffer
	buf.WriteString(pfFrontmatterSep)
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(p); err != nil {
		return fmt.Errorf("encoding prompt frontmatter: %w", err)
	}
	_ = enc.Close()
	buf.WriteString(pfFrontmatterSep)
	buf.WriteString(p.Content)
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func writeCompiledVersion(path string, v model.CompiledVersion) error {
	var buf bytes.Buffer
	buf.WriteString(pfFrontmatterSep)
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding compiled frontmatter: %w", err)
	}
	_ = enc.Close()
	buf.WriteString(pfFrontmatterSep)
	buf.WriteString(v.Content)
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// loadPrompts walks baseDir finding every `*.md` file that isn't inside a
// `*.compiled/` directory. Folder-id is derived from the on-disk directory
// structure (index.yaml is already loaded at this point).
func (s *PromptForgeStore) loadPrompts() error {
	s.prompts = nil
	return filepath.WalkDir(s.baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasSuffix(d.Name(), ".compiled") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		p, parseErr := parsePromptFile(data)
		if parseErr != nil {
			return fmt.Errorf("parsing %s: %w", path, parseErr)
		}
		relDir, _ := filepath.Rel(s.baseDir, filepath.Dir(path))
		if relDir != "." && relDir != "" {
			p.FolderID = s.dirPathToFolderID(relDir)
		}
		s.prompts = append(s.prompts, p)
		return nil
	})
}

func parsePromptFile(data []byte) (model.Prompt, error) {
	content := string(data)
	if !strings.HasPrefix(content, pfFrontmatterSep) {
		return model.Prompt{}, fmt.Errorf("missing opening frontmatter separator")
	}
	content = content[len(pfFrontmatterSep):]
	idx := strings.Index(content, pfFrontmatterSep)
	if idx < 0 {
		return model.Prompt{}, fmt.Errorf("missing closing frontmatter separator")
	}
	frontmatter := content[:idx]
	body := content[idx+len(pfFrontmatterSep):]
	var p model.Prompt
	if err := yaml.Unmarshal([]byte(frontmatter), &p); err != nil {
		return model.Prompt{}, fmt.Errorf("decoding frontmatter: %w", err)
	}
	p.Content = body
	return p, nil
}

func parseCompiledVersion(data []byte) (model.CompiledVersion, error) {
	content := string(data)
	if !strings.HasPrefix(content, pfFrontmatterSep) {
		return model.CompiledVersion{}, fmt.Errorf("missing opening frontmatter separator")
	}
	content = content[len(pfFrontmatterSep):]
	idx := strings.Index(content, pfFrontmatterSep)
	if idx < 0 {
		return model.CompiledVersion{}, fmt.Errorf("missing closing frontmatter separator")
	}
	frontmatter := content[:idx]
	body := content[idx+len(pfFrontmatterSep):]
	var v model.CompiledVersion
	if err := yaml.Unmarshal([]byte(frontmatter), &v); err != nil {
		return model.CompiledVersion{}, fmt.Errorf("decoding frontmatter: %w", err)
	}
	v.Content = body
	return v, nil
}
