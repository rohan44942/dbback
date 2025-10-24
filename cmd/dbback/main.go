package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"time"

	"github.com/rohan44942/dbback/internal/backup"
	"github.com/rohan44942/dbback/internal/config"
	"github.com/rohan44942/dbback/internal/metadata"
	"github.com/rohan44942/dbback/internal/storage"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: dbback <command> [options]\ncommands: backup, restore, list, schedule")
		os.Exit(1)
	}
	cmd := os.Args[1]
	os.MkdirAll("backups", 0755)
	os.MkdirAll("metadata", 0755)

	// load config optionally
	cfgPath := "configs/config.yaml"
	if p := os.Getenv("DBBACK_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, _ := config.Load(cfgPath)

	store, _ := metadata.NewStore(cfg.DBPath)
	local := storage.NewLocal("backups")

	switch cmd {
	case "backup":
		fs := flag.NewFlagSet("backup", flag.ExitOnError)
		typeFlag := fs.String("type", "sqlite", "db type: sqlite|mysql|postgres|mongo")
		sourceFlag := fs.String("source", "", "source connection string or file path")
		nameFlag := fs.String("name", "", "friendly name for the backup (used for s3 destination)")
		destinationFlag := fs.String("destination", "s3", "backup destination: s3 (default) or local (download to stdout)")
		fs.Parse(os.Args[2:])

		if *sourceFlag == "" {
			log.Fatal("--source required")
		}

		switch *destinationFlag {
		case "s3":
			// This is the flow for storing the backup on your S3 service.
			var adapter backup.StorageAdapter

			if cfg != nil && cfg.Storage.Type == "s3" {
				s3Adapter, err := storage.NewS3Adapter(storage.S3Config{
					Endpoint:  cfg.Storage.Endpoint,
					AccessKey: cfg.Storage.AccessKey,
					SecretKey: cfg.Storage.SecretKey,
					Bucket:    cfg.Storage.Bucket,
					UseSSL:    cfg.Storage.UseSSL,
				})
				if err != nil {
					log.Printf("warning: failed to create s3 adapter, falling back to local storage: %v", err)
					adapter = local
				} else {
					log.Println("using s3 storage adapter")
					adapter = s3Adapter
				}
			} else {
				log.Println("s3 not configured, using local storage adapter")
				adapter = local
			}

			id, err := backup.RunBackup(*typeFlag, *sourceFlag, *nameFlag, adapter, store)
			if err != nil {
				log.Fatalf("backup failed: %v", err)
			}
			fmt.Println("backup id:", id)
		case "local":
			// This is the new flow to stream the backup to the user.
			// No metadata is stored, and no ID is returned.
			if err := backup.StreamBackup(*typeFlag, *sourceFlag, os.Stdout); err != nil {
				log.Fatalf("streaming backup failed: %v", err)
			}
		default:
			log.Fatalf("invalid destination: %s. Must be 's3' or 'local'", *destinationFlag)
		}
	case "list":
		list, err := store.ListBackups(100)
		// TODO: Handle error where store is nil if DBPath is not in config
		if err != nil {
			log.Fatalf("list failed: %v", err)
		}
		for _, b := range list {
			fmt.Printf("ID: %s  Name: %s  Type: %s  Path: %s  Time: %s  Size: %d bytes\n", b.ID, b.Name, b.Type, b.StoragePath, b.FinishedAt.Format(time.RFC3339), b.Size)
		}
	case "restore":
		fs := flag.NewFlagSet("restore", flag.ExitOnError)
		id := fs.String("id", "", "backup id")
		target := fs.String("target", "", "target path")
		fs.Parse(os.Args[2:])
		if *id == "" || *target == "" {
			log.Fatal("--id and --target required")
		}
		b, err := store.GetBackup(*id)
		if err != nil {
			log.Fatalf("backup not found: %v", err)
		}

		// Determine which adapter to use for restoring.
		var adapter backup.StorageAdapter
		if cfg != nil && cfg.Storage.Type == "s3" {
			s3Adapter, err := storage.NewS3Adapter(storage.S3Config{
				Endpoint:  cfg.Storage.Endpoint,
				AccessKey: cfg.Storage.AccessKey,
				SecretKey: cfg.Storage.SecretKey,
				Bucket:    cfg.Storage.Bucket,
				UseSSL:    cfg.Storage.UseSSL,
			})
			if err != nil {
				// If S3 fails to init, we assume the backup must be local.
				log.Printf("warning: could not init s3 adapter, assuming local backup: %v", err)
				adapter = local
			} else {
				adapter = s3Adapter
			}
		} else {
			adapter = local
		}

		r, err := adapter.Open(context.Background(), b.StoragePath)
		if err != nil {
			log.Fatalf("open backup failed: %v", err)
		}
		defer r.Close()
		if err := backup.RestoreFromReader(r, *target); err != nil {
			log.Fatalf("restore failed: %v", err)
		}
		fmt.Println("restore complete")
	case "schedule":
		if len(os.Args) < 3 {
			fmt.Println("usage: dbback schedule <add|list>")
			os.Exit(1)
		}
		op := os.Args[2]
		switch op {
		case "add":
			fs := flag.NewFlagSet("schedule add", flag.ExitOnError)
			dbType := fs.String("type", "sqlite", "db type")
			source := fs.String("source", "", "source")
			cronExpr := fs.String("cron", "0 2 * * *", "cron expression")
			ret := fs.Int("retain", cfg.Retention, "retention days")
			fs.Parse(os.Args[3:])
			if *source == "" {
				log.Fatal("--source required")
			}
			sc := metadata.Schedule{
				DBType:        *dbType,
				Source:        *source,
				CronExpr:      *cronExpr,
				RetentionDays: *ret,
				CreatedAt:     time.Now(),
			}
			id, err := store.AddSchedule(sc)
			if err != nil {
				log.Fatalf("add schedule failed: %v", err)
			}
			fmt.Println("schedule id:", id)
		case "list":
			schs, err := store.ListSchedules()
			if err != nil {
				log.Fatalf("list schedules failed: %v", err)
			}
			for _, s := range schs {
				fmt.Printf("ID:%s DB:%s Source:%s Cron:%s Retain:%d LastRun:%v\n", s.ID, s.DBType, s.Source, s.CronExpr, s.RetentionDays, s.LastRun)
			}
		case "remove":
			fs := flag.NewFlagSet("schedule remove", flag.ExitOnError)
			id := fs.String("id", "", "schedule id to remove")
			fs.Parse(os.Args[3:])
			if *id == "" {
				log.Fatal("--id required")
			}

			err := store.DeleteSchedule(*id)
			if err != nil {
				log.Fatalf("failed to remove schedule: %v", err)
			}
			fmt.Printf("schedule %s removed successfully\n", *id)

		default:
			fmt.Println("unknown schedule optiion", op)
			os.Exit(1)
		}
	default:
		fmt.Println("unknown command", cmd)
		os.Exit(1)
	}
}
