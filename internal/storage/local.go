package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	objectName = l.normalizeObjectName(objectName)
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
	return objectName, err
}

// Open retrieves a reader for a local backup file.
// It implements the backup.StorageAdapter interface.
func (l *Local) Open(ctx context.Context, objectName string) (io.ReadCloser, error) {
	objectName = l.normalizeObjectName(objectName)
	return os.Open(filepath.Join(l.Base, objectName))
}

// Delete removes a file from the local storage.
func (l *Local) Delete(ctx context.Context, objectName string) error {
	objectName = l.normalizeObjectName(objectName)
	path := filepath.Join(l.Base, objectName)
	return os.Remove(path)
}

func (l *Local) normalizeObjectName(objectName string) string {
	cleanBase := filepath.Clean(l.Base)
	cleanObject := filepath.Clean(objectName)

	if rel, err := filepath.Rel(cleanBase, cleanObject); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	if strings.HasPrefix(cleanObject, cleanBase+string(os.PathSeparator)) {
		return strings.TrimPrefix(cleanObject, cleanBase+string(os.PathSeparator))
	}
	return objectName
}
