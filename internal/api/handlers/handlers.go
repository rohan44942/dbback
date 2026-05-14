package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rohan44942/dbback/internal/api/controllers"
	"github.com/rohan44942/dbback/internal/metadata"
	"github.com/rohan44942/dbback/internal/scheduler"
)

type Server struct {
	Store     *metadata.Store
	Scheduler *scheduler.Scheduler
	Local     interface {
		Open(context.Context, string) (io.ReadCloser, error)
		Save(context.Context, string, io.Reader, int64, string) (string, error)
		Delete(context.Context, string) error
	}
	S3 interface {
		Open(context.Context, string) (io.ReadCloser, error)
		Save(context.Context, string, io.Reader, int64, string) (string, error)
		Delete(context.Context, string) error
	}
}

func NewServer(store *metadata.Store, sched *scheduler.Scheduler, local, s3 interface {
	Open(context.Context, string) (io.ReadCloser, error)
	Save(context.Context, string, io.Reader, int64, string) (string, error)
	Delete(context.Context, string) error
}) *Server {
	return &Server{Store: store, Scheduler: sched, Local: local, S3: s3}
}

func (s *Server) Routes() http.Handler {
	r := mux.NewRouter()

	backupCtrl := controllers.NewBackupController(s.Store, s.getStorageAdapter())
	scheduleCtrl := controllers.NewScheduleController(s.Store, s.Scheduler)
	authCtrl := controllers.NewAuthController()

	// Backups
	r.HandleFunc("/api/backups", backupCtrl.List).Methods("GET")
	r.HandleFunc("/api/backups/{id}", backupCtrl.Get).Methods("GET")
	r.HandleFunc("/api/backups/{id}", backupCtrl.Delete).Methods("DELETE")
	r.HandleFunc("/api/backups/{id}/download", backupCtrl.Download).Methods("GET")

	// Schedules
	r.HandleFunc("/api/schedules", scheduleCtrl.List).Methods("GET")
	r.HandleFunc("/api/schedules", scheduleCtrl.Create).Methods("POST")
	r.HandleFunc("/api/schedules/{id}", scheduleCtrl.Delete).Methods("DELETE")

	// Auth
	r.HandleFunc("/api/auth/login", authCtrl.Login).Methods("POST")
	r.HandleFunc("/api/auth/register", authCtrl.Register).Methods("POST")
	r.HandleFunc("/api/auth/logout", authCtrl.Logout).Methods("POST")
	r.HandleFunc("/api/auth/refresh", authCtrl.Refresh).Methods("POST")
	r.HandleFunc("/api/auth/me", authCtrl.Me).Methods("GET")

	// Wrap the router with a basic CORS middleware
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Allow your frontend origin (change "*" to "http://localhost:3000" for better security)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if req.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		r.ServeHTTP(w, req)
	})
}

// --- BACKUP HANDLERS ---
type createBackupReq struct {
	DBType string `json:"db_type"`
	Source string `json:"source"`
	Name   string `json:"name"`
}

// func (s *Server) createBackup(w http.ResponseWriter, r *http.Request) {
// 	var req createBackupReq
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "invalid body", http.StatusBadRequest)
// 		return
// 	}
// 	// Use local storage for now; extend for S3 if needed
// 	adapter := s.getStorageAdapter()
// 	backupID, err := s.runBackup(req.DBType, req.Source, req.Name, "", adapter)
// 	if err != nil {
// 		http.Error(w, "failed to create backup: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(map[string]string{"id": backupID})
// }

