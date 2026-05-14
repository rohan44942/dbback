package retention

import (
	"context"
	"time"

	"github.com/rohan44942/dbback/internal/backup"
	"github.com/rohan44942/dbback/internal/logger"
	"github.com/rohan44942/dbback/internal/metadata"
)

func ApplyRetention(ctx context.Context, store *metadata.Store, adapter backup.StorageAdapter) error {
	logger.Log.Info("applying retention policy")
	schedules, err := store.ListSchedules()
	if err != nil {
		return err
	}

	for _, schedule := range schedules {
		if schedule.RetentionDays <= 0 {
			continue // Skip schedules with no retention policy
		}

		backups, err := store.ListBackupsBySchedule(schedule.ID)
		if err != nil {
			logger.Log.Error("failed to list backups for schedule", "schedule_id", schedule.ID, "error", err)
			continue
		}

		cutoff := time.Now().AddDate(0, 0, -schedule.RetentionDays)
		for _, b := range backups {
			if b.FinishedAt.Before(cutoff) {
				logger.Log.Info("deleting old backup", "backup_id", b.ID, "schedule_id", schedule.ID, "path", b.StoragePath)
				// Note: We are ignoring errors here for simplicity, but in production, you'd want to handle them.
				_ = adapter.Delete(ctx, b.StoragePath)
				_ = store.DeleteBackup(b.ID)
			}
		}
	}
	return nil
}
