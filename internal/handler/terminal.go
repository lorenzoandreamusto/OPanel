package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"opanel/internal/service"
)

type TerminalHandler struct {
	svc *service.TerminalService
}

func NewTerminalHandler(svc *service.TerminalService) *TerminalHandler {
	return &TerminalHandler{svc: svc}
}

func (h *TerminalHandler) Connect(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	_, msg, err := conn.ReadMessage()
	if err != nil {
		return
	}

	var sizeMsg struct {
		Type   string `json:"type"`
		Width  uint16 `json:"width"`
		Height uint16 `json:"height"`
	}
	if err := json.Unmarshal(msg, &sizeMsg); err != nil {
		return
	}

	if sizeMsg.Width == 0 {
		sizeMsg.Width = 80
	}
	if sizeMsg.Height == 0 {
		sizeMsg.Height = 24
	}

	session, err := h.svc.CreateSession(sizeMsg.Width, sizeMsg.Height)
	if err != nil {
		conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}
	defer session.Close()

	done := make(chan struct{})

	// PTY -> WebSocket
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := session.Pty.Read(buf)
			if err != nil {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// WebSocket -> PTY
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				conn.Close()
				return
			}

			var resizeMsg struct {
				Type   string `json:"type"`
				Width  uint16 `json:"width"`
				Height uint16 `json:"height"`
			}
			if err := json.Unmarshal(msg, &resizeMsg); err == nil && resizeMsg.Type == "resize" {
				h.svc.Resize(session.Pty, resizeMsg.Width, resizeMsg.Height)
				continue
			}

			session.Pty.Write(msg)
		}
	}()

	<-done
}
