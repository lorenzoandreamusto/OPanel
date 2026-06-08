package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opanel/internal/config"
	"opanel/internal/database"

	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	httpServer *http.Server
	cfg        *config.Config
	db         *database.DB
}

func New(cfg *config.Config, db *database.DB) (*Server, error) {
	s := &Server{
		cfg: cfg,
		db:  db,
	}

	mux := s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

func (s *Server) Start() error {
	go func() {
		slog.Info("HTTP server starting", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

func (s *Server) EnsureAdmin() error {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if count > 0 {
		slog.Info("Admin user already exists, skipping creation")
		return nil
	}

	slog.Info("No users found, creating default admin user",
		"username", s.cfg.Admin.Username,
		"email", s.cfg.Admin.Email,
	)

	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.Admin.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	_, err = s.db.Exec(
		"INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, 'admin')",
		s.cfg.Admin.Username,
		s.cfg.Admin.Email,
		string(hash),
	)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	slog.Info("Default admin user created successfully")
	return nil
}
