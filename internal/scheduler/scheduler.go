package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rohan44942/dbback/internal/backup"
	"github.com/rohan44942/dbback/internal/metadata"
	"github.com/rohan44942/dbback/internal/storage"
)

type Scheduler struct {
	cron   *cron.Cron
	store  *metadata.Store
	s3     *storage.S3Adapter
	local  *storage.Local
	ctx    context.Context
	cancel context.CancelFunc
}

func New(store *metadata.Store, s3 *storage.S3Adapter, local *storage.Local) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		cron:   cron.New(cron.WithSeconds()),
		store:  store,
		s3:     s3,
		local:  local,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.cancel()
}

func (s *Scheduler) AddSchedule(sc metadata.Schedule) (cron.EntryID, error) {
	id, err := s.cron.AddFunc(sc.CronExpr, func() {
		log.Printf("running scheduled job %s for source %s", sc.ID, sc.Source)
		// call backup.RunBackup with configured params
		// for now we will use local store if available
		backupID, err := backup.RunBackup(sc.DBType, sc.Source, "scheduled-"+sc.ID, s.local, s.store)
		if err != nil {
			log.Printf("scheduled backup failed: %v", err)
			return
		}
		// update last run, later implement this fully
		_ = s.store.UpdateScheduleLastRun(sc.ID, time.Now())
		log.Printf("scheduled backup done: %s", backupID)
	})
	return id, err
}
