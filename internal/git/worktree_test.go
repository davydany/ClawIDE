package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitWorktreeBlocks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // number of blocks
	}{
		{
			name: "single block",
			input: "worktree /home/user/project\n" +
				"HEAD abc123\n" +
				"branch refs/heads/main\n\n",
			want: 1,
		},
		{
			name: "multiple blocks",
			input: "worktree /home/user/project\n" +
				"HEAD abc123\n" +
				"branch refs/heads/main\n\n" +
				"worktree /home/user/project-worktrees/feature\n" +
				"HEAD def456\n" +
				"branch refs/heads/feature\n\n",
			want: 2,
		},
		{
			name: "trailing without blank line",
			input: "worktree /home/user/project\n" +
				"HEAD abc123\n" +
				"branch refs/heads/main",
			want: 1,
		},
		{
			name:  "empty input",
			input: "",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := splitWorktreeBlocks(tt.input)
			assert.Len(t, blocks, tt.want)
		})
	}
}

func TestParseWorktreeBlock(t *testing.T) {
	tests := []struct {
		name       string
		block      string
		wantPath   string
		wantBranch string
		wantHEAD   string
	}{
		{
			name:       "normal branch",
			block:      "worktree /home/user/project\nHEAD abc123\nbranch refs/heads/main",
			wantPath:   "/home/user/project",
			wantBranch: "main",
			wantHEAD:   "abc123",
		},
		{
			name:       "detached HEAD",
			block:      "worktree /home/user/project\nHEAD abc123\ndetached",
			wantPath:   "/home/user/project",
			wantBranch: "(detached)",
			wantHEAD:   "abc123",
		},
		{
			name:       "nested branch ref",
			block:      "worktree /home/user/wt\nHEAD def456\nbranch refs/heads/feature/auth",
			wantPath:   "/home/user/wt",
			wantBranch: "feature/auth",
			wantHEAD:   "def456",
		},
		{
			name:       "minimal block",
			block:      "worktree /path",
			wantPath:   "/path",
			wantBranch: "",
			wantHEAD:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wt := parseWorktreeBlock(tt.block)
			assert.Equal(t, tt.wantPath, wt.Path)
			assert.Equal(t, tt.wantBranch, wt.Branch)
			assert.Equal(t, tt.wantHEAD, wt.HEAD)
		})
	}
}

func TestWorktreeDir(t *testing.T) {
	got := WorktreeDir("/home/user/projects/myapp", "feature/auth")
	assert.Equal(t, "/home/user/projects/myapp-worktrees/feature/auth", got)

	got = WorktreeDir("/home/user/projects/myapp", "main")
	assert.Equal(t, "/home/user/projects/myapp-worktrees/main", got)
}
