package main

import (
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
		source := fs.String("source", "", "source connection string or file path")
		name := fs.String("name", "", "friendly name")
		fs.Parse(os.Args[2:])
		if *source == "" {
			log.Fatal("--source required")
		}
		id, err := backup.RunBackup(*typeFlag, *source, *name, local, store)
		if err != nil {
			log.Fatalf("backup failed: %v", err)
		}
		fmt.Println("backup id:", id)
	case "list":
		list, err := store.ListBackups(100)
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
		r, err := local.Open(b.StoragePath)
		if err != nil {
			log.Fatalf("open backup failed: %v", err)
		}
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
		default:
			fmt.Println("unknown schedule op")
		}
	default:
		fmt.Println("unknown command", cmd)
		os.Exit(1)
	}
}
