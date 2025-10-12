package metadata

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

type FileStore struct {
	path string
	mu   sync.Mutex
	data []BackupMeta
}

func NewFileStore(path string) *FileStore {
	fs := &FileStore{path: path}
	fs.load()
	return fs
}

func (f *FileStore) load() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, err := os.Stat(f.path); os.IsNotExist(err) {
		f.data = []BackupMeta{}
		return
	}
	b, err := ioutil.ReadFile(f.path)
	if err != nil {
		panic(err)
	}
	if len(b) == 0 {
		f.data = []BackupMeta{}
		return
	}
	if err := json.Unmarshal(b, &f.data); err != nil {
		panic(err)
	}
}

func (f *FileStore) save() {
	b, _ := json.MarshalIndent(f.data, "", "  ")
	_ = os.WriteFile(f.path, b, 0644)
}

func (f *FileStore) Add(meta BackupMeta) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	meta.ID = uuid.New().String()
	f.data = append([]BackupMeta{meta}, f.data...)
	f.save()
	return meta.ID
}

func (f *FileStore) List() []BackupMeta {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]BackupMeta(nil), f.data...)
}

func (f *FileStore) Get(id string) (BackupMeta, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, b := range f.data {
		if b.ID == id {
			return b, true
		}
	}
	return BackupMeta{}, false
}

func (fs *FileStore) UpdateScheduleLastRun(scheduleID string, t time.Time) error {
	// abhi ke liye nil bhj rhe h bad me update kr denge schedule kaise kaise store krenge 
	// uske according 
	return nil
}
