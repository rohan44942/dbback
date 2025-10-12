package metadata

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

type BackupMeta struct {
	ID          string
	Name        string
	Type        string
	StoragePath string
	StartedAt   time.Time
	FinishedAt  time.Time
	Size        int64
}

type Schedule struct {
	ID            string
	DBType        string
	Source        string
	CronExpr      string
	RetentionDays int
	CreatedAt     time.Time
	LastRun       sql.NullTime
}

func NewStore(sqlitePath string) (*Store, error) {
	db, err := sql.Open("sqlite3", sqlitePath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS backups (
			id TEXT PRIMARY KEY,
			name TEXT,
			type TEXT,
			storage_path TEXT,
			started_at INTEGER,
			finished_at INTEGER,
			size INTEGER
		);`,
		`CREATE INDEX IF NOT EXISTS idx_backups_finished_at ON backups(finished_at);`,
		`CREATE TABLE IF NOT EXISTS schedules (
			id TEXT PRIMARY KEY,
			db_type TEXT,
			source TEXT,
			cron_expr TEXT,
			retention_days INTEGER,
			created_at INTEGER,
			last_run INTEGER
		);`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AddBackup(b BackupMeta) (string, error) {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	_, err := s.db.Exec(
		`INSERT INTO backups(id,name,type,storage_path,started_at,finished_at,size) VALUES(?,?,?,?,?,?,?)`,
		b.ID, b.Name, b.Type, b.StoragePath, b.StartedAt.Unix(), b.FinishedAt.Unix(), b.Size,
	)
	if err != nil {
		return "", err
	}
	return b.ID, nil
}

func (s *Store) ListBackups(limit int) ([]BackupMeta, error) {
	rows, err := s.db.Query(`SELECT id,name,type,storage_path,started_at,finished_at,size FROM backups ORDER BY finished_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []BackupMeta
	for rows.Next() {
		var b BackupMeta
		var started, finished int64
		if err := rows.Scan(&b.ID, &b.Name, &b.Type, &b.StoragePath, &started, &finished, &b.Size); err != nil {
			return nil, err
		}
		b.StartedAt = time.Unix(started, 0)
		b.FinishedAt = time.Unix(finished, 0)
		res = append(res, b)
	}
	return res, nil
}

func (s *Store) GetBackup(id string) (BackupMeta, error) {
	row := s.db.QueryRow(`SELECT id,name,type,storage_path,started_at,finished_at,size FROM backups WHERE id = ?`, id)
	var b BackupMeta
	var started, finished int64
	if err := row.Scan(&b.ID, &b.Name, &b.Type, &b.StoragePath, &started, &finished, &b.Size); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BackupMeta{}, sql.ErrNoRows
		}
		return BackupMeta{}, err
	}
	b.StartedAt = time.Unix(started, 0)
	b.FinishedAt = time.Unix(finished, 0)
	return b, nil
}

func (s *Store) DeleteBackup(id string) error {
	_, err := s.db.Exec(`DELETE FROM backups WHERE id = ?`, id)
	return err
}

func (s *Store) AddSchedule(sc Schedule) (string, error) {
	if sc.ID == "" {
		sc.ID = uuid.New().String()
	}
	// Handle nullable last_run properly
	var lastRun interface{}
	if sc.LastRun.Valid {
		lastRun = sc.LastRun.Time.Unix()
	} else {
		lastRun = nil
	}

	_, err := s.db.Exec(
		`INSERT INTO schedules(id,db_type,source,cron_expr,retention_days,created_at,last_run) VALUES(?,?,?,?,?,?,?)`,
		sc.ID, sc.DBType, sc.Source, sc.CronExpr, sc.RetentionDays, sc.CreatedAt.Unix(), lastRun,
	)
	if err != nil {
		return "", err
	}
	return sc.ID, nil
}

func (s *Store) ListSchedules() ([]Schedule, error) {
	rows, err := s.db.Query(`SELECT id,db_type,source,cron_expr,retention_days,created_at,last_run FROM schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Schedule
	for rows.Next() {
		var sc Schedule
		var created int64
		var lastRun sql.NullInt64

		if err := rows.Scan(&sc.ID, &sc.DBType, &sc.Source, &sc.CronExpr, &sc.RetentionDays, &created, &lastRun); err != nil {
			return nil, err
		}

		sc.CreatedAt = time.Unix(created, 0)
		if lastRun.Valid {
			sc.LastRun = sql.NullTime{Time: time.Unix(lastRun.Int64, 0), Valid: true}
		} else {
			sc.LastRun = sql.NullTime{Valid: false}
		}

		res = append(res, sc)
	}
	return res, nil
}

func (s *Store) UpdateScheduleLastRun(id string, t time.Time) error {
	_, err := s.db.Exec(`UPDATE schedules SET last_run = ? WHERE id = ?`, t.Unix(), id)
	return err
}
