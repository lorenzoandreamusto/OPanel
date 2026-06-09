package handler

import (
	"encoding/json"
	"net/http"

	"opanel/internal/service"
)

type WordPressHandler struct {
	svc *service.WordPressService
}

func NewWordPressHandler(svc *service.WordPressService) *WordPressHandler {
	return &WordPressHandler{svc: svc}
}

func (h *WordPressHandler) Install(w http.ResponseWriter, r *http.Request) {
	var req service.InstallWordPressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.DomainName == "" || req.AdminUser == "" || req.AdminPass == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "domain_name, admin_user, admin_password are required"})
		return
	}

	result, err := h.svc.Install(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}
