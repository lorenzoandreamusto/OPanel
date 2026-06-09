package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"opanel/internal/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type MonitorHandler struct {
	svc *service.MonitoringService
}

func NewMonitorHandler(svc *service.MonitoringService) *MonitorHandler {
	return &MonitorHandler{svc: svc}
}

func (h *MonitorHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(stats)
}

func (h *MonitorHandler) WebSocketStats(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats, err := h.svc.GetStats()
		if err != nil {
			continue
		}

		if err := conn.WriteJSON(stats); err != nil {
			return
		}
	}
}
