package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rohan44942/dbback/internal/metadata"
	"github.com/rohan44942/dbback/internal/scheduler"
)

// ScheduleController handles schedule-related endpoints
type ScheduleController struct {
	Store     *metadata.Store
	Scheduler *scheduler.Scheduler
}

type createScheduleRequest struct {
	DBType        string `json:"db_type"`
	Source        string `json:"source"`
	CronExpr      string `json:"cron_expr"`
	RetentionDays int    `json:"retention_days"`
}

func NewScheduleController(store *metadata.Store, scheduler *scheduler.Scheduler) *ScheduleController {
	return &ScheduleController{
		Store:     store,
		Scheduler: scheduler,
	}
}

// GET /schedules
func (c *ScheduleController) List(w http.ResponseWriter, r *http.Request) {
	schedules, err := c.Store.ListSchedules()
	if err != nil {
		http.Error(w, "failed to list schedules", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedules)
}

// POST /schedules
func (c *ScheduleController) Create(w http.ResponseWriter, r *http.Request) {
	var req createScheduleRequest
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

	id, err := c.Store.AddSchedule(sc)
	if err != nil {
		http.Error(w, "failed to create schedule", http.StatusInternalServerError)
		return
	}

	fullSchedule, err := c.Store.GetSchedule(id)
	if err != nil {
		http.Error(w, "failed to load created schedule", http.StatusInternalServerError)
		return
	}
	if c.Scheduler != nil {
		if _, err := c.Scheduler.AddSchedule(fullSchedule); err != nil {
			http.Error(w, "failed to start schedule", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(fullSchedule)
}

// PUT /schedules/{id}
func (c *ScheduleController) Update(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement update schedule logic
	w.WriteHeader(http.StatusNotImplemented)
}

// DELETE /schedules/{id}
func (c *ScheduleController) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := c.Store.DeleteSchedule(id); err != nil {
		http.Error(w, "failed to delete schedule", http.StatusInternalServerError)
		return
	}
	if c.Scheduler != nil {
		c.Scheduler.RemoveSchedule(id)
	}
	w.WriteHeader(http.StatusNoContent)
}
