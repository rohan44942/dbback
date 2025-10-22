package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type Local struct {
	Base string
}

func NewLocal(base string) *Local {
	os.MkdirAll(base, 0755)
	return &Local{Base: base}
}

// Save stores the content from the reader to a local file.
// It implements the backup.StorageAdapter interface.
func (l *Local) Save(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	path := filepath.Join(l.Base, objectName)
	// Ensure the directory for the object exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(f, reader)
	return path, err
}

// Open retrieves a reader for a local backup file.
// It implements the backup.StorageAdapter interface.
func (l *Local) Open(ctx context.Context, objectName string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(l.Base, objectName))
}
