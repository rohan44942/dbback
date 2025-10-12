package retention

import (
	"context"
	"time"

	"github.com/rohan44942/dbback/internal/metadata"
)

func ApplyRetention(ctx context.Context, store *metadata.Store, adapter storageAdapter, keepDays int) error {
	backups, err := store.ListBackups(1000)
	if err != nil {
		return err
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	for _, b := range backups {
		if b.FinishedAt.Before(cutoff) {
			_ = adapter.Delete(ctx, b.StoragePath)
			_ = store.DeleteBackup(b.ID)
		}
	}
	return nil
}

type storageAdapter interface {
	Delete(ctx context.Context, path string) error
}
