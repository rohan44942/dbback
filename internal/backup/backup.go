package backup

import (
    "compress/gzip"
    "errors"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/rohan44942/dbback/internal/metadata"
    "github.com/rohan44942/dbback/internal/storage"
)

func RunBackup(dbType, source, name string, store *storage.Local, meta *metadata.FileStore) (string, error) {
    start := time.Now()
    if name == "" {
        name = fmt.Sprintf("backup_%s_%d", dbType, time.Now().Unix())
    }
    tmpName := fmt.Sprintf("%s.tmp", name)
    tmpPath := filepath.Join(os.TempDir(), tmpName)
    f, err := os.Create(tmpPath)
    if err != nil {
        return "", err
    }
    defer func() { f.Close(); os.Remove(tmpPath) }()

    gw := gzip.NewWriter(f)
    defer gw.Close()

    var in io.ReadCloser

    switch strings.ToLower(dbType) {
    case "sqlite":
        rf, err := os.Open(source)
        if err != nil {
            return "", err
        }
        in = rf
        defer in.Close()
    case "mysql":
        cmd := exec.Command("sh", "-c", fmt.Sprintf("mysqldump %s", source))
        stdout, err := cmd.StdoutPipe()
        if err != nil {
            return "", err
        }
        if err := cmd.Start(); err != nil {
            return "", err
        }
        in = stdout
        defer cmd.Wait()
    case "postgres", "postgresql":
        cmd := exec.Command("sh", "-c", fmt.Sprintf("pg_dump %s", source))
        stdout, err := cmd.StdoutPipe()
        if err != nil {
            return "", err
        }
        if err := cmd.Start(); err != nil {
            return "", err
        }
        in = stdout
        defer cmd.Wait()
    case "mongo", "mongodb":
        cmd := exec.Command("sh", "-c", fmt.Sprintf("mongodump --uri '%s' --archive", source))
        stdout, err := cmd.StdoutPipe()
        if err != nil {
            return "", err
        }
        if err := cmd.Start(); err != nil {
            return "", err
        }
        in = stdout
        defer cmd.Wait()
    default:
        return "", errors.New("unsupported db type")
    }

    if in != nil {
        if _, err := io.Copy(gw, in); err != nil {
            return "", err
        }
    } else {
        return "", errors.New("no input for backup")
    }

    gw.Close()
    f.Close()

    tmpR, err := os.Open(tmpPath)
    if err != nil {
        return "", err
    }
    defer tmpR.Close()

    storageName := fmt.Sprintf("%s-%d.gz", name, time.Now().Unix())
    path, size, err := store.Save(storageName, tmpR)
    if err != nil {
        return "", err
    }

    metaRec := metadata.BackupMeta{
        Name:        name,
        Type:        dbType,
        StoragePath: path,
        StartedAt:   start,
        FinishedAt:  time.Now(),
        Size:        size,
    }
    id := meta.Add(metaRec)
    return id, nil
}

func RestoreFromReader(r io.ReadCloser, target string) error {
    defer r.Close()
    gr, err := gzip.NewReader(r)
    if err != nil {
        return err
    }
    defer gr.Close()
    out, err := os.Create(target)
    if err != nil {
        return err
    }
    defer out.Close()
    _, err = io.Copy(out, gr)
    return err
}