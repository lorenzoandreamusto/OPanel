package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"opanel/internal/database"
	"opanel/internal/middleware"
	"opanel/internal/model"
	"opanel/internal/service"
)

var validDBName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
var validDBUsername = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

type DatabaseHandler struct {
	db         *database.DB
	mariadbSvc *service.MariaDBService
}

func NewDatabaseHandler(db *database.DB) *DatabaseHandler {
	return &DatabaseHandler{
		db:         db,
		mariadbSvc: service.NewMariaDBService(db),
	}
}

func (h *DatabaseHandler) ListDatabases(w http.ResponseWriter, r *http.Request) {
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

	var rows *sql.Rows
	var err error

	if isAdmin {
		rows, err = h.db.Query("SELECT id, name, owner_id, created_at, updated_at FROM databases ORDER BY name")
	} else {
		rows, err = h.db.Query("SELECT id, name, owner_id, created_at, updated_at FROM databases WHERE owner_id = ? ORDER BY name", claims.UserID)
	}
	if err != nil {
		http.Error(w, `{"error":"failed to list databases"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var databases []model.Database
	for rows.Next() {
		var d model.Database
		if err := rows.Scan(&d.ID, &d.Name, &d.OwnerID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			http.Error(w, `{"error":"failed to scan database"}`, http.StatusInternalServerError)
			return
		}
		databases = append(databases, d)
	}

	if databases == nil {
		databases = []model.Database{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(databases)
}

func (h *DatabaseHandler) GetDatabase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, `{"error":"database id required"}`, http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid database id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var db model.Database
	err = h.db.QueryRow(
		"SELECT id, name, owner_id, created_at, updated_at FROM databases WHERE id = ?", id,
	).Scan(&db.ID, &db.Name, &db.OwnerID, &db.CreatedAt, &db.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"database not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"failed to fetch database"}`, http.StatusInternalServerError)
		return
	}

	if claims.Role != "admin" && db.OwnerID != claims.UserID {
		http.Error(w, `{"error":"database not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(db)
}

func (h *DatabaseHandler) CreateDatabase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req model.CreateDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"database name is required"}`, http.StatusBadRequest)
		return
	}

	if !validDBName.MatchString(req.Name) {
		http.Error(w, `{"error":"database name must start with a letter and contain only letters, numbers, and underscores"}`, http.StatusBadRequest)
		return
	}

	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM databases WHERE name = ?", req.Name).Scan(&count)
	if err != nil {
		http.Error(w, `{"error":"failed to check database"}`, http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, `{"error":"database already exists"}`, http.StatusConflict)
		return
	}

	if err := h.mariadbSvc.CreateDatabase(req.Name); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	result, err := h.db.Exec(
		"INSERT INTO databases (name, owner_id) VALUES (?, ?)",
		req.Name, claims.UserID,
	)
	if err != nil {
		h.mariadbSvc.DropDatabase(req.Name)
		http.Error(w, `{"error":"failed to track database"}`, http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	var db model.Database
	err = h.db.QueryRow(
		"SELECT id, name, owner_id, created_at, updated_at FROM databases WHERE id = ?", id,
	).Scan(&db.ID, &db.Name, &db.OwnerID, &db.CreatedAt, &db.UpdatedAt)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch created database"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(db)
}

func (h *DatabaseHandler) DeleteDatabase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, `{"error":"database id required"}`, http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid database id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var db model.Database
	err = h.db.QueryRow(
		"SELECT id, name, owner_id, created_at, updated_at FROM databases WHERE id = ?", id,
	).Scan(&db.ID, &db.Name, &db.OwnerID, &db.CreatedAt, &db.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"database not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"failed to fetch database"}`, http.StatusInternalServerError)
		return
	}

	if claims.Role != "admin" && db.OwnerID != claims.UserID {
		http.Error(w, `{"error":"database not found"}`, http.StatusNotFound)
		return
	}

	if err := h.mariadbSvc.DropDatabase(db.Name); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if _, err := h.db.Exec("DELETE FROM databases WHERE id = ?", id); err != nil {
		http.Error(w, `{"error":"failed to remove database tracking"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.MessageResponse{Message: "database deleted"})
}

func (h *DatabaseHandler) CreateDatabaseUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	dbIDStr := r.PathValue("id")
	if dbIDStr == "" {
		http.Error(w, `{"error":"database id required"}`, http.StatusBadRequest)
		return
	}

	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid database id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var dbRecord model.Database
	err = h.db.QueryRow(
		"SELECT id, name, owner_id FROM databases WHERE id = ?", dbID,
	).Scan(&dbRecord.ID, &dbRecord.Name, &dbRecord.OwnerID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"database not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"failed to fetch database"}`, http.StatusInternalServerError)
		return
	}

	if claims.Role != "admin" && dbRecord.OwnerID != claims.UserID {
		http.Error(w, `{"error":"database not found"}`, http.StatusNotFound)
		return
	}

	var req model.CreateDatabaseUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"username and password are required"}`, http.StatusBadRequest)
		return
	}

	if !validDBUsername.MatchString(req.Username) {
		http.Error(w, `{"error":"username must start with a letter and contain only letters, numbers, and underscores"}`, http.StatusBadRequest)
		return
	}

	if req.Privileges == "" {
		req.Privileges = "ALL PRIVILEGES"
	}

	host := "%"
	if err := h.mariadbSvc.CreateUser(req.Username, host, req.Password); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if err := h.mariadbSvc.GrantPrivileges(req.Username, host, dbRecord.Name, req.Privileges); err != nil {
		h.mariadbSvc.DropUser(req.Username, host)
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	result, err := h.db.Exec(
		"INSERT INTO database_users (username, host, database_id, privileges) VALUES (?, ?, ?, ?)",
		req.Username, host, dbID, req.Privileges,
	)
	if err != nil {
		http.Error(w, `{"error":"failed to track database user"}`, http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	var dbUser model.DatabaseUser
	err = h.db.QueryRow(
		"SELECT id, username, host, database_id, privileges, created_at FROM database_users WHERE id = ?", id,
	).Scan(&dbUser.ID, &dbUser.Username, &dbUser.Host, &dbUser.DatabaseID, &dbUser.Privileges, &dbUser.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch created database user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dbUser)
}

func (h *DatabaseHandler) DeleteDatabaseUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	dbIDStr := r.PathValue("id")
	if dbIDStr == "" {
		http.Error(w, `{"error":"database id required"}`, http.StatusBadRequest)
		return
	}

	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid database id"}`, http.StatusBadRequest)
		return
	}

	userIDStr := r.PathValue("userId")
	if userIDStr == "" {
		http.Error(w, `{"error":"user id required"}`, http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var dbRecord model.Database
	err = h.db.QueryRow(
		"SELECT id, name, owner_id FROM databases WHERE id = ?", dbID,
	).Scan(&dbRecord.ID, &dbRecord.Name, &dbRecord.OwnerID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"database not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"failed to fetch database"}`, http.StatusInternalServerError)
		return
	}

	if claims.Role != "admin" && dbRecord.OwnerID != claims.UserID {
		http.Error(w, `{"error":"database not found"}`, http.StatusNotFound)
		return
	}

	var dbUser model.DatabaseUser
	err = h.db.QueryRow(
		"SELECT id, username, host, database_id, privileges, created_at FROM database_users WHERE id = ? AND database_id = ?",
		userID, dbID,
	).Scan(&dbUser.ID, &dbUser.Username, &dbUser.Host, &dbUser.DatabaseID, &dbUser.Privileges, &dbUser.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"database user not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"failed to fetch database user"}`, http.StatusInternalServerError)
		return
	}

	if err := h.mariadbSvc.DropUser(dbUser.Username, dbUser.Host); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if _, err := h.db.Exec("DELETE FROM database_users WHERE id = ?", userID); err != nil {
		http.Error(w, `{"error":"failed to remove database user tracking"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.MessageResponse{Message: "database user deleted"})
}

func (h *DatabaseHandler) UpdateDatabaseUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	dbIDStr := r.PathValue("id")
	if dbIDStr == "" {
		http.Error(w, `{"error":"database id required"}`, http.StatusBadRequest)
		return
	}

	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid database id"}`, http.StatusBadRequest)
		return
	}

	userIDStr := r.PathValue("userId")
	if userIDStr == "" {
		http.Error(w, `{"error":"user id required"}`, http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var dbRecord model.Database
	err = h.db.QueryRow(
		"SELECT id, name, owner_id FROM databases WHERE id = ?", dbID,
	).Scan(&dbRecord.ID, &dbRecord.Name, &dbRecord.OwnerID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"database not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"failed to fetch database"}`, http.StatusInternalServerError)
		return
	}

	if claims.Role != "admin" && dbRecord.OwnerID != claims.UserID {
		http.Error(w, `{"error":"database not found"}`, http.StatusNotFound)
		return
	}

	var dbUser model.DatabaseUser
	err = h.db.QueryRow(
		"SELECT id, username, host, database_id, privileges, created_at FROM database_users WHERE id = ? AND database_id = ?",
		userID, dbID,
	).Scan(&dbUser.ID, &dbUser.Username, &dbUser.Host, &dbUser.DatabaseID, &dbUser.Privileges, &dbUser.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"database user not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"failed to fetch database user"}`, http.StatusInternalServerError)
		return
	}

	var req model.UpdateDatabaseUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Password == "" && req.Privileges == "" {
		http.Error(w, `{"error":"password or privileges is required"}`, http.StatusBadRequest)
		return
	}

	if req.Password != "" {
		if err := h.mariadbSvc.ChangePassword(dbUser.Username, dbUser.Host, req.Password); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	if req.Privileges != "" {
		if err := h.mariadbSvc.GrantPrivileges(dbUser.Username, dbUser.Host, dbRecord.Name, req.Privileges); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		if _, err := h.db.Exec("UPDATE database_users SET privileges = ? WHERE id = ?", req.Privileges, userID); err != nil {
			http.Error(w, `{"error":"failed to update tracked privileges"}`, http.StatusInternalServerError)
			return
		}
	}

	var updated model.DatabaseUser
	err = h.db.QueryRow(
		"SELECT id, username, host, database_id, privileges, created_at FROM database_users WHERE id = ?", userID,
	).Scan(&updated.ID, &updated.Username, &updated.Host, &updated.DatabaseID, &updated.Privileges, &updated.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch updated database user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}
