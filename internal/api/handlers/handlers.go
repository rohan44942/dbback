package api

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rohan44942/dbback/internal/api/controllers"
	apimiddleware "github.com/rohan44942/dbback/internal/api/middleware"
	"github.com/rohan44942/dbback/internal/config"
	"github.com/rohan44942/dbback/internal/metadata"
	"github.com/rohan44942/dbback/internal/scheduler"
)

type Server struct {
	Store       *metadata.Store
	Scheduler   *scheduler.Scheduler
	Local       StorageAdapter
	S3          StorageAdapter
	CORSOrigins []string
}

type StorageAdapter interface {
	Open(context.Context, string) (io.ReadCloser, error)
	Save(context.Context, string, io.Reader, int64, string) (string, error)
	Delete(context.Context, string) error
}

func NewServer(store *metadata.Store, sched *scheduler.Scheduler, local, s3 StorageAdapter, cfg *config.AppConfig) *Server {
	return &Server{
		Store:       store,
		Scheduler:   sched,
		Local:       local,
		S3:          s3,
		CORSOrigins: config.CORSOrigins(cfg),
	}
}

func (s *Server) Routes() http.Handler {
	r := mux.NewRouter()

	backupCtrl := controllers.NewBackupController(s.Store, s.getStorageAdapter())
	scheduleCtrl := controllers.NewScheduleController(s.Store, s.Scheduler)
	authCtrl := controllers.NewAuthController(s.Store)
	healthCtrl := controllers.NewHealthController(s.Store)

	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return apimiddleware.RequireAuth(authCtrl.Secret, next)
	}

	r.HandleFunc("/health", healthCtrl.Check).Methods("GET")

	r.HandleFunc("/api/backups", requireAuth(backupCtrl.List)).Methods("GET")
	r.HandleFunc("/api/backups", requireAuth(backupCtrl.Create)).Methods("POST")
	r.HandleFunc("/api/backups/{id}", requireAuth(backupCtrl.Get)).Methods("GET")
	r.HandleFunc("/api/backups/{id}", requireAuth(backupCtrl.Delete)).Methods("DELETE")
	r.HandleFunc("/api/backups/{id}/download", requireAuth(backupCtrl.Download)).Methods("GET")
	r.HandleFunc("/api/backups/{id}/restore", requireAuth(backupCtrl.Restore)).Methods("POST")

	r.HandleFunc("/api/schedules", requireAuth(scheduleCtrl.List)).Methods("GET")
	r.HandleFunc("/api/schedules", requireAuth(scheduleCtrl.Create)).Methods("POST")
	r.HandleFunc("/api/schedules/{id}", requireAuth(scheduleCtrl.Update)).Methods("PUT")
	r.HandleFunc("/api/schedules/{id}", requireAuth(scheduleCtrl.Delete)).Methods("DELETE")

	r.HandleFunc("/api/auth/login", authCtrl.Login).Methods("POST")
	r.HandleFunc("/api/auth/register", authCtrl.Register).Methods("POST")
	r.HandleFunc("/api/auth/logout", authCtrl.Logout).Methods("POST")
	r.HandleFunc("/api/auth/refresh", requireAuth(authCtrl.Refresh)).Methods("POST")
	r.HandleFunc("/api/auth/me", requireAuth(authCtrl.Me)).Methods("GET")

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")
		if origin != "" && s.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else if len(s.CORSOrigins) == 1 {
			w.Header().Set("Access-Control-Allow-Origin", s.CORSOrigins[0])
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		r.ServeHTTP(w, req)
	})
}

func (s *Server) isAllowedOrigin(origin string) bool {
	origin = config.NormalizeOrigin(origin)
	for _, allowed := range s.CORSOrigins {
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

func (s *Server) getStorageAdapter() StorageAdapter {
	if s.S3 != nil {
		return s.S3
	}
	return s.Local
}
