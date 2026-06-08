package service

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

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

type DomainService struct {
	db     *database.DB
	sys    *SystemService
	nginx  *NginxService
	phpfpm *PHPFPMService
}

func NewDomainService(db *database.DB) *DomainService {
	return &DomainService{
		db:    db,
		sys:   NewSystemService(),
		nginx: NewNginxService(NginxTemplateDir, NginxConfigDir),
		phpfpm: NewPHPFPMService(NginxTemplateDir, "8.4", "/etc/php/8.4/fpm/pool.d", "/run/php"),
	}
}

// CreateDomain creates a new domain with all system resources
func (s *DomainService) CreateDomain(name string, ownerID int) (*model.Domain, error) {
	// 1. Validate domain name
	if name == "" {
		return nil, fmt.Errorf("domain name is required")
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
	username := "op_" + name

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

	// 11. Create PHP-FPM pool
	phpSocket := s.phpfpm.GetSocketPath(name)
	if err := s.phpfpm.CreatePool(name, username, documentRoot); err != nil {
		slog.Warn("Failed to create PHP-FPM pool (non-fatal)", "error", err)
	}

	// 12. Generate Nginx config
	nginxData := NginxTemplateData{
		Domain:        name,
		DocumentRoot:  documentRoot,
		LogDir:        logDir,
		PHPFPM_SOCKET: phpSocket,
	}
	if err := s.nginx.GenerateConfig(nginxData); err != nil {
		slog.Warn("Failed to generate nginx config (non-fatal)", "error", err)
	}

	// 13. Reload Nginx and PHP-FPM
	if err := s.nginx.Reload(); err != nil {
		slog.Warn("Failed to reload nginx (non-fatal)", "error", err)
	}
	if err := s.phpfpm.Reload(); err != nil {
		slog.Warn("Failed to reload PHP-FPM (non-fatal)", "error", err)
	}

	// 13. Insert into database
	result, err := s.db.Exec(
		"INSERT INTO domains (name, ip_address, status, owner_id, document_root) VALUES (?, ?, 'active', ?, ?)",
		name, ipAddress, ownerID, documentRoot,
	)
	if err != nil {
		// Cleanup system resources
		s.sys.DeleteUser(username)
		os.RemoveAll(domainDir)
		return nil, fmt.Errorf("failed to insert domain into database: %w", err)
	}

	id, _ := result.LastInsertId()

	// 14. Return created domain
	var domain model.Domain
	err = s.db.QueryRow(
		"SELECT id, name, ip_address, status, owner_id, document_root, created_at, updated_at FROM domains WHERE id = ?", id,
	).Scan(&domain.ID, &domain.Name, &domain.IPAddress, &domain.Status, &domain.OwnerID, &domain.DocumentRoot, &domain.CreatedAt, &domain.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created domain: %w", err)
	}

	slog.Info("Domain created successfully", "domain", name, "id", id)
	return &domain, nil
}

// DeleteDomain removes a domain and all its system resources
func (s *DomainService) DeleteDomain(id int) (*model.Domain, error) {
	// 1. Fetch domain
	var domain model.Domain
	err := s.db.QueryRow(
		"SELECT id, name, ip_address, status, owner_id, document_root, created_at, updated_at FROM domains WHERE id = ?", id,
	).Scan(&domain.ID, &domain.Name, &domain.IPAddress, &domain.Status, &domain.OwnerID, &domain.DocumentRoot, &domain.CreatedAt, &domain.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("domain not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch domain: %w", err)
	}

	username := "op_" + domain.Name

	// 2. Remove PHP-FPM pool
	if err := s.phpfpm.RemovePool(domain.Name); err != nil {
		slog.Warn("Failed to remove PHP-FPM pool", "error", err)
	}

	// 3. Remove Nginx config
	if err := s.nginx.RemoveConfig(domain.Name); err != nil {
		slog.Warn("Failed to remove nginx config", "error", err)
	}

	// 4. Reload Nginx
	if err := s.nginx.Reload(); err != nil {
		slog.Warn("Failed to reload nginx", "error", err)
	}

	// 5. Reload PHP-FPM
	if err := s.phpfpm.Reload(); err != nil {
		slog.Warn("Failed to reload PHP-FPM", "error", err)
	}

	// 6. Delete Linux user (this also removes home dir with -r)
	if err := s.sys.DeleteUser(username); err != nil {
		slog.Warn("Failed to delete Linux user", "error", err)
	}

	// 7. Remove directory structure (fallback if userdel didn't remove it)
	domainDir := filepath.Join(VhostsBaseDir, domain.Name)
	if err := os.RemoveAll(domainDir); err != nil {
		slog.Warn("Failed to remove domain directory", "error", err)
	}

	// 8. Delete from database
	if _, err := s.db.Exec("DELETE FROM domains WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("failed to delete domain from database: %w", err)
	}

	slog.Info("Domain deleted", "domain", domain.Name, "id", id)
	return &domain, nil
}

// GetDomain fetches a domain by ID
func (s *DomainService) GetDomain(id int) (*model.Domain, error) {
	var domain model.Domain
	err := s.db.QueryRow(
		"SELECT id, name, ip_address, status, owner_id, document_root, created_at, updated_at FROM domains WHERE id = ?", id,
	).Scan(&domain.ID, &domain.Name, &domain.IPAddress, &domain.Status, &domain.OwnerID, &domain.DocumentRoot, &domain.CreatedAt, &domain.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("domain not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch domain: %w", err)
	}
	return &domain, nil
}

// ListDomains returns all domains for a user (or all if admin)
func (s *DomainService) ListDomains(ownerID int, isAdmin bool) ([]model.Domain, error) {
	var rows *sql.Rows
	var err error

	if isAdmin {
		rows, err = s.db.Query("SELECT id, name, ip_address, status, owner_id, document_root, created_at, updated_at FROM domains ORDER BY name")
	} else {
		rows, err = s.db.Query("SELECT id, name, ip_address, status, owner_id, document_root, created_at, updated_at FROM domains WHERE owner_id = ? ORDER BY name", ownerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query domains: %w", err)
	}
	defer rows.Close()

	var domains []model.Domain
	for rows.Next() {
		var d model.Domain
		if err := rows.Scan(&d.ID, &d.Name, &d.IPAddress, &d.Status, &d.OwnerID, &d.DocumentRoot, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		domains = append(domains, d)
	}

	if domains == nil {
		domains = []model.Domain{}
	}

	return domains, nil
}

// UpdateDomainStatus updates a domain's status
func (s *DomainService) UpdateDomainStatus(id int, status string) (*model.Domain, error) {
	result, err := s.db.Exec("UPDATE domains SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", status, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update domain: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, fmt.Errorf("domain not found")
	}

	return s.GetDomain(id)
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
