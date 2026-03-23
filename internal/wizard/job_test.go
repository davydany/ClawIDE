package wizard

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJob(t *testing.T) {
	req := WizardRequest{
		ProjectName: "test",
		Language:    "python",
		Framework:   "django",
	}
	job := NewJob(req)

	assert.NotEmpty(t, job.ID)
	assert.Equal(t, JobStatusPending, job.Status)
	assert.Equal(t, req, job.Request)
	assert.Len(t, job.Steps, 7)
	assert.False(t, job.CreatedAt.IsZero())
}

func TestJob_StepLifecycle(t *testing.T) {
	job := NewJob(WizardRequest{ProjectName: "test"})

	// Start a step
	job.StartStep("validate")
	snap := job.Snapshot()
	assert.Equal(t, JobStatusRunning, snap.Status)
	assert.Equal(t, JobStatusRunning, snap.Steps[0].Status)
	assert.False(t, snap.Steps[0].StartedAt.IsZero())

	// Complete a step
	job.CompleteStep("validate", "all checks passed")
	snap = job.Snapshot()
	assert.Equal(t, JobStatusCompleted, snap.Steps[0].Status)
	assert.Equal(t, "all checks passed", snap.Steps[0].Message)
	assert.False(t, snap.Steps[0].EndedAt.IsZero())
}

func TestJob_FailStep(t *testing.T) {
	job := NewJob(WizardRequest{ProjectName: "test"})

	job.StartStep("create_directory")
	job.FailStep("create_directory", errors.New("permission denied"))

	snap := job.Snapshot()
	assert.Equal(t, JobStatusFailed, snap.Status)
	assert.Equal(t, JobStatusFailed, snap.Steps[1].Status)
	assert.Equal(t, "permission denied", snap.Steps[1].Message)
	assert.Equal(t, "permission denied", snap.Error)
}

func TestJob_Complete(t *testing.T) {
	job := NewJob(WizardRequest{ProjectName: "test"})
	job.Complete("/path/to/project")

	snap := job.Snapshot()
	assert.Equal(t, JobStatusCompleted, snap.Status)
	assert.Equal(t, "/path/to/project", snap.OutputDir)
}

func TestJob_MarkRolledBack(t *testing.T) {
	job := NewJob(WizardRequest{ProjectName: "test"})
	job.Fail(errors.New("something broke"))
	job.MarkRolledBack()

	snap := job.Snapshot()
	assert.Equal(t, JobStatusRolledBack, snap.Status)
}

func TestJob_Snapshot_IsCopy(t *testing.T) {
	job := NewJob(WizardRequest{ProjectName: "test"})
	snap := job.Snapshot()

	// Modifying the snapshot shouldn't affect the original
	snap.Steps[0].Status = JobStatusCompleted
	assert.Equal(t, JobStatusPending, job.Steps[0].Status)
}

func TestJobTracker_AddAndGet(t *testing.T) {
	tracker := NewJobTracker()

	req := WizardRequest{ProjectName: "test", Language: "go"}
	job := tracker.Add(req)
	assert.NotEmpty(t, job.ID)

	retrieved, err := tracker.Get(job.ID)
	require.NoError(t, err)
	assert.Equal(t, job.ID, retrieved.ID)
	assert.Equal(t, req.ProjectName, retrieved.Request.ProjectName)
}

func TestJobTracker_GetNotFound(t *testing.T) {
	tracker := NewJobTracker()
	_, err := tracker.Get("nonexistent")
	assert.Error(t, err)
}

func TestJobTracker_Remove(t *testing.T) {
	tracker := NewJobTracker()
	job := tracker.Add(WizardRequest{ProjectName: "test"})

	tracker.Remove(job.ID)

	_, err := tracker.Get(job.ID)
	assert.Error(t, err)
}

func TestJobTracker_List(t *testing.T) {
	tracker := NewJobTracker()
	tracker.Add(WizardRequest{ProjectName: "project1"})
	tracker.Add(WizardRequest{ProjectName: "project2"})

	jobs := tracker.List()
	assert.Len(t, jobs, 2)
}

func TestJobTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewJobTracker()
	done := make(chan struct{})

	// Concurrent writes
	for i := 0; i < 50; i++ {
		go func() {
			tracker.Add(WizardRequest{ProjectName: "test"})
			done <- struct{}{}
		}()
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		go func() {
			tracker.List()
			done <- struct{}{}
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	assert.Len(t, tracker.List(), 50)
}

func TestNewJob_EmptyProject(t *testing.T) {
	req := WizardRequest{
		ProjectName:  "empty-test",
		EmptyProject: true,
	}
	job := NewJob(req)

	assert.NotEmpty(t, job.ID)
	assert.Equal(t, JobStatusPending, job.Status)
	assert.Len(t, job.Steps, 5, "empty project should have 5 steps")

	expectedSteps := []string{"validate", "create_directory", "copy_docs", "generate_claude_md", "init_git"}
	for i, expected := range expectedSteps {
		assert.Equal(t, expected, job.Steps[i].Name, "step %d should be %s", i, expected)
	}
}

func TestNewJob_TemplateProject(t *testing.T) {
	req := WizardRequest{
		ProjectName:  "template-test",
		Language:     "python",
		Framework:    "django",
		EmptyProject: false,
	}
	job := NewJob(req)
	assert.Len(t, job.Steps, 7, "template project should have 7 steps")
}

func TestNewJob_CloneProject(t *testing.T) {
	req := WizardRequest{
		ProjectName:  "clone-test",
		CloneProject: true,
		GitCloneURL:  "https://github.com/user/repo.git",
	}
	job := NewJob(req)

	assert.NotEmpty(t, job.ID)
	assert.Equal(t, JobStatusPending, job.Status)
	assert.Len(t, job.Steps, 3, "clone project should have 3 steps")

	expectedSteps := []string{"validate", "clone_repository", "register_project"}
	for i, expected := range expectedSteps {
		assert.Equal(t, expected, job.Steps[i].Name, "step %d should be %s", i, expected)
	}
}
