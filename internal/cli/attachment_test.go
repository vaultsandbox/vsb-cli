package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vaultsandbox "github.com/vaultsandbox/client-go"
)

func TestDownloadAttachment(t *testing.T) {
	t.Run("saves file successfully", func(t *testing.T) {
		dir := t.TempDir()
		// Set the global attachmentDir for the test
		oldDir := attachmentDir
		attachmentDir = dir
		defer func() { attachmentDir = oldDir }()

		content := []byte("test attachment content")
		err := downloadAttachment("test.txt", content)
		require.NoError(t, err)

		// Verify file was saved
		saved, err := os.ReadFile(filepath.Join(dir, "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, content, saved)
	})

	t.Run("handles filename collision", func(t *testing.T) {
		dir := t.TempDir()
		oldDir := attachmentDir
		attachmentDir = dir
		defer func() { attachmentDir = oldDir }()

		// Create first file
		err := downloadAttachment("test.txt", []byte("first"))
		require.NoError(t, err)

		// Save with same name - should create test_1.txt
		err = downloadAttachment("test.txt", []byte("second"))
		require.NoError(t, err)

		// Verify both files exist
		assert.FileExists(t, filepath.Join(dir, "test.txt"))
		assert.FileExists(t, filepath.Join(dir, "test_1.txt"))

		// Verify content
		first, _ := os.ReadFile(filepath.Join(dir, "test.txt"))
		second, _ := os.ReadFile(filepath.Join(dir, "test_1.txt"))
		assert.Equal(t, "first", string(first))
		assert.Equal(t, "second", string(second))
	})

	t.Run("creates directory if needed", func(t *testing.T) {
		baseDir := t.TempDir()
		nestedDir := filepath.Join(baseDir, "nested", "path")
		oldDir := attachmentDir
		attachmentDir = nestedDir
		defer func() { attachmentDir = oldDir }()

		err := downloadAttachment("file.txt", []byte("content"))
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(nestedDir, "file.txt"))
	})

	t.Run("handles binary content", func(t *testing.T) {
		dir := t.TempDir()
		oldDir := attachmentDir
		attachmentDir = dir
		defer func() { attachmentDir = oldDir }()

		content := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		err := downloadAttachment("binary.dat", content)
		require.NoError(t, err)

		saved, _ := os.ReadFile(filepath.Join(dir, "binary.dat"))
		assert.Equal(t, content, saved)
	})

	t.Run("handles empty content", func(t *testing.T) {
		dir := t.TempDir()
		oldDir := attachmentDir
		attachmentDir = dir
		defer func() { attachmentDir = oldDir }()

		err := downloadAttachment("empty.txt", []byte{})
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(dir, "empty.txt"))
		saved, _ := os.ReadFile(filepath.Join(dir, "empty.txt"))
		assert.Empty(t, saved)
	})
}

func TestDownloadAllAttachments(t *testing.T) {
	t.Run("downloads multiple attachments", func(t *testing.T) {
		dir := t.TempDir()
		oldDir := attachmentDir
		attachmentDir = dir
		defer func() { attachmentDir = oldDir }()

		attachments := []vaultsandbox.Attachment{
			{Filename: "file1.txt", Content: []byte("content1")},
			{Filename: "file2.txt", Content: []byte("content2")},
			{Filename: "file3.pdf", Content: []byte("pdf content")},
		}

		err := downloadAllAttachments(attachments)
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(dir, "file1.txt"))
		assert.FileExists(t, filepath.Join(dir, "file2.txt"))
		assert.FileExists(t, filepath.Join(dir, "file3.pdf"))

		// Verify content
		content1, _ := os.ReadFile(filepath.Join(dir, "file1.txt"))
		content2, _ := os.ReadFile(filepath.Join(dir, "file2.txt"))
		assert.Equal(t, "content1", string(content1))
		assert.Equal(t, "content2", string(content2))
	})

	t.Run("handles empty attachment list", func(t *testing.T) {
		dir := t.TempDir()
		oldDir := attachmentDir
		attachmentDir = dir
		defer func() { attachmentDir = oldDir }()

		err := downloadAllAttachments([]vaultsandbox.Attachment{})
		require.NoError(t, err)

		// Directory should be empty (except for any temp files)
		entries, _ := os.ReadDir(dir)
		assert.Empty(t, entries)
	})

	t.Run("handles single attachment", func(t *testing.T) {
		dir := t.TempDir()
		oldDir := attachmentDir
		attachmentDir = dir
		defer func() { attachmentDir = oldDir }()

		attachments := []vaultsandbox.Attachment{
			{Filename: "single.txt", Content: []byte("single file")},
		}

		err := downloadAllAttachments(attachments)
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(dir, "single.txt"))
	})

	t.Run("handles same filename attachments", func(t *testing.T) {
		dir := t.TempDir()
		oldDir := attachmentDir
		attachmentDir = dir
		defer func() { attachmentDir = oldDir }()

		attachments := []vaultsandbox.Attachment{
			{Filename: "doc.txt", Content: []byte("first")},
			{Filename: "doc.txt", Content: []byte("second")},
		}

		err := downloadAllAttachments(attachments)
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(dir, "doc.txt"))
		assert.FileExists(t, filepath.Join(dir, "doc_1.txt"))
	})
}
