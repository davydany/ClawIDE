package updater

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
)

// installing guards against concurrent install attempts.
var installing atomic.Bool

// IsInstalling returns true if an install is currently in progress.
func IsInstalling() bool {
	return installing.Load()
}

// IsDocker returns true when running inside a Docker container.
func IsDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

// Install downloads, verifies, extracts, replaces the binary, and restarts.
// It runs as a goroutine triggered by the handler.
func (u *Updater) Install() error {
	if !installing.CompareAndSwap(false, true) {
		return fmt.Errorf("install already in progress")
	}
	defer installing.Store(false)

	if IsDocker() {
		return fmt.Errorf("running in Docker: use docker pull to update instead")
	}

	state := u.State()
	if !state.UpdateAvailable || state.AssetURL == "" {
		return fmt.Errorf("no update available to install")
	}

	tmpDir := u.cfg.UpdateTempDir()
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Download archive
	archivePath := filepath.Join(tmpDir, platformAssetName(state.LatestVersion))
	log.Printf("[updater] downloading %s", state.AssetURL)
	if err := downloadFile(u.client, state.AssetURL, archivePath); err != nil {
		return fmt.Errorf("downloading archive: %w", err)
	}

	// 2. Download checksums
	checksumsURL := strings.TrimSuffix(state.AssetURL, filepath.Base(state.AssetURL)) + "checksums.txt"
	checksumsPath := filepath.Join(tmpDir, "checksums.txt")
	if err := downloadFile(u.client, checksumsURL, checksumsPath); err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}

	// 3. Verify SHA-256
	if err := verifyChecksum(archivePath, checksumsPath); err != nil {
		return fmt.Errorf("checksum verification: %w", err)
	}
	log.Println("[updater] checksum verified")

	// 4. Extract binary from tar.gz
	extractedBinary := filepath.Join(tmpDir, "clawide")
	if err := extractBinary(archivePath, extractedBinary); err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}
	log.Println("[updater] binary extracted")

	// 5. Replace current binary
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}

	if err := replaceBinary(currentBinary, extractedBinary); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}
	log.Println("[updater] binary replaced")

	// 6. Restart
	log.Println("[updater] restarting ClawIDE...")
	cmd := exec.Command(currentBinary, "--restart")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting new process: %w", err)
	}

	os.Exit(0)
	return nil // unreachable
}

func downloadFile(client *http.Client, url, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// verifyChecksum reads checksums.txt and verifies the archive's SHA-256 hash.
func verifyChecksum(archivePath, checksumsPath string) error {
	expected, err := findExpectedChecksum(checksumsPath, filepath.Base(archivePath))
	if err != nil {
		return err
	}

	actual, err := fileSHA256(archivePath)
	if err != nil {
		return fmt.Errorf("computing hash: %w", err)
	}

	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

func findExpectedChecksum(checksumsPath, filename string) (string, error) {
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<hash>  <filename>" or "<hash> <filename>"
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == filename {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("no checksum found for %s", filename)
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// extractBinary extracts the "clawide" binary from a tar.gz archive.
func extractBinary(archivePath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip open: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	binaryName := "clawide"
	if runtime.GOOS == "windows" {
		binaryName = "clawide.exe"
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("binary %q not found in archive", binaryName)
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Match the binary name regardless of directory prefix
		name := filepath.Base(hdr.Name)
		if name == binaryName && hdr.Typeflag == tar.TypeReg {
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return fmt.Errorf("extracting binary: %w", err)
			}
			return nil
		}
	}
}

// replaceBinary replaces the current binary with the new one.
// Strategy: rename current to .old, copy new in place, remove .old.
func replaceBinary(currentPath, newPath string) error {
	// Check write permissions
	dir := filepath.Dir(currentPath)
	if err := checkWritable(dir); err != nil {
		return fmt.Errorf("binary directory not writable: %w", err)
	}

	backupPath := currentPath + ".old"

	// Remove any stale backup
	os.Remove(backupPath)

	// Rename current to backup
	if err := os.Rename(currentPath, backupPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	// Copy new binary into place
	src, err := os.Open(newPath)
	if err != nil {
		// Restore backup on failure
		os.Rename(backupPath, currentPath)
		return fmt.Errorf("opening new binary: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(currentPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		os.Rename(backupPath, currentPath)
		return fmt.Errorf("creating new binary: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(currentPath)
		os.Rename(backupPath, currentPath)
		return fmt.Errorf("copying new binary: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)
	return nil
}

func checkWritable(dir string) error {
	tmp := filepath.Join(dir, ".clawide-write-test")
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	f.Close()
	os.Remove(tmp)
	return nil
}
