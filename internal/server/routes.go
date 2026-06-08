package server

import (
	"net/http"

	"opanel/internal/handler"
	"opanel/internal/middleware"
)

func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	authHandler := handler.NewAuthHandler(s.db, s.cfg.JWT.Secret)
	userHandler := handler.NewUserHandler(s.db)
	domainHandler := handler.NewDomainHandler(s.db)
	databaseHandler := handler.NewDatabaseHandler(s.db)

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
	mux.HandleFunc("POST /api/databases/{id}/users", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.CreateDatabaseUser))
	mux.HandleFunc("DELETE /api/databases/{id}/users/{userId}", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.DeleteDatabaseUser))
	mux.HandleFunc("PUT /api/databases/{id}/users/{userId}", middleware.Auth(s.cfg.JWT.Secret, databaseHandler.UpdateDatabaseUser))

	return mux
}
