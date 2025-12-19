package infra

import (
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type FileAPI interface {
	FileStat(instanceAlias string, relPath string) (modTime time.Time, exists bool, err error)
	FileReadBase64(instanceAlias string, relPath string) (string, error)
}

// LocalFS implements FileAPI using local filesystem with a base directory layout:
// base/<alias>/** where QR file is located under that alias directory.
type LocalFS struct {
	Base string
}

func NewLocalFS(base string) *LocalFS { return &LocalFS{Base: base} }

func (l *LocalFS) resolve(alias, rel string) string {
	if l.Base == "" {
		return rel
	}
	return filepath.Join(l.Base, alias, rel)
}

func (l *LocalFS) FileStat(alias, rel string) (time.Time, bool, error) {
	p := l.resolve(alias, rel)
	info, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return info.ModTime(), !info.IsDir(), nil
}

func (l *LocalFS) FileReadBase64(alias, rel string) (string, error) {
	p := l.resolve(alias, rel)
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (l *LocalFS) FSFromPath(alias string) (fs.FS, error) {
	p := l.resolve(alias, ".")
	return os.DirFS(p), nil
}
