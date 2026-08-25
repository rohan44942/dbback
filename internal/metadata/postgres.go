package metadata

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

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
	ScheduleID  sql.NullString
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

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	GoogleID     string
	AuthProvider string
	CreatedAt    time.Time
}

func NewStore(databaseURL string) (*Store, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("database URL is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	s := &Store{db: db}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping() error {
	return s.db.Ping()
}

func (s *Store) migrate() error {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		data, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := s.db.Exec(string(data)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func (s *Store) AddBackup(b BackupMeta) (string, error) {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	var scheduleID interface{}
	if b.ScheduleID.Valid {
		scheduleID = b.ScheduleID.String
	}
	_, err := s.db.Exec(
		`INSERT INTO backups(id,name,type,storage_path,started_at,finished_at,size,schedule_id)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		b.ID, b.Name, b.Type, b.StoragePath, b.StartedAt, b.FinishedAt, b.Size, scheduleID,
	)
	if err != nil {
		return "", err
	}
	return b.ID, nil
}

func (s *Store) ListBackups(limit int) ([]BackupMeta, error) {
	rows, err := s.db.Query(
		`SELECT id,name,type,storage_path,started_at,finished_at,size,schedule_id
		 FROM backups ORDER BY finished_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBackups(rows)
}

func (s *Store) GetBackup(id string) (BackupMeta, error) {
	row := s.db.QueryRow(
		`SELECT id,name,type,storage_path,started_at,finished_at,size,schedule_id
		 FROM backups WHERE id = $1`, id)
	return scanBackupRow(row)
}

func (s *Store) ListBackupsBySchedule(scheduleID string) ([]BackupMeta, error) {
	rows, err := s.db.Query(
		`SELECT id,name,type,storage_path,started_at,finished_at,size,schedule_id
		 FROM backups WHERE schedule_id = $1 ORDER BY finished_at DESC`, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBackups(rows)
}

func (s *Store) DeleteBackup(id string) error {
	_, err := s.db.Exec(`DELETE FROM backups WHERE id = $1`, id)
	return err
}

func (s *Store) AddSchedule(sc Schedule) (string, error) {
	if sc.ID == "" {
		sc.ID = uuid.New().String()
	}
	var lastRun interface{}
	if sc.LastRun.Valid {
		lastRun = sc.LastRun.Time
	}
	_, err := s.db.Exec(
		`INSERT INTO schedules(id,db_type,source,cron_expr,retention_days,created_at,last_run)
		 VALUES($1,$2,$3,$4,$5,$6,$7)`,
		sc.ID, sc.DBType, sc.Source, sc.CronExpr, sc.RetentionDays, sc.CreatedAt, lastRun,
	)
	if err != nil {
		return "", err
	}
	return sc.ID, nil
}

func (s *Store) ListSchedules() ([]Schedule, error) {
	rows, err := s.db.Query(
		`SELECT id,db_type,source,cron_expr,retention_days,created_at,last_run
		 FROM schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func (s *Store) GetSchedule(id string) (Schedule, error) {
	row := s.db.QueryRow(
		`SELECT id,db_type,source,cron_expr,retention_days,created_at,last_run
		 FROM schedules WHERE id = $1`, id)
	return scanScheduleRow(row)
}

func (s *Store) UpdateScheduleLastRun(id string, t time.Time) error {
	_, err := s.db.Exec(`UPDATE schedules SET last_run = $1 WHERE id = $2`, t, id)
	return err
}

func (s *Store) UpdateSchedule(sc Schedule) error {
	_, err := s.db.Exec(
		`UPDATE schedules SET db_type = $1, source = $2, cron_expr = $3, retention_days = $4 WHERE id = $5`,
		sc.DBType, sc.Source, sc.CronExpr, sc.RetentionDays, sc.ID,
	)
	return err
}

func (s *Store) DeleteSchedule(id string) error {
	_, err := s.db.Exec(`DELETE FROM schedules WHERE id = $1`, id)
	return err
}

func scanBackups(rows *sql.Rows) ([]BackupMeta, error) {
	res := make([]BackupMeta, 0)
	for rows.Next() {
		b, err := scanBackupRows(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, b)
	}
	return res, rows.Err()
}

func scanBackupRows(rows *sql.Rows) (BackupMeta, error) {
	var b BackupMeta
	if err := rows.Scan(&b.ID, &b.Name, &b.Type, &b.StoragePath, &b.StartedAt, &b.FinishedAt, &b.Size, &b.ScheduleID); err != nil {
		return BackupMeta{}, err
	}
	return b, nil
}

func scanBackupRow(row *sql.Row) (BackupMeta, error) {
	var b BackupMeta
	if err := row.Scan(&b.ID, &b.Name, &b.Type, &b.StoragePath, &b.StartedAt, &b.FinishedAt, &b.Size, &b.ScheduleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BackupMeta{}, sql.ErrNoRows
		}
		return BackupMeta{}, err
	}
	return b, nil
}

func scanSchedules(rows *sql.Rows) ([]Schedule, error) {
	res := make([]Schedule, 0)
	for rows.Next() {
		sc, err := scanScheduleRows(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, sc)
	}
	return res, rows.Err()
}

func scanScheduleRows(rows *sql.Rows) (Schedule, error) {
	var sc Schedule
	var lastRun sql.NullTime
	if err := rows.Scan(&sc.ID, &sc.DBType, &sc.Source, &sc.CronExpr, &sc.RetentionDays, &sc.CreatedAt, &lastRun); err != nil {
		return Schedule{}, err
	}
	sc.LastRun = lastRun
	return sc, nil
}

func scanScheduleRow(row *sql.Row) (Schedule, error) {
	var sc Schedule
	var lastRun sql.NullTime
	if err := row.Scan(&sc.ID, &sc.DBType, &sc.Source, &sc.CronExpr, &sc.RetentionDays, &sc.CreatedAt, &lastRun); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Schedule{}, sql.ErrNoRows
		}
		return Schedule{}, err
	}
	sc.LastRun = lastRun
	return sc, nil
}
