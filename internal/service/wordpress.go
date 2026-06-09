package service

import (
	"archive/zip"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type WordPressService struct {
	db          *sql.DB
	mariadbSvc  *MariaDBService
	VhostsDir   string
}

type WordPressInstall struct {
	ID        int64     `json:"id"`
	DomainID  int64     `json:"domain_id"`
	SiteName  string    `json:"site_name"`
	AdminUser string    `json:"admin_user"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type InstallWordPressRequest struct {
	DomainID   int64  `json:"domain_id"`
	DomainName string `json:"domain_name"`
	SiteName   string `json:"site_name"`
	AdminUser  string `json:"admin_user"`
	AdminPass  string `json:"admin_password"`
	AdminEmail string `json:"admin_email,omitempty"`
}

type InstallWordPressResponse struct {
	Install *WordPressInstall `json:"install"`
	Message string            `json:"message"`
}

func NewWordPressService(db *sql.DB, mariadbSvc *MariaDBService, vhostsDir string) *WordPressService {
	return &WordPressService{
		db:         db,
		mariadbSvc: mariadbSvc,
		VhostsDir:  vhostsDir,
	}
}

func generateSalt() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func generatePassword() string {
	b := make([]byte, 24)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (s *WordPressService) registerDatabase(name string, ownerID int64) (int64, error) {
	var exists int
	err := s.db.QueryRow("SELECT COUNT(*) FROM databases WHERE name = ?", name).Scan(&exists)
	if err != nil {
		return 0, err
	}
	if exists > 0 {
		var id int64
		err := s.db.QueryRow("SELECT id FROM databases WHERE name = ?", name).Scan(&id)
		return id, err
	}
	result, err := s.db.Exec("INSERT INTO databases (name, owner_id) VALUES (?, ?)", name, ownerID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *WordPressService) registerDatabaseUser(username, host, dbName, privileges string, dbID int64) error {
	var exists int
	err := s.db.QueryRow("SELECT COUNT(*) FROM database_users WHERE username = ? AND database_id = ?", username, dbID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	_, err = s.db.Exec(
		"INSERT INTO database_users (username, host, database_id, privileges) VALUES (?, ?, ?, ?)",
		username, host, dbID, privileges,
	)
	return err
}

func (s *WordPressService) Install(req InstallWordPressRequest) (*InstallWordPressResponse, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM wordpress_installs WHERE domain_id = ? AND status = 'completed'",
		req.DomainID,
	).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing install: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("wordpress already installed for this domain")
	}

	adminEmail := req.AdminEmail
	if adminEmail == "" {
		adminEmail = "admin@" + req.DomainName
	}

	result, err := s.db.Exec(
		"INSERT INTO wordpress_installs (domain_id, site_name, admin_user, status) VALUES (?, ?, ?, 'installing')",
		req.DomainID, req.SiteName, req.AdminUser,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record install: %w", err)
	}

	id, _ := result.LastInsertId()

	httpdocsDir := filepath.Join(s.VhostsDir, req.DomainName, "httpdocs")

	if err := os.MkdirAll(httpdocsDir, 0755); err != nil {
		s.db.Exec("UPDATE wordpress_installs SET status = 'failed' WHERE id = ?", id)
		return nil, fmt.Errorf("failed to create httpdocs directory: %w", err)
	}

	slog.Info("Cleaning httpdocs directory before WordPress install", "dir", httpdocsDir)
	if err := cleanDirectory(httpdocsDir); err != nil {
		s.db.Exec("UPDATE wordpress_installs SET status = 'failed' WHERE id = ?", id)
		return nil, fmt.Errorf("failed to clean httpdocs directory: %w", err)
	}

	if err := s.downloadWordPress(httpdocsDir); err != nil {
		s.db.Exec("UPDATE wordpress_installs SET status = 'failed' WHERE id = ?", id)
		return nil, fmt.Errorf("wordpress download failed: %w", err)
	}

	dbName := strings.ReplaceAll(req.DomainName, ".", "_")
	dbUser := "wp_" + strings.ReplaceAll(req.DomainName, ".", "_")
	dbPass := generatePassword()

	if err := s.mariadbSvc.CreateDatabase(dbName); err != nil {
		s.db.Exec("UPDATE wordpress_installs SET status = 'failed' WHERE id = ?", id)
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	if err := s.mariadbSvc.CreateUser(dbUser, "%", dbPass); err != nil {
		s.db.Exec("UPDATE wordpress_installs SET status = 'failed' WHERE id = ?", id)
		return nil, fmt.Errorf("failed to create database user: %w", err)
	}

	if err := s.mariadbSvc.GrantPrivileges(dbUser, "%", dbName, "ALL PRIVILEGES"); err != nil {
		s.db.Exec("UPDATE wordpress_installs SET status = 'failed' WHERE id = ?", id)
		return nil, fmt.Errorf("failed to grant privileges: %w", err)
	}

	if err := s.createWPConfig(httpdocsDir, dbName, dbUser, dbPass); err != nil {
		s.db.Exec("UPDATE wordpress_installs SET status = 'failed' WHERE id = ?", id)
		return nil, fmt.Errorf("failed to create wp-config.php: %w", err)
	}

	if err := s.runWPCLI(req, dbName, dbUser, dbPass); err != nil {
		s.db.Exec("UPDATE wordpress_installs SET status = 'failed' WHERE id = ?", id)
		return nil, fmt.Errorf("wordpress install failed: %w", err)
	}

	if _, err := s.db.Exec("UPDATE wordpress_installs SET status = 'completed' WHERE id = ?", id); err != nil {
		slog.Warn("Failed to update install status", "error", err)
	}

	var ownerID int64
	if err := s.db.QueryRow("SELECT owner_id FROM domains WHERE id = ?", req.DomainID).Scan(&ownerID); err != nil {
		slog.Warn("Failed to get domain owner for DB registration", "domain_id", req.DomainID, "error", err)
		ownerID = 1
	}

	dbID, err := s.registerDatabase(dbName, ownerID)
	if err != nil {
		slog.Warn("Failed to register WordPress database in panel", "database", dbName, "error", err)
	}

	if err := s.registerDatabaseUser(dbUser, "%", dbName, "ALL PRIVILEGES", dbID); err != nil {
		slog.Warn("Failed to register WordPress DB user in panel", "user", dbUser, "error", err)
	}

	slog.Info("WordPress installed successfully", "domain", req.DomainName, "database", dbName, "user", dbUser)

	return &InstallWordPressResponse{
		Install: &WordPressInstall{
			ID:        id,
			DomainID:  req.DomainID,
			SiteName:  req.SiteName,
			AdminUser: req.AdminUser,
			Status:    "completed",
			CreatedAt: time.Now(),
		},
		Message: "wordpress installed successfully",
	}, nil
}

func cleanDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

func (s *WordPressService) downloadWordPress(destDir string) error {
	zipURL := "https://wordpress.org/latest.zip"
	tmpZip := filepath.Join(os.TempDir(), "wp-latest.zip")

	resp, err := http.Get(zipURL)
	if err != nil {
		return fmt.Errorf("failed to download wordpress: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wordpress download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(tmpZip)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	out.Close()

	r, err := zip.OpenReader(tmpZip)
	if err != nil {
		return err
	}
	defer r.Close()
	os.Remove(tmpZip)

	for _, f := range r.File {
		relPath := f.Name
		if !strings.HasPrefix(relPath, "wordpress/") {
			continue
		}
		relPath = strings.TrimPrefix(relPath, "wordpress/")
		if relPath == "" {
			continue
		}

		target := filepath.Join(destDir, relPath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *WordPressService) createWPConfig(httpdocsDir, dbName, dbUser, dbPass string) error {
	configTemplate := `<?php
define( 'DB_NAME', '%s' );
define( 'DB_USER', '%s' );
define( 'DB_PASSWORD', '%s' );
define( 'DB_HOST', 'localhost' );
define( 'DB_CHARSET', 'utf8mb4' );
define( 'DB_COLLATE', '' );

define( 'AUTH_KEY',         '%s' );
define( 'SECURE_AUTH_KEY',  '%s' );
define( 'LOGGED_IN_KEY',    '%s' );
define( 'NONCE_KEY',        '%s' );
define( 'AUTH_SALT',        '%s' );
define( 'SECURE_AUTH_SALT', '%s' );
define( 'LOGGED_IN_SALT',   '%s' );
define( 'NONCE_SALT',       '%s' );

$table_prefix = 'wp_';

define( 'WP_DEBUG', false );

if ( ! defined( 'ABSPATH' ) ) {
    define( 'ABSPATH', __DIR__ . '/' );
}

require_once ABSPATH . 'wp-settings.php';
`

	config := fmt.Sprintf(configTemplate, dbName, dbUser, dbPass,
		generateSalt(), generateSalt(), generateSalt(), generateSalt(),
		generateSalt(), generateSalt(), generateSalt(), generateSalt(),
	)
	configPath := filepath.Join(httpdocsDir, "wp-config.php")
	return os.WriteFile(configPath, []byte(config), 0644)
}

func (s *WordPressService) runWPCLI(req InstallWordPressRequest, dbName, dbUser, dbPass string) error {
	wpPath := filepath.Join(s.VhostsDir, req.DomainName, "httpdocs")
	adminEmail := req.AdminEmail
	if adminEmail == "" {
		adminEmail = "admin@" + req.DomainName
	}

	cmd := exec.Command("wp", "core", "install",
		"--path="+wpPath,
		"--url=http://"+req.DomainName,
		"--title="+req.SiteName,
		"--admin_user="+req.AdminUser,
		"--admin_password="+req.AdminPass,
		"--admin_email="+adminEmail,
		"--allow-root",
	)
	cmd.Env = append(os.Environ(),
		"DB_USER="+dbUser,
		"DB_PASSWORD="+dbPass,
		"DB_NAME="+dbName,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wp core install: %s (error: %w)", string(output), err)
	}

	return nil
}
