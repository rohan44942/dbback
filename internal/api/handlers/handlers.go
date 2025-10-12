package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rohan44942/dbback/internal/metadata"
)

type Server struct {
	Store *metadata.Store
}

func NewServer(store *metadata.Store) *Server {
	return &Server{Store: store}
}

func (s *Server) Routes() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/api/backups", s.listBackups).Methods("GET")
	r.HandleFunc("/api/backups/{id}", s.getBackup).Methods("GET")
	r.HandleFunc("/api/schedules", s.listSchedules).Methods("GET")
	r.HandleFunc("/api/schedules", s.createSchedule).Methods("POST")
	return r
}

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
}
