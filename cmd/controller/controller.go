package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "github.com/rohan44942/dbback/internal/api/handlers"
	"github.com/rohan44942/dbback/internal/config"
	"github.com/rohan44942/dbback/internal/metadata"
	"github.com/rohan44942/dbback/internal/scheduler"
	"github.com/rohan44942/dbback/internal/storage"
)

func main() {
	cfgPath := "configs/config.yaml"
	if p := os.Getenv("DBBACK_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	store, err := metadata.NewStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("metadata store: %v", err)
	}

	var s3Adapter *storage.S3Adapter
	localAdapter := storage.NewLocal("backups")
	if cfg.Storage.Type == "s3" {
		s3Cfg := storage.S3Config{
			Endpoint:  cfg.Storage.Endpoint,
			AccessKey: cfg.Storage.AccessKey,
			SecretKey: cfg.Storage.SecretKey,
			Bucket:    cfg.Storage.Bucket,
			UseSSL:    cfg.Storage.UseSSL,
		}
		s3Adapter, err = storage.NewS3Adapter(s3Cfg)
		if err != nil {
			log.Fatalf("s3 adapter: %v", err)
		}
	}

	// Start the scheduler to run backup jobs
	sched := scheduler.New(store, s3Adapter, localAdapter)
	schedules, err := store.ListSchedules()
	if err != nil {
		log.Printf("could not load schedules to run: %v", err)
	}
	for _, sc := range schedules {
		log.Printf("before the add schedule of scheduler")
		sched.AddSchedule(sc)
	}
	sched.Start()
	log.Printf("scheduler started, loaded %d schedules", len(schedules))

	srv := api.NewServer(store, sched)
	handler := srv.Routes()
	s := &http.Server{
		Addr:    cfg.Server.Addr,
		Handler: handler,
	}

	// graceful shutdown
	go func() {
		log.Printf("starting controller at %s", cfg.Server.Addr)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.Shutdown(ctx)
	sched.Stop()
}
