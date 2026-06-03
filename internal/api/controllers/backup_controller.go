package controllers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rohan44942/dbback/internal/backup"
	"github.com/rohan44942/dbback/internal/metadata"
)

type Storage interface {
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Save(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error)
	Delete(ctx context.Context, path string) error
}

type BackupController struct {
	Store   *metadata.Store
	Storage Storage
}

type createBackupRequest struct {
	DBType string `json:"db_type"`
	Source string `json:"source"`
	Name   string `json:"name"`
}

type restoreBackupRequest struct {
	Target string `json:"target"`
}

func NewBackupController(store *metadata.Store, storage Storage) *BackupController {
	return &BackupController{
		Store:   store,
		Storage: storage,
	}
}

func (c *BackupController) List(w http.ResponseWriter, r *http.Request) {
	backups, err := c.Store.ListBackups(1000)
	if err != nil {
		http.Error(w, "failed to list backups", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

func (c *BackupController) Create(w http.ResponseWriter, r *http.Request) {
	var req createBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	req.DBType = strings.TrimSpace(req.DBType)
	req.Source = strings.TrimSpace(req.Source)
	req.Name = strings.TrimSpace(req.Name)
	if req.DBType == "" || req.Source == "" {
		http.Error(w, "db_type and source are required", http.StatusBadRequest)
		return
	}

	id, err := backup.RunBackup(req.DBType, req.Source, req.Name, "", c.Storage, c.Store, "")
	if err != nil {
		http.Error(w, "failed to create backup: "+err.Error(), http.StatusInternalServerError)
		return
	}
	created, err := c.Store.GetBackup(id)
	if err != nil {
		http.Error(w, "backup created but could not be loaded", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (c *BackupController) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	b, err := c.Store.GetBackup(id)
	if err != nil {
		http.Error(w, "backup not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
}

func (c *BackupController) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	b, err := c.Store.GetBackup(id)
	if err != nil {
		http.Error(w, "backup not found", http.StatusNotFound)
		return
	}
	if err := c.Storage.Delete(r.Context(), b.StoragePath); err != nil {
		http.Error(w, "failed to delete backup file", http.StatusInternalServerError)
		return
	}
	if err := c.Store.DeleteBackup(id); err != nil {
		http.Error(w, "failed to delete backup", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *BackupController) Download(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
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

func (c *BackupController) Restore(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req restoreBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	req.Target = strings.TrimSpace(req.Target)
	if req.Target == "" {
		http.Error(w, "target is required", http.StatusBadRequest)
		return
	}

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

	if err := backup.RestoreFromReader(rc, req.Target); err != nil {
		http.Error(w, "restore failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "restored", "target": req.Target})
}
