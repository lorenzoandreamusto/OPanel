package service

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"opanel/internal/database"
	"opanel/internal/model"
)

const (
	VhostsBaseDir  = "/var/www/vhosts"
	OPanelGroup    = "opanel_users"
	SshdConfigPath = "/etc/ssh/sshd_config"
	NginxTemplateDir = "/opt/opanel/templates"
	NginxConfigDir   = "/etc/nginx/sites-enabled"
)

const domainSelectColumns = "id, name, ip_address, status, owner_id, document_root, php_version, hosting_type, ssl_enabled, auto_db, created_at, updated_at"

type DomainService struct {
	db       *database.DB
	sys      *SystemService
	nginx    *NginxService
	phpfpm   *PHPFPMService
	mariadb  *MariaDBService
}

func NewDomainService(db *database.DB, templatesDir, nginxConfDir, phpVersion, phpFPMPoolDir, phpFPMSocketDir string, mariadb *MariaDBService) *DomainService {
	return &DomainService{
		db:      db,
		sys:     NewSystemService(),
		nginx:   NewNginxService(templatesDir, nginxConfDir),
		phpfpm:  NewPHPFPMService(templatesDir, phpVersion, phpFPMPoolDir, phpFPMSocketDir),
		mariadb: mariadb,
	}
}

