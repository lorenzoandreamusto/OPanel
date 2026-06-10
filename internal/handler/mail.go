package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"opanel/internal/database"
	"opanel/internal/middleware"
	"opanel/internal/model"
	"opanel/internal/service"
)

type MailHandler struct {
	mailService *service.MailService
}

func NewMailHandler(db *database.DB) *MailHandler {
	return &MailHandler{
		mailService: service.NewMailService(db),
	}
}

func (h *MailHandler) ListMailDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	domains, err := h.mailService.ListMailDomains()
	if err != nil {
		http.Error(w, `{"error":"failed to list mail domains"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domains)
}

func (h *MailHandler) GetMailDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid mail domain id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	domain, err := h.mailService.GetMailDomain(id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain)
}

func (h *MailHandler) CreateMailDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req model.CreateMailDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.DomainID == 0 {
		http.Error(w, `{"error":"domain_id is required"}`, http.StatusBadRequest)
		return
	}

	domain, err := h.mailService.CreateMailDomain(&req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(domain)
}

func (h *MailHandler) DeleteMailDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid mail domain id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil || claims.Role != "admin" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	if err := h.mailService.DeleteMailDomain(id); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "mail domain deleted"})
}

func (h *MailHandler) ListMailAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	domainID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid mail domain id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	accounts, err := h.mailService.ListMailAccounts(domainID)
	if err != nil {
		http.Error(w, `{"error":"failed to list mail accounts"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

func (h *MailHandler) CreateMailAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	domainID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid mail domain id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req model.CreateMailAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"username and password are required"}`, http.StatusBadRequest)
		return
	}

	if strings.Contains(req.Username, "@") {
		http.Error(w, `{"error":"username should not contain @ symbol"}`, http.StatusBadRequest)
		return
	}

	account, err := h.mailService.CreateMailAccount(domainID, &req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(account)
}

func (h *MailHandler) UpdateMailAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("accountId")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid account id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req model.UpdateMailAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	account, err := h.mailService.UpdateMailAccount(id, &req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

func (h *MailHandler) DeleteMailAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("accountId")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid account id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil || claims.Role != "admin" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	if err := h.mailService.DeleteMailAccount(id); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "mail account deleted"})
}

func (h *MailHandler) GetAutoconfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	domain := r.PathValue("domain")
	email := r.URL.Query().Get("email")
	if domain == "" {
		http.Error(w, `{"error":"domain is required"}`, http.StatusBadRequest)
		return
	}

	config, err := h.mailService.GetAutoconfig(domain, email)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func (h *MailHandler) GetDKIMRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	domain := r.PathValue("domain")
	if domain == "" {
		http.Error(w, `{"error":"domain is required"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	record, err := h.mailService.GetDKIMPublicKey(domain)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"domain":   domain,
		"selector": "default",
		"record":   record,
	})
}
