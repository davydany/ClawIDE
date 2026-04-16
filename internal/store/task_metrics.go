package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// DayStats holds the created/closed counts for a single day.
type DayStats struct {
	Created int `json:"created"`
	Closed  int `json:"closed"`
}

// TaskMetrics tracks daily task creation and closure counts. Stored as a JSON file at
// ~/.clawide/projects/<project-id>/task-metrics.json so it never pollutes the project repo.
type TaskMetrics struct {
	mu   sync.Mutex
	path string
	data metricsData
}

type metricsData struct {
	Daily map[string]DayStats `json:"daily"` // keyed by "2006-01-02"
}

// NewTaskMetrics creates or opens the metrics file for a project. globalDataDir is typically
// ~/.clawide (from config.DataDir). For the global board, pass "" as projectID.
func NewTaskMetrics(globalDataDir, projectID string) (*TaskMetrics, error) {
	var dir string
	if projectID == "" {
		dir = globalDataDir
	} else {
		dir = filepath.Join(globalDataDir, "projects", projectID)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating metrics dir: %w", err)
	}
	m := &TaskMetrics{path: filepath.Join(dir, "task-metrics.json")}
	if err := m.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading metrics: %w", err)
	}
	if m.data.Daily == nil {
		m.data.Daily = make(map[string]DayStats)
	}
	return m, nil
}

func (m *TaskMetrics) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.data)
}

func (m *TaskMetrics) save() error {
	data, err := json.MarshalIndent(m.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0644)
}

func today() string {
	return time.Now().Format("2006-01-02")
}

// RecordCreated increments today's "created" count by 1.
func (m *TaskMetrics) RecordCreated() {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := today()
	s := m.data.Daily[key]
	s.Created++
	m.data.Daily[key] = s
	m.save() // best-effort; metrics are non-critical
}

// RecordClosed increments today's "closed" count by 1.
func (m *TaskMetrics) RecordClosed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := today()
	s := m.data.Daily[key]
	s.Closed++
	m.data.Daily[key] = s
	m.save()
}

// DaySummary is one entry in the API response.
type DaySummary struct {
	Date    string `json:"date"`
	Created int    `json:"created"`
	Closed  int    `json:"closed"`
}

// Recent returns the last N days of stats (including days with zero activity so the chart has
// no gaps). Results are sorted oldest-first.
func (m *TaskMetrics) Recent(days int) []DaySummary {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	out := make([]DaySummary, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		s := m.data.Daily[d]
		out = append(out, DaySummary{Date: d, Created: s.Created, Closed: s.Closed})
	}
	return out
}

// All returns every recorded day, sorted oldest-first.
func (m *TaskMetrics) All() []DaySummary {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.data.Daily))
	for k := range m.data.Daily {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]DaySummary, 0, len(keys))
	for _, k := range keys {
		s := m.data.Daily[k]
		out = append(out, DaySummary{Date: k, Created: s.Created, Closed: s.Closed})
	}
	return out
}
