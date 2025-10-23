package backup

import (
	"compress/gzip"
	"context"

	// "errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	// "os/exec"

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
	// We will use a pipe to stream the dump through a gzip compressor directly to the storage adapter.
	// This avoids creating a temporary file on disk.
	pr, pw := io.Pipe()
	gw := gzip.NewWriter(pw)

	var dumpErr error
	go func() {
		defer pw.Close()
		defer gw.Close()
		dumpErr = performDump(dbType, source, gw)
	}()

	// The object name should reflect that it's compressed.
	objectName := fmt.Sprintf("%s/%s-%d.gz", dbType, nameOrDefault(name, "backup"), start.Unix())
	// objectName := fmt.Sprintf("%s/%s-%d.bak", dbType, nameOrDefault(name, "backup"), start.Unix())

	// The size is unknown for a stream, so we pass -1.
	// The storage adapter (e.g., S3) will handle this.
	loc, err := adapter.Save(ctx, objectName, pr, -1, "application/gzip")
	fmt.Print("logging the file path: ", loc)
	if err != nil {
		return "", fmt.Errorf("failed to save to storage: %w", err)
	}

	// Check for any errors from the dump goroutine.
	if dumpErr != nil {
		return "", fmt.Errorf("database dump failed: %w", dumpErr)
	}

	// Since we streamed the data, we don't know the final size beforehand.
	// We can either get it from the storage adapter's response (if available)
	// or perform a Stat call. For now, we'll leave it as 0.
	// A better solution would be to update the StorageAdapter interface
	// to return the size.
	// For now, let's assume size is not critical for the metadata record.
	var finalSize int64 = 0

	meta := metadata.BackupMeta{
		Name:        nameOrDefault(name, "backup"),
		Type:        dbType,
		StoragePath: loc,
		StartedAt:   start,
		FinishedAt:  time.Now(),
		Size:        finalSize, // Size of the *compressed* file.
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
func performDump(dbType, source string, writer io.Writer) error {
	switch dbType {
	case "sqlite":
		in, err := os.Open(source)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(writer, in)
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
		cmd.Stdout = writer
		return cmd.Run()
	}
}

// RestoreFromReader restores backup file from provided reader.
func RestoreFromReader(r io.Reader, targetPath string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return err // This is where "gzip: invalid header" would come from.
	}
	defer gr.Close()

	// Ensure the target directory exists before creating the file.
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, gr)
	return err
}

// StreamBackup performs a database dump and writes it directly to the provided writer.
// This is ideal for streaming a backup to a user for local download.
func StreamBackup(dbType, source string, w io.Writer) error {
	// We can reuse performDump by creating a temporary file and streaming it,
	// but for efficiency, we can write directly to the writer.
	// Let's adapt performDump's logic to stream.
	if dbType == "sqlite" {
		return performDump(dbType, source, w) // Special case for sqlite to copy file
	}

	// For command-based dumps, we can pipe stdout directly.
	return performDump(dbType, source, w)
}
