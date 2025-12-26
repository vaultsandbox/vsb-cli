package files

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetUniqueFilename returns a unique filename in the given directory.
// If the file already exists, it appends _1, _2, etc. before the extension.
func GetUniqueFilename(dir, name string) string {
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	ext := filepath.Ext(name)
	base := name[:len(name)-len(ext)]

	for i := 1; ; i++ {
		path = filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
	}
}

// SaveFile saves data to a file in the given directory, using a unique filename
// if the file already exists. Returns the final path used.
func SaveFile(dir, name string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	path := GetUniqueFilename(dir, name)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return path, nil
}
