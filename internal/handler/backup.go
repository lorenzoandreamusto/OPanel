package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"opanel/internal/service"
)

type BackupHandler struct {
	svc *service.BackupService
}

func NewBackupHandler(svc *service.BackupService) *BackupHandler {
	return &BackupHandler{svc: svc}
}

func (h *BackupHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	domainIDStr := r.URL.Query().Get("domain_id")
	domainID, _ := strconv.ParseInt(domainIDStr, 10, 64)

	backups, err := h.svc.ListBackups(domainID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if backups == nil {
		backups = []service.Backup{}
	}
	json.NewEncoder(w).Encode(backups)
}

func (h *BackupHandler) CreateBackup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DomainID   int64  `json:"domain_id"`
		DomainName string `json:"domain_name"`
		Name       string `json:"name,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	backup, err := h.svc.CreateBackup(req.DomainID, req.DomainName, req.Name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(backup)
}

func (h *BackupHandler) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	path, err := h.svc.GetBackupPath(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\"backup.tar.gz\"")
	http.ServeFile(w, r, path)
}

func (h *BackupHandler) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	if err := h.svc.DeleteBackup(id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "backup deleted"})
}

func (h *BackupHandler) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	var req struct {
		DomainName string `json:"domain_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.svc.RestoreBackup(id, req.DomainName); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "backup restored"})
}
