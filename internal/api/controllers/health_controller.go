package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/rohan44942/dbback/internal/metadata"
)

type HealthController struct {
	Store *metadata.Store
}

func NewHealthController(store *metadata.Store) *HealthController {
	return &HealthController{Store: store}
}

func (c *HealthController) Check(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	dbStatus := "ok"
	if err := c.Store.Ping(); err != nil {
		dbStatus = "error"
		status = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": status,
		"db":     dbStatus,
	})
}