func scanDomain(row *sql.Row) (*model.Domain, error) {
	var d model.Domain
	err := row.Scan(
		&d.ID, &d.Name, &d.IPAddress, &d.Status, &d.OwnerID, &d.DocumentRoot,
		&d.PHPVersion, &d.HostingType, &d.SSLEnabled, &d.AutoDB,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func scanDomainRows(rows *sql.Rows) ([]model.Domain, error) {
	var domains []model.Domain
	for rows.Next() {
		var d model.Domain
		if err := rows.Scan(
			&d.ID, &d.Name, &d.IPAddress, &d.Status, &d.OwnerID, &d.DocumentRoot,
			&d.PHPVersion, &d.HostingType, &d.SSLEnabled, &d.AutoDB,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		domains = append(domains, d)
	}
	if domains == nil {
		domains = []model.Domain{}
	}
	return domains, nil
}

// CreateDomain creates a new domain with all system resources
func (s *DomainService) CreateDomain(req *model.CreateDomainRequest, ownerID int) (*model.Domain, error) {
	name := req.Name

	// 1. Validate domain name
	if name == "" {
		return nil, fmt.Errorf("domain name is required")
	}

	// Apply defaults
	phpVersion := req.PHPVersion
	if phpVersion == "" {
		phpVersion = "8.4"
	}
	hostingType := req.HostingType
	if hostingType == "" {
		hostingType = "php"
	}

	// 2. Check if domain already exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM domains WHERE name = ?", name).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check domain: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("domain %s already exists", name)
	}

	// 3. Get server IP
	ipAddress, err := s.sys.GetIPInterface("")
	if err != nil {
		ipAddress = "127.0.0.1"
	}

	// 4. Define paths
	domainDir := filepath.Join(VhostsBaseDir, name)
	documentRoot := filepath.Join(domainDir, "httpdocs")
	logDir := filepath.Join(domainDir, "logs")
	tmpDir := filepath.Join(domainDir, "tmp")
	username := domainToUsername(name)

	// 5. Create directory structure
	for _, dir := range []string{domainDir, documentRoot, logDir, tmpDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	slog.Info("Created domain directories", "domain", name, "path", domainDir)

	// 5b. Copy default index.html to httpdocs
	defaultIndexSrc := filepath.Join(NginxTemplateDir, "nginx", "index.html")
	defaultIndexDst := filepath.Join(documentRoot, "index.html")
	if err := copyFile(defaultIndexSrc, defaultIndexDst); err != nil {
		slog.Warn("Failed to copy default index.html (non-fatal)", "error", err)
	} else {
		slog.Info("Copied default index.html", "domain", name)
	}

	// 6. Ensure opanel_users group exists
	if err := s.sys.EnsureGroup(OPanelGroup); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	// 7. Create Linux user
	if err := s.sys.CreateUser(username, domainDir); err != nil {
		// Cleanup directories on failure
		os.RemoveAll(domainDir)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 8. Add user to opanel_users group
	if err := s.sys.AddUserToGroup(username, OPanelGroup); err != nil {
		// Cleanup
		s.sys.DeleteUser(username)
		os.RemoveAll(domainDir)
		return nil, fmt.Errorf("failed to add user to group: %w", err)
	}

	// 9. Set ownership
	if err := s.sys.SetOwnership(domainDir, username); err != nil {
		// Cleanup
		s.sys.DeleteUser(username)
		os.RemoveAll(domainDir)
		return nil, fmt.Errorf("failed to set ownership: %w", err)
	}

	// 10. Setup SFTP chroot
	if err := s.sys.SetupSFTPChroot(SshdConfigPath, OPanelGroup); err != nil {
		slog.Warn("Failed to setup SFTP chroot (non-fatal)", "error", err)
	}

	// 11. Create PHP-FPM pool (skip for static hosting)
	var phpSocket string
	if hostingType != "static" {
		phpSocket = s.phpfpm.GetSocketPath(name, phpVersion)
		if err := s.phpfpm.CreatePool(name, username, documentRoot, phpVersion); err != nil {
			slog.Warn("Failed to create PHP-FPM pool (non-fatal)", "error", err)
		}
	}

	// 12. Generate Nginx config
	nginxData := NginxTemplateData{
		Domain:        name,
		DocumentRoot:  documentRoot,
		LogDir:        logDir,
		PHPFPM_SOCKET: phpSocket,
	}
	if err := s.nginx.GenerateConfig(nginxData, "active", hostingType); err != nil {
		slog.Warn("Failed to generate nginx config (non-fatal)", "error", err)
	}

	// 13. Reload Nginx and PHP-FPM
	if err := s.nginx.Reload(); err != nil {
		slog.Warn("Failed to reload nginx (non-fatal)", "error", err)
	}
	if hostingType != "static" {
		if err := s.phpfpm.Reload(); err != nil {
			slog.Warn("Failed to reload PHP-FPM (non-fatal)", "error", err)
		}
	}

	// 14. Insert into database
	sslInt := 0
	if req.SSLEnabled {
		sslInt = 1
	}
	autoDBInt := 0
	if req.AutoDB {
		autoDBInt = 1
	}
	result, err := s.db.Exec(
		"INSERT INTO domains (name, ip_address, status, owner_id, document_root, php_version, hosting_type, ssl_enabled, auto_db) VALUES (?, ?, 'active', ?, ?, ?, ?, ?, ?)",
		name, ipAddress, ownerID, documentRoot, phpVersion, hostingType, sslInt, autoDBInt,
	)
	if err != nil {
		// Cleanup system resources
		s.sys.DeleteUser(username)
		os.RemoveAll(domainDir)
		return nil, fmt.Errorf("failed to insert domain into database: %w", err)
	}

	id, _ := result.LastInsertId()

	// 16. Auto-create database if requested
	if req.AutoDB && s.mariadb != nil {
		dbName := domainToDBName(name)
		if err := s.mariadb.CreateDatabase(dbName); err != nil {
			slog.Warn("Failed to auto-create MariaDB database", "error", err, "db", dbName)
		} else {
			// Track in SQLite
			if _, err := s.db.Exec("INSERT INTO databases (name, owner_id) VALUES (?, ?)", dbName, ownerID); err != nil {
				slog.Warn("Failed to track auto-created database", "error", err, "db", dbName)
			} else {
				slog.Info("Auto-created database", "domain", name, "db", dbName)
			}
		}
	}

	// 17. Return created domain
	domain, err := scanDomain(s.db.QueryRow(
		"SELECT "+domainSelectColumns+" FROM domains WHERE id = ?", id,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created domain: %w", err)
	}

	slog.Info("Domain created successfully", "domain", name, "id", id)
	return domain, nil
}

// DeleteDomain removes a domain and all its system resources
func (s *DomainService) DeleteDomain(id int) (*model.Domain, error) {
	// 1. Fetch domain
	domain, err := scanDomain(s.db.QueryRow(
		"SELECT "+domainSelectColumns+" FROM domains WHERE id = ?", id,
	))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("domain not found")
		}
		return nil, fmt.Errorf("failed to fetch domain: %w", err)
	}

	username := domainToUsername(domain.Name)

	// 2. Remove PHP-FPM pool
	if err := s.phpfpm.RemovePool(domain.Name); err != nil {
		slog.Warn("Failed to remove PHP-FPM pool", "error", err)
	}

	// 3. Remove auto-created database if auto_db was set
	if domain.AutoDB && s.mariadb != nil {
		dbName := domainToDBName(domain.Name)
		// Delete from SQLite first
		if _, err := s.db.Exec("DELETE FROM databases WHERE name = ?", dbName); err != nil {
			slog.Warn("Failed to remove auto-db tracking", "error", err)
		}
		// Drop from MariaDB
		if err := s.mariadb.DropDatabase(dbName); err != nil {
			slog.Warn("Failed to drop auto-created MariaDB database", "error", err, "db", dbName)
		} else {
			slog.Info("Auto-deleted database", "domain", domain.Name, "db", dbName)
		}
	}

	// 4. Remove Nginx config
	if err := s.nginx.RemoveConfig(domain.Name); err != nil {
		slog.Warn("Failed to remove nginx config", "error", err)
	}

	// 5. Reload Nginx
	if err := s.nginx.Reload(); err != nil {
		slog.Warn("Failed to reload nginx", "error", err)
	}

	// 6. Reload PHP-FPM
	if err := s.phpfpm.Reload(); err != nil {
		slog.Warn("Failed to reload PHP-FPM", "error", err)
	}

	// 7. Delete Linux user (this also removes home dir with -r)
	if err := s.sys.DeleteUser(username); err != nil {
		slog.Warn("Failed to delete Linux user", "error", err)
	}

	// 8. Remove directory structure (fallback if userdel didn't remove it)
	domainDir := filepath.Join(VhostsBaseDir, domain.Name)
	if err := os.RemoveAll(domainDir); err != nil {
		slog.Warn("Failed to remove domain directory", "error", err)
	}

	// 9. Delete from database
	if _, err := s.db.Exec("DELETE FROM domains WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("failed to delete domain from database: %w", err)
	}

	slog.Info("Domain deleted", "domain", domain.Name, "id", id)
	return domain, nil
}

// GetDomain fetches a domain by ID
func (s *DomainService) GetDomain(id int) (*model.Domain, error) {
	domain, err := scanDomain(s.db.QueryRow(
		"SELECT "+domainSelectColumns+" FROM domains WHERE id = ?", id,
	))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("domain not found")
		}
		return nil, fmt.Errorf("failed to fetch domain: %w", err)
	}
	return domain, nil
}

// ListDomains returns all domains for a user (or all if admin)
func (s *DomainService) ListDomains(ownerID int, isAdmin bool) ([]model.Domain, error) {
	var rows *sql.Rows
	var err error

	if isAdmin {
		rows, err = s.db.Query("SELECT " + domainSelectColumns + " FROM domains ORDER BY name")
	} else {
		rows, err = s.db.Query("SELECT "+domainSelectColumns+" FROM domains WHERE owner_id = ? ORDER BY name", ownerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query domains: %w", err)
	}
	defer rows.Close()

	return scanDomainRows(rows)
}

// UpdateDomain updates a domain's fields and applies system changes
func (s *DomainService) UpdateDomain(id int, req *model.UpdateDomainRequest) (*model.Domain, error) {
	// 1. Fetch current domain state
	domain, err := scanDomain(s.db.QueryRow(
		"SELECT "+domainSelectColumns+" FROM domains WHERE id = ?", id,
	))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("domain not found")
		}
		return nil, fmt.Errorf("failed to fetch domain: %w", err)
	}

	setClauses := []string{}
	args := []interface{}{}

	newStatus := domain.Status
	newHostingType := domain.HostingType
	newPHPVersion := domain.PHPVersion

	if req.Status != "" {
		setClauses = append(setClauses, "status = ?")
		args = append(args, req.Status)
		newStatus = req.Status
	}
	if req.PHPVersion != "" {
		setClauses = append(setClauses, "php_version = ?")
		args = append(args, req.PHPVersion)
		newPHPVersion = req.PHPVersion
	}
	if req.HostingType != "" {
		setClauses = append(setClauses, "hosting_type = ?")
		args = append(args, req.HostingType)
		newHostingType = req.HostingType
	}
	if req.SSLEnabled != nil {
		sslInt := 0
		if *req.SSLEnabled {
			sslInt = 1
		}
		setClauses = append(setClauses, "ssl_enabled = ?")
		args = append(args, sslInt)
	}
	if req.AutoDB != nil {
		autoDBInt := 0
		if *req.AutoDB {
			autoDBInt = 1
		}
		setClauses = append(setClauses, "auto_db = ?")
		args = append(args, autoDBInt)
	}

	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE domains SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err = s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update domain: %w", err)
	}

	// 2. Apply system changes if status, hosting_type, or php_version changed
	statusChanged := newStatus != domain.Status
	hostingTypeChanged := newHostingType != domain.HostingType
	phpVersionChanged := newPHPVersion != domain.PHPVersion

	if statusChanged || hostingTypeChanged || phpVersionChanged {
		username := domainToUsername(domain.Name)
		documentRoot := domain.DocumentRoot
		logDir := filepath.Join(filepath.Dir(documentRoot), "logs")

		// Build Nginx template data
		phpSocket := ""
		if newHostingType != "static" {
			phpSocket = s.phpfpm.GetSocketPath(domain.Name, newPHPVersion)
		}
		nginxData := NginxTemplateData{
			Domain:        domain.Name,
			DocumentRoot:  documentRoot,
			LogDir:        logDir,
			PHPFPM_SOCKET: phpSocket,
		}

		// Regenerate Nginx config (handles status and hosting_type)
		if err := s.nginx.GenerateConfig(nginxData, newStatus, newHostingType); err != nil {
			slog.Warn("Failed to regenerate nginx config", "error", err)
		}
		if err := s.nginx.Reload(); err != nil {
			slog.Warn("Failed to reload nginx", "error", err)
		}

		// Handle PHP-FPM pool changes
		if hostingTypeChanged || phpVersionChanged {
			oldPoolIsStatic := domain.HostingType == "static"
			newPoolIsStatic := newHostingType == "static"

			if oldPoolIsStatic && !newPoolIsStatic {
				// Static → PHP: create pool
				if err := s.phpfpm.CreatePool(domain.Name, username, documentRoot, newPHPVersion); err != nil {
					slog.Warn("Failed to create PHP-FPM pool", "error", err)
				}
				if err := s.phpfpm.Reload(); err != nil {
					slog.Warn("Failed to reload PHP-FPM", "error", err)
				}
			} else if !oldPoolIsStatic && newPoolIsStatic {
				// PHP → Static: remove pool
				if err := s.phpfpm.RemovePool(domain.Name); err != nil {
					slog.Warn("Failed to remove PHP-FPM pool", "error", err)
				}
				if err := s.phpfpm.Reload(); err != nil {
					slog.Warn("Failed to reload PHP-FPM", "error", err)
				}
			} else if !oldPoolIsStatic && !newPoolIsStatic {
				// PHP → PHP (different version): recreate pool
				if phpVersionChanged {
					if err := s.phpfpm.RemovePool(domain.Name); err != nil {
						slog.Warn("Failed to remove old PHP-FPM pool", "error", err)
					}
					if err := s.phpfpm.CreatePool(domain.Name, username, documentRoot, newPHPVersion); err != nil {
						slog.Warn("Failed to create new PHP-FPM pool", "error", err)
					}
					if err := s.phpfpm.Reload(); err != nil {
						slog.Warn("Failed to reload PHP-FPM", "error", err)
					}
				}
			}
		}
	}

	return s.GetDomain(id)
}

func domainToUsername(name string) string {
	return "op_" + strings.ReplaceAll(name, ".", "-")
}

// domainToDBName converts a domain name to a valid MariaDB database name (dots → underscores)
func domainToDBName(name string) string {
	return strings.ReplaceAll(name, ".", "_")
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return destFile.Sync()
}
