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
	"github.com/rohan44942/dbback/internal/logger"
	"github.com/rohan44942/dbback/internal/metadata"
	"github.com/rohan44942/dbback/internal/scheduler"
	"github.com/rohan44942/dbback/internal/storage"
)

func main() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("failed to create logs directory: %v", err)
	}
	logger.Init("logs/controller.log")
	logger.Log.Info("Starting dbback controller")
	cfgPath := "configs/config.yaml"
	if p := os.Getenv("DBBACK_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		logger.Log.Error("failed to load config", "path", cfgPath, "error", err)
		os.Exit(1)
	}

	store, err := metadata.NewStore(cfg.DBPath)
	if err != nil {
		logger.Log.Error("failed to create metadata store", "path", cfg.DBPath, "error", err)
		os.Exit(1)
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
			logger.Log.Error("failed to create s3 adapter", "error", err)
			os.Exit(1)
		}
	}

	// Start the scheduler to run backup jobs
	sched := scheduler.New(store, s3Adapter, localAdapter, cfg.SlackWebhookURL)
	schedules, err := store.ListSchedules()
	if err != nil {
		logger.Log.Warn("could not load schedules to run", "error", err)
	}
	for _, sc := range schedules {
		sched.AddSchedule(sc)
	}
	sched.Start()
	logger.Log.Info("scheduler started", "schedules_loaded", len(schedules))

	srv := api.NewServer(store, sched)
	handler := srv.Routes()
	s := &http.Server{
		Addr:    cfg.Server.Addr,
		Handler: handler,
	}

	// graceful shutdown
	go func() {
		logger.Log.Info("starting http server", "address", s.Addr)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Log.Info("shutting down server and scheduler")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.Shutdown(ctx)
	sched.Stop()
}
