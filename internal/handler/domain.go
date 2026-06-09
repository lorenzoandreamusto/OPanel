package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"opanel/internal/database"
	"opanel/internal/middleware"
	"opanel/internal/model"
	"opanel/internal/service"
)

var validDomainName = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$`)

type DomainHandler struct {
	domainService *service.DomainService
}

func NewDomainHandler(db *database.DB, templatesDir, nginxConfDir, phpVersion, phpFPMPoolDir, phpFPMSocketDir string, mariadb *service.MariaDBService) *DomainHandler {
	return &DomainHandler{
		domainService: service.NewDomainService(db, templatesDir, nginxConfDir, phpVersion, phpFPMPoolDir, phpFPMSocketDir, mariadb),
	}
}

func (h *DomainHandler) ListDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	isAdmin := claims.Role == "admin"
	domains, err := h.domainService.ListDomains(claims.UserID, isAdmin)
	if err != nil {
		http.Error(w, `{"error":"failed to list domains"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domains)
}

func (h *DomainHandler) GetDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, `{"error":"domain id required"}`, http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid domain id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	domain, err := h.domainService.GetDomain(id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	if claims.Role != "admin" && domain.OwnerID != claims.UserID {
		http.Error(w, `{"error":"domain not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain)
}

func (h *DomainHandler) CreateDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req model.CreateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"domain name is required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Name) > 253 {
		http.Error(w, `{"error":"domain name too long (max 253 characters)"}`, http.StatusBadRequest)
		return
	}

	if strings.Contains(req.Name, "..") {
		http.Error(w, `{"error":"domain name cannot contain consecutive dots"}`, http.StatusBadRequest)
		return
	}

	if !validDomainName.MatchString(req.Name) {
		http.Error(w, `{"error":"invalid domain name format"}`, http.StatusBadRequest)
		return
	}

	domain, err := h.domainService.CreateDomain(&req, claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(domain)
}

func (h *DomainHandler) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, `{"error":"domain id required"}`, http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid domain id"}`, http.StatusBadRequest)
		return
	}

	domain, err := h.domainService.DeleteDomain(id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain)
}

func (h *DomainHandler) UpdateDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, `{"error":"domain id required"}`, http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid domain id"}`, http.StatusBadRequest)
		return
	}

	var req model.UpdateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	domain, err := h.domainService.GetDomain(id)
	if err != nil {
		http.Error(w, `{"error":"domain not found"}`, http.StatusNotFound)
		return
	}

	if claims.Role != "admin" && domain.OwnerID != claims.UserID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	updated, err := h.domainService.UpdateDomain(id, &req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}
