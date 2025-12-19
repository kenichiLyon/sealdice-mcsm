package fs

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// FileReader handles file system operations.
type FileReader struct {
	BaseDir string
}

// NewFileReader creates a new FileReader.
func NewFileReader(baseDir string) *FileReader {
	return &FileReader{
		BaseDir: baseDir,
	}
}

// ReadQRCode reads a file from the given path (relative to BaseDir if relative) and returns its content as Base64.
func (f *FileReader) ReadQRCode(path string) (string, error) {
	fullPath := path
	if !filepath.IsAbs(path) && f.BaseDir != "" {
		fullPath = filepath.Join(f.BaseDir, path)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	// Simple check if it's likely an image or text.
	// For safety, we just return base64 encoded content.
	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}

// FileExists checks if a file exists.
func (f *FileReader) FileExists(path string) bool {
	fullPath := path
	if !filepath.IsAbs(path) && f.BaseDir != "" {
		fullPath = filepath.Join(f.BaseDir, path)
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
