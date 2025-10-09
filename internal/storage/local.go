package storage

import (
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

func (l *Local) Save(name string, r io.Reader) (string, int64, error) {
	path := filepath.Join(l.Base, name)
	f, err := os.Create(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	return path, n, err
}

func (l *Local) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}
