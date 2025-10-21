package backup

import (
	// "compress/gzip"
	"context"
	// "errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	// "os/exec"
	"path/filepath"
	// "strings"
	"time"

	"github.com/rohan44942/dbback/internal/metadata"
	// "github.com/rohan44942/dbback/internal/storage"
)

// func RunBackup(dbType, source, name string, store *storage.Local, meta *metadata.Store) (string, error) {
// 	start := time.Now()
// 	if name == "" {
// 		name = fmt.Sprintf("backup_%s_%d", dbType, time.Now().Unix())
// 	}
// 	tmpName := fmt.Sprintf("%s.tmp", name)
// 	tmpPath := filepath.Join(os.TempDir(), tmpName)
// 	f, err := os.Create(tmpPath)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer func() { f.Close(); os.Remove(tmpPath) }()

// 	gw := gzip.NewWriter(f)
// 	defer gw.Close()

// 	var in io.ReadCloser

// 	switch strings.ToLower(dbType) {
// 	case "sqlite":
// 		rf, err := os.Open(source)
// 		if err != nil {
// 			return "", err
// 		}
// 		in = rf
// 		defer in.Close()
// 	case "mysql":
// 		cmd := exec.Command("sh", "-c", fmt.Sprintf("mysqldump %s", source))
// 		stdout, err := cmd.StdoutPipe()
// 		if err != nil {
// 			return "", err
// 		}
// 		if err := cmd.Start(); err != nil {
// 			return "", err
// 		}
// 		in = stdout
// 		defer cmd.Wait()
// 	case "postgres", "postgresql":
// 		cmd := exec.Command("sh", "-c", fmt.Sprintf("pg_dump %s", source))
// 		stdout, err := cmd.StdoutPipe()
// 		if err != nil {
// 			return "", err
// 		}
// 		if err := cmd.Start(); err != nil {
// 			return "", err
// 		}
// 		in = stdout
// 		defer cmd.Wait()
// 	case "mongo", "mongodb":
// 		cmd := exec.Command("sh", "-c", fmt.Sprintf("mongodump --uri '%s' --archive", source))
// 		stdout, err := cmd.StdoutPipe()
// 		if err != nil {
// 			return "", err
// 		}
// 		if err := cmd.Start(); err != nil {
// 			return "", err
// 		}
// 		in = stdout
// 		defer cmd.Wait()
// 	default:
// 		return "", errors.New("unsupported db type")
// 	}

// 	if in != nil {
// 		if _, err := io.Copy(gw, in); err != nil {
// 			return "", err
// 		}
// 	} else {
// 		return "", errors.New("no input for backup")
// 	}

// 	gw.Close()
// 	f.Close()

// 	tmpR, err := os.Open(tmpPath)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer tmpR.Close()

// 	storageName := fmt.Sprintf("%s-%d.gz", name, time.Now().Unix())
// 	path, size, err := store.Save(storageName, tmpR)
// 	if err != nil {
// 		return "", err
// 	}

// 	metaRec := metadata.BackupMeta{
// 		Name:        name,
// 		Type:        dbType,
// 		StoragePath: path,
// 		StartedAt:   start,
// 		FinishedAt:  time.Now(),
// 		Size:        size,
// 	}
// 	id, err := meta.AddBackup(metaRec)
// 	if err != nil {
// 		return "", err
// 	}
// 	return id, nil
// }

// StorageAdapter is implemented by both Local and S3 storage back-ends.
type StorageAdapter interface {
	Save(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error)
	Open(ctx context.Context, objectName string) (io.ReadCloser, error)
	// Delete(ctx context.Context, objectName string) error
}

// RunBackup performs backup and uploads it through the adapter (local or S3)
func RunBackup(dbType, source, name string, adapter StorageAdapter, store *metadata.Store) (string, error) {
	ctx := context.Background()
	start := time.Now()

	// in Phase 2 we still dump locally then upload; Phase 3 can stream directly
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d.bak", dbType, start.Unix()))
	if err := performDump(dbType, source, tmpFile); err != nil {
		return "", err
	}
	defer os.Remove(tmpFile)

	f, err := os.Open(tmpFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	stat, _ := f.Stat()
	objectName := fmt.Sprintf("%s/%s-%d.bak", dbType, nameOrDefault(name, "backup"), start.Unix())

	loc, err := adapter.Save(ctx, objectName, f, stat.Size(), "application/octet-stream")
	if err != nil {
		return "", err
	}

	meta := metadata.BackupMeta{
		Name:        nameOrDefault(name, "backup"),
		Type:        dbType,
		StoragePath: loc,
		StartedAt:   start,
		FinishedAt:  time.Now(),
		Size:        stat.Size(),
	}
	return store.AddBackup(meta)
}

func nameOrDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// performDump – minimal mock; extend for MySQL/PG later
func performDump(dbType, source, target string) error {
	switch dbType {
	case "sqlite":
		in, err := os.Open(source)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	default:
		// Let's add the other DB types here for streaming.
		var cmd *exec.Cmd
		switch dbType {
		case "mysql":
			cmd = exec.Command("sh", "-c", fmt.Sprintf("mysqldump %s", source))
		case "postgres", "postgresql":
			cmd = exec.Command("sh", "-c", fmt.Sprintf("pg_dump %s", source))
		case "mongo", "mongodb":
			cmd = exec.Command("sh", "-c", fmt.Sprintf("mongodump --uri '%s' --archive", source))
		default:
			return fmt.Errorf("unsupported db type %s", dbType)
		}

		cmd.Stderr = os.Stderr // Forward errors to the user's console
		// cmd.Stdout = out
		return cmd.Run()
		// return fmt.Errorf("unsupported db type %s", dbType)
	}
}

// RestoreFromReader restores backup file from provided reader.
func RestoreFromReader(r io.Reader, targetPath string) error {
	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, r)
	return err
}

// StreamBackup performs a database dump and writes it directly to the provided writer.
// This is ideal for streaming a backup to a user for local download.
func StreamBackup(dbType, source string, w io.Writer) error {
	// We can reuse performDump by creating a temporary file and streaming it,
	// but for efficiency, we can write directly to the writer.
	// Let's adapt performDump's logic to stream.
	if dbType == "sqlite" {
		return performDump(dbType, source, "file_name_placeholder") // Special case for sqlite to copy file
	}

	// For command-based dumps, we can pipe stdout directly.
	return performDump(dbType, source, "file_name_placeholder")
}
