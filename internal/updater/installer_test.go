package updater

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTarGz(t *testing.T, dir string, binaryName string, content []byte) string {
	t.Helper()
	archivePath := filepath.Join(dir, "test.tar.gz")

	f, err := os.Create(archivePath)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: binaryName,
		Mode: 0755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write(content)
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())

	return archivePath
}

func createChecksumFile(t *testing.T, dir string, archivePath string) string {
	t.Helper()

	hash, err := fileSHA256(archivePath)
	require.NoError(t, err)

	checksumsPath := filepath.Join(dir, "checksums.txt")
	content := hash + "  " + filepath.Base(archivePath) + "\n"
	require.NoError(t, os.WriteFile(checksumsPath, []byte(content), 0644))

	return checksumsPath
}

func TestExtractBinary(t *testing.T) {
	dir := t.TempDir()
	binaryContent := []byte("#!/bin/sh\necho hello")

	archivePath := createTestTarGz(t, dir, "clawide", binaryContent)

	destPath := filepath.Join(dir, "extracted-clawide")
	err := extractBinary(archivePath, destPath)
	require.NoError(t, err)

	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, binaryContent, got)

	info, err := os.Stat(destPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestExtractBinary_NestedPath(t *testing.T) {
	dir := t.TempDir()
	binaryContent := []byte("#!/bin/sh\necho nested")

	// Create archive with nested path
	archivePath := filepath.Join(dir, "nested.tar.gz")
	f, err := os.Create(archivePath)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: "clawide-v1.0.0/clawide",
		Mode: 0755,
		Size: int64(len(binaryContent)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write(binaryContent)
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())

	destPath := filepath.Join(dir, "extracted")
	err = extractBinary(archivePath, destPath)
	require.NoError(t, err)

	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, binaryContent, got)
}

func TestExtractBinary_NotFound(t *testing.T) {
	dir := t.TempDir()

	// Create archive without the clawide binary
	archivePath := createTestTarGz(t, dir, "other-file", []byte("not the binary"))

	destPath := filepath.Join(dir, "extracted")
	err := extractBinary(archivePath, destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestVerifyChecksum_Valid(t *testing.T) {
	dir := t.TempDir()

	archivePath := filepath.Join(dir, "test.tar.gz")
	content := []byte("test archive content")
	require.NoError(t, os.WriteFile(archivePath, content, 0644))

	checksumsPath := createChecksumFile(t, dir, archivePath)

	err := verifyChecksum(archivePath, checksumsPath)
	assert.NoError(t, err)
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	dir := t.TempDir()

	archivePath := filepath.Join(dir, "test.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, []byte("real content"), 0644))

	// Write checksums with wrong hash
	checksumsPath := filepath.Join(dir, "checksums.txt")
	content := "0000000000000000000000000000000000000000000000000000000000000000  test.tar.gz\n"
	require.NoError(t, os.WriteFile(checksumsPath, []byte(content), 0644))

	err := verifyChecksum(archivePath, checksumsPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

func TestVerifyChecksum_FileNotInChecksums(t *testing.T) {
	dir := t.TempDir()

	archivePath := filepath.Join(dir, "test.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, []byte("content"), 0644))

	checksumsPath := filepath.Join(dir, "checksums.txt")
	content := "abc123  other-file.tar.gz\n"
	require.NoError(t, os.WriteFile(checksumsPath, []byte(content), 0644))

	err := verifyChecksum(archivePath, checksumsPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no checksum found")
}

func TestFileSHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")
	content := []byte("hello world")
	require.NoError(t, os.WriteFile(path, content, 0644))

	got, err := fileSHA256(path)
	require.NoError(t, err)

	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])
	assert.Equal(t, expected, got)
}

func TestReplaceBinary(t *testing.T) {
	dir := t.TempDir()

	// Create "current" binary
	currentPath := filepath.Join(dir, "clawide")
	require.NoError(t, os.WriteFile(currentPath, []byte("old binary"), 0755))

	// Create "new" binary
	newPath := filepath.Join(dir, "clawide-new")
	require.NoError(t, os.WriteFile(newPath, []byte("new binary"), 0755))

	err := replaceBinary(currentPath, newPath)
	require.NoError(t, err)

	got, err := os.ReadFile(currentPath)
	require.NoError(t, err)
	assert.Equal(t, "new binary", string(got))

	// Backup should be cleaned up
	_, err = os.Stat(currentPath + ".old")
	assert.True(t, os.IsNotExist(err))
}

func TestReplaceBinary_NotWritable(t *testing.T) {
	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0555))
	defer os.Chmod(readOnlyDir, 0755) // restore for cleanup

	currentPath := filepath.Join(readOnlyDir, "clawide")
	newPath := filepath.Join(dir, "clawide-new")
	require.NoError(t, os.WriteFile(newPath, []byte("new"), 0755))

	err := replaceBinary(currentPath, newPath)
	assert.Error(t, err)
}

func TestIsDocker(t *testing.T) {
	// In test environment, /.dockerenv should not exist (unless actually in Docker)
	_, statErr := os.Stat("/.dockerenv")
	expected := statErr == nil
	assert.Equal(t, expected, IsDocker())
}

func TestIsInstalling(t *testing.T) {
	assert.False(t, IsInstalling())

	installing.Store(true)
	assert.True(t, IsInstalling())

	installing.Store(false)
	assert.False(t, IsInstalling())
}

func TestFindExpectedChecksum(t *testing.T) {
	dir := t.TempDir()
	checksumsPath := filepath.Join(dir, "checksums.txt")

	content := `abc123def456  clawide-v1.0.0-darwin-arm64.tar.gz
789xyz000111  clawide-v1.0.0-linux-amd64.tar.gz
`
	require.NoError(t, os.WriteFile(checksumsPath, []byte(content), 0644))

	hash, err := findExpectedChecksum(checksumsPath, "clawide-v1.0.0-darwin-arm64.tar.gz")
	require.NoError(t, err)
	assert.Equal(t, "abc123def456", hash)

	hash, err = findExpectedChecksum(checksumsPath, "clawide-v1.0.0-linux-amd64.tar.gz")
	require.NoError(t, err)
	assert.Equal(t, "789xyz000111", hash)

	_, err = findExpectedChecksum(checksumsPath, "nonexistent.tar.gz")
	assert.Error(t, err)
}
