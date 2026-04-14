// Package fsutil provides small filesystem helpers used across ClawIDE.
package fsutil

import (
	"os"
	"path/filepath"
)

// MoveDir moves a directory from src to dst. It first attempts an atomic
// os.Rename (fast path when src and dst live on the same filesystem). On
// cross-filesystem moves (EXDEV on Linux, similar on macOS), it falls back
// to a recursive copy followed by os.RemoveAll on the source.
//
// The caller must ensure dst does not already exist; the parent of dst is
// created on demand.
func MoveDir(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Fallback: copy then remove.
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	return os.RemoveAll(src)
}
