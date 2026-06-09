package server

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"opanel/internal/handler"
	"opanel/internal/middleware"
	"opanel/internal/service"
)

func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	authHandler := handler.NewAuthHandler(s.db, s.cfg.JWT.Secret, s.cfg.JWT.ExpiryHours)
	userHandler := handler.NewUserHandler(s.db)
	mariadbSvc := service.NewMariaDBService(s.db, s.cfg.MariaDB.SocketPath, s.cfg.MariaDB.Host, s.cfg.MariaDB.Port)
	domainHandler := handler.NewDomainHandler(s.db, s.cfg.Paths.TemplatesDir, s.cfg.Paths.NginxConfDir, s.cfg.System.PHPVersion, s.cfg.Paths.PHPFPMPoolDir, s.cfg.Paths.PHPFPMSocketDir, mariadbSvc)
	databaseHandler := handler.NewDatabaseHandler(s.db, mariadbSvc)

	// Public routes
	mux.HandleFunc("GET /api/health", handler.HealthCheck)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	// Authenticated routes
	mux.HandleFunc("POST /api/auth/logout", middleware.Auth(s.cfg.JWT.Secret, authHandler.Logout))
	mux.HandleFunc("GET /api/auth/me", middleware.Auth(s.cfg.JWT.Secret, authHandler.GetMe))

	// Admin routes
	mux.HandleFunc("GET /api/users", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(userHandler.ListUsers)))
	mux.HandleFunc("GET /api/users/{id}", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(userHandler.GetUser)))
	mux.HandleFunc("POST /api/users", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(userHandler.CreateUser)))
	mux.HandleFunc("PUT /api/users/{id}", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(userHandler.UpdateUser)))
	mux.HandleFunc("DELETE /api/users/{id}", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(userHandler.DeleteUser)))

	// Domain routes (authenticated)
	mux.HandleFunc("GET /api/domains", middleware.Auth(s.cfg.JWT.Secret, domainHandler.ListDomains))
	mux.HandleFunc("GET /api/domains/{id}", middleware.Auth(s.cfg.JWT.Secret, domainHandler.GetDomain))
	mux.HandleFunc("POST /api/domains", middleware.Auth(s.cfg.JWT.Secret, domainHandler.CreateDomain))
	mux.HandleFunc("PUT /api/domains/{id}", middleware.Auth(s.cfg.JWT.Secret, domainHandler.UpdateDomain))
	mux.HandleFunc("DELETE /api/domains/{id}", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(domainHandler.DeleteDomain)))

	// Database routes (authenticated)
	mux.HandleFunc("GET /api/databases", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.ListDatabases))
	mux.HandleFunc("GET /api/databases/{id}", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.GetDatabase))
	mux.HandleFunc("POST /api/databases", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.CreateDatabase))
	mux.HandleFunc("DELETE /api/databases/{id}", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.DeleteDatabase))
	mux.HandleFunc("GET /api/databases/{id}/users", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.ListDatabaseUsers))
	mux.HandleFunc("POST /api/databases/{id}/users", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.CreateDatabaseUser))
	mux.HandleFunc("DELETE /api/databases/{id}/users/{userId}", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.DeleteDatabaseUser))
	mux.HandleFunc("PUT /api/databases/{id}/users/{userId}", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.UpdateDatabaseUser))

	// SPA static file serving
	spaHandler := s.setupSPA()
	if spaHandler != nil {
		mux.Handle("/", spaHandler)
	}

	return mux
}

func (s *Server) setupSPA() http.Handler {
	staticDir := filepath.Join(filepath.Dir(s.cfg.Paths.TemplatesDir), "static")

	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		return nil
	}

	fsys := os.DirFS(staticDir)
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// If the path has a file extension, serve the file directly
		if ext := filepath.Ext(path); ext != "" {
			if _, err := fs.Stat(fsys, strings.TrimPrefix(path, "/")); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Try to serve the file
		if _, err := fs.Stat(fsys, strings.TrimPrefix(path, "/")); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// For all non-file paths, serve index.html (SPA fallback)
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
