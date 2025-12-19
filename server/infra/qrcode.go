package infra

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// QRCodeReader handles file system operations for QR codes.
type QRCodeReader struct {
	BaseDir string
}

// NewQRCodeReader creates a new QRCodeReader.
func NewQRCodeReader(baseDir string) *QRCodeReader {
	return &QRCodeReader{
		BaseDir: baseDir,
	}
}

// ReadQRCode reads a file from the given path (relative to BaseDir if relative) and returns its content as Base64.
func (f *QRCodeReader) ReadQRCode(path string) (string, error) {
	fullPath := path
	if !filepath.IsAbs(path) && f.BaseDir != "" {
		fullPath = filepath.Join(f.BaseDir, path)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}

// FileExists checks if a file exists.
func (f *QRCodeReader) FileExists(path string) bool {
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