func (s *Server) deleteBackup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	err := s.Store.DeleteBackup(id)
	if err != nil {
		http.Error(w, "failed to delete backup", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) downloadBackup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	b, err := s.Store.GetBackup(id)
	if err != nil {
		http.Error(w, "backup not found", http.StatusNotFound)
		return
	}
	adapter := s.getStorageAdapter()
	rc, err := adapter.Open(r.Context(), b.StoragePath)
	if err != nil {
		http.Error(w, "failed to open backup file", http.StatusInternalServerError)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Disposition", "attachment; filename="+b.Name+".gz")
	w.Header().Set("Content-Type", "application/gzip")
	_, _ = io.Copy(w, rc)
}

// --- SCHEDULE HANDLERS ---
type updateScheduleReq struct {
	DBType        *string `json:"db_type,omitempty"`
	Source        *string `json:"source,omitempty"`
	CronExpr      *string `json:"cron_expr,omitempty"`
	RetentionDays *int    `json:"retention_days,omitempty"`
}

// func (s *Server) updateSchedule(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	id := vars["id"]
// 	sc, err := s.Store.GetSchedule(id)
// 	if err != nil {
// 		http.Error(w, "schedule not found", http.StatusNotFound)
// 		return
// 	}
// 	var req updateScheduleReq
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "invalid body", http.StatusBadRequest)
// 		return
// 	}
// 	if req.DBType != nil {
// 		sc.DBType = *req.DBType
// 	}
// 	if req.Source != nil {
// 		sc.Source = *req.Source
// 	}
// 	if req.CronExpr != nil {
// 		sc.CronExpr = *req.CronExpr
// 	}
// 	if req.RetentionDays != nil {
// 		sc.RetentionDays = *req.RetentionDays
// 	}
// 	// Remove and re-add schedule in scheduler
// 	if s.Scheduler != nil {
// 		s.Scheduler.RemoveSchedule(id)
// 	}
// 	err = s.Store.UpdateSchedule(sc)
// 	if err != nil {
// 		http.Error(w, "failed to update schedule", http.StatusInternalServerError)
// 		return
// 	}
// 	if s.Scheduler != nil {
// 		s.Scheduler.AddSchedule(sc)
// 	}
// 	json.NewEncoder(w).Encode(sc)
// }

// --- UTILITIES ---
// getStorageAdapter returns the preferred storage adapter (local for now)
func (s *Server) getStorageAdapter() interface {
	Open(context.Context, string) (io.ReadCloser, error)
	Save(context.Context, string, io.Reader, int64, string) (string, error)
	Delete(context.Context, string) error
} {
	if s.S3 != nil {
		return s.S3
	}
	return s.Local
}

// runBackup is a helper to invoke backup.RunBackup
// func (s *Server) runBackup(dbType, source, name, scheduleID string, adapter interface {
// 	Save(context.Context, string, io.Reader, int64, string) (string, error)
// }) (string, error) {
// 	return backup.RunBackup(dbType, source, name, scheduleID, adapter, s.Store, "")
// }

func (s *Server) listBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := s.Store.ListBackups(1000)
	if err != nil {
		http.Error(w, "failed to list backups", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(backups)
}

func (s *Server) getBackup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	b, err := s.Store.GetBackup(id)
	if err != nil {
		http.Error(w, "backup not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(b)
}

type createScheduleReq struct {
	DBType        string `json:"db_type"`
	Source        string `json:"source"`
	CronExpr      string `json:"cron_expr"`
	RetentionDays int    `json:"retention_days"`
}

func (s *Server) listSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := s.Store.ListSchedules()
	if err != nil {
		http.Error(w, "failed to list schedules", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(schedules)
}

func (s *Server) createSchedule(w http.ResponseWriter, r *http.Request) {
	var req createScheduleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	sc := metadata.Schedule{
		DBType:        req.DBType,
		Source:        req.Source,
		CronExpr:      req.CronExpr,
		RetentionDays: req.RetentionDays,
		CreatedAt:     time.Now(),
	}
	id, err := s.Store.AddSchedule(sc)
	if err != nil {
		http.Error(w, "failed to create schedule", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})

	// Also add it to the running scheduler instance
	if s.Scheduler != nil {
		// We need the full schedule object back from the store
		fullSchedule, _ := s.Store.GetSchedule(id) // Error handling omitted for brevity
		s.Scheduler.AddSchedule(fullSchedule)
	}
}

func (s *Server) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Remove from database
	err := s.Store.DeleteSchedule(id)
	if err != nil {
		http.Error(w, "failed to delete schedule from store", http.StatusInternalServerError)
		return
	}

	// Remove from running scheduler instance
	if s.Scheduler != nil {
		s.Scheduler.RemoveSchedule(id)
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content is appropriate for successful deletion
}
