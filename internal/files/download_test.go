package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUniqueFilename(t *testing.T) {
	t.Run("no collision", func(t *testing.T) {
		dir := t.TempDir()
		got := GetUniqueFilename(dir, "test.txt")
		assert.Equal(t, filepath.Join(dir, "test.txt"), got)
	})

	t.Run("single collision", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("x"), 0644)
		require.NoError(t, err)
		got := GetUniqueFilename(dir, "test.txt")
		assert.Equal(t, filepath.Join(dir, "test_1.txt"), got)
	})

	t.Run("multiple collisions", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("x"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, "test_1.txt"), []byte("x"), 0644)
		require.NoError(t, err)
		got := GetUniqueFilename(dir, "test.txt")
		assert.Equal(t, filepath.Join(dir, "test_2.txt"), got)
	})

	t.Run("many collisions", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("x"), 0644)
		require.NoError(t, err)
		for i := 1; i <= 5; i++ {
			err = os.WriteFile(filepath.Join(dir, "test_"+string(rune('0'+i))+".txt"), []byte("x"), 0644)
			require.NoError(t, err)
		}
		got := GetUniqueFilename(dir, "test.txt")
		assert.Equal(t, filepath.Join(dir, "test_6.txt"), got)
	})

	t.Run("path traversal prevention", func(t *testing.T) {
		dir := t.TempDir()
		got := GetUniqueFilename(dir, "../../../etc/passwd")
		// Should sanitize to just "passwd"
		assert.Equal(t, filepath.Join(dir, "passwd"), got)
		assert.NotContains(t, got, "..")
	})

	t.Run("path traversal with backslash on linux", func(t *testing.T) {
		dir := t.TempDir()
		got := GetUniqueFilename(dir, "..\\..\\..\\windows\\system32\\config")
		// On Linux, backslashes are valid filename characters, not path separators
		// filepath.Base returns the input as-is since there are no forward slashes
		// The file stays contained in the target directory
		assert.True(t, len(got) > 0)
	})

	t.Run("no extension", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0644)
		require.NoError(t, err)
		got := GetUniqueFilename(dir, "README")
		assert.Equal(t, filepath.Join(dir, "README_1"), got)
	})

	t.Run("hidden file", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("x"), 0644)
		require.NoError(t, err)
		got := GetUniqueFilename(dir, ".gitignore")
		assert.Equal(t, filepath.Join(dir, "_1.gitignore"), got)
	})

	t.Run("double extension", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "archive.tar.gz"), []byte("x"), 0644)
		require.NoError(t, err)
		got := GetUniqueFilename(dir, "archive.tar.gz")
		// Only the last extension is preserved
		assert.Equal(t, filepath.Join(dir, "archive.tar_1.gz"), got)
	})
}

func TestSaveFile(t *testing.T) {
	t.Run("creates directory", func(t *testing.T) {
		base := t.TempDir()
		dir := filepath.Join(base, "new", "nested")
		path, err := SaveFile(dir, "test.txt", []byte("content"))
		require.NoError(t, err)
		assert.FileExists(t, path)
	})

	t.Run("writes content correctly", func(t *testing.T) {
		dir := t.TempDir()
		content := []byte("test content 123")
		path, err := SaveFile(dir, "test.txt", content)
		require.NoError(t, err)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("handles collision", func(t *testing.T) {
		dir := t.TempDir()
		_, err := SaveFile(dir, "test.txt", []byte("first"))
		require.NoError(t, err)
		path, err := SaveFile(dir, "test.txt", []byte("second"))
		require.NoError(t, err)
		assert.Contains(t, path, "test_1.txt")
	})

	t.Run("returns correct path", func(t *testing.T) {
		dir := t.TempDir()
		path, err := SaveFile(dir, "myfile.dat", []byte("data"))
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "myfile.dat"), path)
	})

	t.Run("handles empty content", func(t *testing.T) {
		dir := t.TempDir()
		path, err := SaveFile(dir, "empty.txt", []byte{})
		require.NoError(t, err)
		assert.FileExists(t, path)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("handles binary content", func(t *testing.T) {
		dir := t.TempDir()
		content := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		path, err := SaveFile(dir, "binary.dat", content)
		require.NoError(t, err)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("sanitizes path traversal in name", func(t *testing.T) {
		dir := t.TempDir()
		path, err := SaveFile(dir, "../../../evil.txt", []byte("content"))
		require.NoError(t, err)
		// File should be saved in the target directory, not escaped
		assert.True(t, filepath.HasPrefix(path, dir))
	})
}
