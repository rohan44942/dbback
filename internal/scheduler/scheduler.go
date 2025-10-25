package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rohan44942/dbback/internal/backup"
	"github.com/rohan44942/dbback/internal/metadata"
	"github.com/rohan44942/dbback/internal/notify"
	"github.com/rohan44942/dbback/internal/storage"
)

type Scheduler struct {
	cron            *cron.Cron
	store           *metadata.Store
	s3              *storage.S3Adapter // Can be nil if S3 is not configured
	local           *storage.Local     // Can be nil if local is not configured (though it's always NewLocal("backups"))
	slackWebhookURL string
	ctx             context.Context
	cancel          context.CancelFunc
	// Map to store cron.EntryID by metadata.Schedule.ID for easy removal
	logger           *slog.Logger
	scheduleEntryIDs map[string]cron.EntryID
}

func New(store *metadata.Store, s3 *storage.S3Adapter, local *storage.Local, slackWebhookURL string) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		cron:             cron.New(),
		store:            store,
		s3:               s3,
		local:            local,
		slackWebhookURL:  slackWebhookURL,
		ctx:              ctx,
		cancel:           cancel,
		logger:           slog.Default().With("component", "scheduler"),
		scheduleEntryIDs: make(map[string]cron.EntryID),
	}
}

func (s *Scheduler) Start() {
	s.cron.Start() // The fmt.Print here can cause messy log output.
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.cancel()
}

func (s *Scheduler) AddSchedule(sc metadata.Schedule) (cron.EntryID, error) {
	// Check if the schedule is already added to prevent duplicates
	if _, exists := s.scheduleEntryIDs[sc.ID]; exists {
		s.logger.Warn("schedule already added to cron, skipping", "schedule_id", sc.ID)
		return s.scheduleEntryIDs[sc.ID], nil // Return existing ID
	}

	id, err := s.cron.AddFunc(sc.CronExpr, func() {
		s.logger.Info("running scheduled job", "schedule_id", sc.ID, "source", sc.Source)
		// call backup.RunBackup with configured params
		// The adapter selection logic should ideally be more robust,
		// but for now, we pass s.s3 (which might be nil) and s.local.
		// The RunBackup function will handle which one to use.
		backupID, err := backup.RunBackup(sc.DBType, sc.Source, "scheduled-"+sc.ID, s.s3, s.store, s.slackWebhookURL)
		if err != nil {
			errMsg := fmt.Sprintf("scheduled backup failed for source %s (schedule %s)", sc.Source, sc.ID)
			s.logger.Error(errMsg, "error", err)
			_ = notify.SendSlackNotification(s.slackWebhookURL, fmt.Sprintf(":x: %s: %v", errMsg, err))
			return
		}
		// Update last run time in the metadata store
		_ = s.store.UpdateScheduleLastRun(sc.ID, time.Now())
		s.logger.Info("scheduled backup finished successfully", "schedule_id", sc.ID, "backup_id", backupID)
	})
	if err == nil {
		// Store the cron.EntryID so we can remove it later
		s.scheduleEntryIDs[sc.ID] = id
	}
	return id, err
}

func (s *Scheduler) RemoveSchedule(scheduleID string) {
	if entryID, ok := s.scheduleEntryIDs[scheduleID]; ok {
		s.cron.Remove(entryID)
		delete(s.scheduleEntryIDs, scheduleID)
		s.logger.Info("removed schedule from cron", "schedule_id", scheduleID)
	} else {
		s.logger.Warn("schedule not found in cron, nothing to remove", "schedule_id", scheduleID)
	}
}
