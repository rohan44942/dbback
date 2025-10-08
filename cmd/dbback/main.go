package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/rohan44942/dbback/internal/backup"
	"github.com/rohan44942/dbback/internal/metadata"
	"github.com/rohan44942/dbback/internal/storage"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: dbback <command> [options]\ncommands: backup, restore, list")
		os.Exit(1)
	}
	cmd := os.Args[1]
	os.MkdirAll("backups", 0755)
	os.MkdirAll("metadata", 0755)

	metaStore := metadata.NewFileStore(filepath.Join("metadata", "metadata.json"))
	localStore := storage.NewLocal("backups")

	switch cmd {
	case "backup":
		fs := flag.NewFlagSet("backup", flag.ExitOnError)
		typeFlag := fs.String("type", "sqlite", "db type: sqlite|mysql|postgres|mongo")
		source := fs.String("source", "", "source connection string or file path")
		name := fs.String("name", "", "friendly name")
		fs.Parse(os.Args[2:])
		if *source == "" {
			log.Fatal("--source required")
		}
		id, err := backup.RunBackup(*typeFlag, *source, *name, localStore, metaStore)
		if err != nil {
			log.Fatalf("backup failed: %v", err)
		}
		fmt.Println("backup id:", id)
	case "list":
		list := metaStore.List()
		for _, b := range list {
			fmt.Printf("ID: %s  Name: %s  Type: %s  Path: %s  Time: %s  Size: %d bytes\n", b.ID, b.Name, b.Type, b.StoragePath, b.StartedAt.Format(time.RFC3339), b.Size)
		}
	case "restore":
		fs := flag.NewFlagSet("restore", flag.ExitOnError)
		id := fs.String("id", "", "backup id")
		target := fs.String("target", "", "target path")
		fs.Parse(os.Args[2:])
		if *id == "" || *target == "" {
			log.Fatal("--id and --target required")
		}
		b, ok := metaStore.Get(*id)
		if !ok {
			log.Fatalf("backup id not found: %s", *id)
		}
		r, err := localStore.Open(b.StoragePath)
		if err != nil {
			log.Fatalf("open backup failed: %v", err)
		}
		defer r.Close()
		if err := backup.RestoreFromReader(r, *target); err != nil {
			log.Fatalf("restore failed: %v", err)
		}
		fmt.Println("restore complete")
	default:
		fmt.Println("unknown command", cmd)
		os.Exit(1)
	}
}
