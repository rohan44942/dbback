package controllers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rohan44942/dbback/internal/metadata"
)

// Storage defines the interface for interacting with backup files
type Storage interface {
	Open(ctx context.Context, path string) (io.ReadCloser, error)
}

// BackupController handles backup-related endpoints
type BackupController struct {
	Store   *metadata.Store
	Storage Storage
}

func NewBackupController(store *metadata.Store, storage Storage) *BackupController {
	return &BackupController{
		Store:   store,
		Storage: storage,
	}
}

// GET /backups
func (c *BackupController) List(w http.ResponseWriter, r *http.Request) {
	backups, err := c.Store.ListBackups(1000)
	if err != nil {
		http.Error(w, "failed to list backups", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

// POST /backups
func (c *BackupController) Create(w http.ResponseWriter, r *http.Request) {
	// Logic to trigger manual backup
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "triggered"})
}

// GET /backups/{id}
func (c *BackupController) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	b, err := c.Store.GetBackup(id)
	if err != nil {
		http.Error(w, "backup not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
}

// DELETE /backups/{id}
func (c *BackupController) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	err := c.Store.DeleteBackup(id)
	if err != nil {
		http.Error(w, "failed to delete backup", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /backups/{id}/download
func (c *BackupController) Download(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	b, err := c.Store.GetBackup(id)
	if err != nil {
		http.Error(w, "backup not found", http.StatusNotFound)
		return
	}

	rc, err := c.Storage.Open(r.Context(), b.StoragePath)
	if err != nil {
		http.Error(w, "failed to open backup file", http.StatusInternalServerError)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Disposition", "attachment; filename="+b.Name+".gz")
	w.Header().Set("Content-Type", "application/gzip")
	_, _ = io.Copy(w, rc)
}
