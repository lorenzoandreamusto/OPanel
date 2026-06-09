package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"opanel/internal/service"
)

type FileHandler struct {
	svc *service.FileManagerService
}

func NewFileHandler(svc *service.FileManagerService) *FileHandler {
	return &FileHandler{svc: svc}
}

func (h *FileHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/httpdocs"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	files, err := h.svc.ListFiles(domain, path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(files)
}

func (h *FileHandler) ReadFile(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	path := r.URL.Query().Get("path")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "path is required"})
		return
	}

	content, err := h.svc.ReadFile(domain, path)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"content": string(content), "path": path})
}

func (h *FileHandler) WriteFile(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "path is required"})
		return
	}

	if err := h.svc.WriteFile(domain, req.Path, []byte(req.Content)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "file saved"})
}

func (h *FileHandler) CreateDir(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.svc.CreateDir(domain, req.Path); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "directory created"})
}

func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	path := r.URL.Query().Get("path")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "path is required"})
		return
	}

	if err := h.svc.Delete(domain, path); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
}

func (h *FileHandler) Rename(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")

	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.svc.Rename(domain, req.OldPath, req.NewPath); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "renamed"})
}

func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")

	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "no file provided"})
		return
	}
	defer file.Close()

	uploadPath := r.FormValue("path")
	if uploadPath == "" {
		uploadPath = "/" + header.Filename
	}

	if err := h.svc.UploadFile(domain, uploadPath, file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "file uploaded", "path": uploadPath})
}

func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	path := r.URL.Query().Get("path")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "path is required"})
		return
	}

	content, err := h.svc.ReadFile(domain, path)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	parts := strings.Split(path, "/")
	filename := parts[len(parts)-1]

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(content)
}
