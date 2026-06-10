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

	// Sprint 6 services
	fileHandler := handler.NewFileHandler(service.NewFileManagerService(s.cfg.Paths.VhostsDir))
	monitorHandler := handler.NewMonitorHandler(service.NewMonitoringService())
	terminalHandler := handler.NewTerminalHandler(service.NewTerminalService())

	backupsDir := filepath.Join(filepath.Dir(s.cfg.Paths.VhostsDir), "backups")
	backupHandler := handler.NewBackupHandler(service.NewBackupService(s.db.DB, backupsDir, s.cfg.Paths.VhostsDir))
	wordpressHandler := handler.NewWordPressHandler(service.NewWordPressService(s.db.DB, mariadbSvc, s.cfg.Paths.VhostsDir))

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

	// File Manager (authenticated)
	mux.HandleFunc("GET /api/files/{domain}", middleware.Auth(s.cfg.JWT.Secret, fileHandler.ListFiles))
	mux.HandleFunc("GET /api/files/{domain}/read", middleware.Auth(s.cfg.JWT.Secret, fileHandler.ReadFile))
	mux.HandleFunc("POST /api/files/{domain}/write", middleware.Auth(s.cfg.JWT.Secret, fileHandler.WriteFile))
	mux.HandleFunc("POST /api/files/{domain}/mkdir", middleware.Auth(s.cfg.JWT.Secret, fileHandler.CreateDir))
	mux.HandleFunc("DELETE /api/files/{domain}", middleware.Auth(s.cfg.JWT.Secret, fileHandler.Delete))
	mux.HandleFunc("PUT /api/files/{domain}/rename", middleware.Auth(s.cfg.JWT.Secret, fileHandler.Rename))
	mux.HandleFunc("POST /api/files/{domain}/upload", middleware.Auth(s.cfg.JWT.Secret, fileHandler.Upload))
	mux.HandleFunc("GET /api/files/{domain}/download", middleware.Auth(s.cfg.JWT.Secret, fileHandler.Download))

	// Monitoring (authenticated)
	mux.HandleFunc("GET /api/monitoring/stats", middleware.Auth(s.cfg.JWT.Secret, monitorHandler.GetStats))
	mux.HandleFunc("GET /api/monitoring/ws", monitorHandler.WebSocketStats)

	// Terminal (authenticated, admin only)
	mux.HandleFunc("GET /api/terminal/ws", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(terminalHandler.Connect)))

	// Backups (authenticated)
	mux.HandleFunc("GET /api/backups", middleware.Auth(s.cfg.JWT.Secret, backupHandler.ListBackups))
	mux.HandleFunc("POST /api/backups", middleware.Auth(s.cfg.JWT.Secret, backupHandler.CreateBackup))
	mux.HandleFunc("GET /api/backups/{id}/download", middleware.Auth(s.cfg.JWT.Secret, backupHandler.DownloadBackup))
	mux.HandleFunc("DELETE /api/backups/{id}", middleware.Auth(s.cfg.JWT.Secret, backupHandler.DeleteBackup))
	mux.HandleFunc("POST /api/backups/{id}/restore", middleware.Auth(s.cfg.JWT.Secret, backupHandler.RestoreBackup))

	// WordPress (authenticated)
	mux.HandleFunc("POST /api/wordpress/install", middleware.Auth(s.cfg.JWT.Secret, wordpressHandler.Install))

	// DNS (authenticated)
	dnsHandler := handler.NewDNSHandler(s.db)
	mux.HandleFunc("GET /api/dns/zones", middleware.Auth(s.cfg.JWT.Secret, dnsHandler.ListZones))
	mux.HandleFunc("GET /api/dns/zones/{id}", middleware.Auth(s.cfg.JWT.Secret, dnsHandler.GetZone))
	mux.HandleFunc("POST /api/dns/zones", middleware.Auth(s.cfg.JWT.Secret, dnsHandler.CreateZone))
	mux.HandleFunc("DELETE /api/dns/zones/{id}", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(dnsHandler.DeleteZone)))
	mux.HandleFunc("GET /api/dns/zones/{id}/records", middleware.Auth(s.cfg.JWT.Secret, dnsHandler.ListRecords))
	mux.HandleFunc("POST /api/dns/zones/{id}/records", middleware.Auth(s.cfg.JWT.Secret, dnsHandler.CreateRecord))
	mux.HandleFunc("PUT /api/dns/records/{recordId}", middleware.Auth(s.cfg.JWT.Secret, dnsHandler.UpdateRecord))
	mux.HandleFunc("DELETE /api/dns/records/{recordId}", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(dnsHandler.DeleteRecord)))

	// Mail (authenticated)
	mailHandler := handler.NewMailHandler(s.db)
	mux.HandleFunc("GET /api/mail/domains", middleware.Auth(s.cfg.JWT.Secret, mailHandler.ListMailDomains))
	mux.HandleFunc("GET /api/mail/domains/{id}", middleware.Auth(s.cfg.JWT.Secret, mailHandler.GetMailDomain))
	mux.HandleFunc("POST /api/mail/domains", middleware.Auth(s.cfg.JWT.Secret, mailHandler.CreateMailDomain))
	mux.HandleFunc("DELETE /api/mail/domains/{id}", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(mailHandler.DeleteMailDomain)))
	mux.HandleFunc("GET /api/mail/domains/{id}/accounts", middleware.Auth(s.cfg.JWT.Secret, mailHandler.ListMailAccounts))
	mux.HandleFunc("POST /api/mail/domains/{id}/accounts", middleware.Auth(s.cfg.JWT.Secret, mailHandler.CreateMailAccount))
	mux.HandleFunc("PUT /api/mail/accounts/{accountId}", middleware.Auth(s.cfg.JWT.Secret, mailHandler.UpdateMailAccount))
	mux.HandleFunc("DELETE /api/mail/accounts/{accountId}", middleware.Auth(s.cfg.JWT.Secret, middleware.RequireAdmin(mailHandler.DeleteMailAccount)))
	mux.HandleFunc("GET /api/mail/autoconfig/{domain}", mailHandler.GetAutoconfig)
	mux.HandleFunc("GET /api/mail/dkim/{domain}", middleware.Auth(s.cfg.JWT.Secret, mailHandler.GetDKIMRecord))

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
