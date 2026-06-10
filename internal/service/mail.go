package service

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"opanel/internal/database"
	"opanel/internal/model"

	_ "github.com/go-sql-driver/mysql"
)

const (
	DovecotConfDir    = "/etc/dovecot"
	PostfixConfDir    = "/etc/postfix"
	RspamdConfDir     = "/etc/rspamd"
	MailboxesBaseDir  = "/var/vmail"
	DKIMKeyDir        = "/var/lib/rspamd/dkim"
	DKIMSelector      = "default"
)

type MailService struct {
	db *database.DB
}

func NewMailService(db *database.DB) *MailService {
	return &MailService{db: db}
}

// connectMailDB connects to the MariaDB opanel_mail database
func (s *MailService) connectMailDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", "root@unix(/var/run/mysqld/mysqld.sock)/opanel_mail")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mail MariaDB: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping mail MariaDB: %w", err)
	}
	return db, nil
}

// hashPassword hashes a password using doveadm pw
func hashPassword(password string) string {
	cmd := exec.Command("doveadm", "pw", "-s", "SHA256-CRYPT", "-p", password)
	out, err := cmd.Output()
	if err != nil {
		// Fallback: use plaintext prefixed with {PLAIN}
		return "{PLAIN}" + password
	}
	return strings.TrimSpace(string(out))
}

// --- Mail Domains ---

// CreateMailDomain enables mail for a domain
func (s *MailService) CreateMailDomain(req *model.CreateMailDomainRequest) (*model.MailDomain, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM mail_domains WHERE domain_id = ?", req.DomainID).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check mail domain: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("mail domain already exists for this domain")
	}

	var domainName string
	err = s.db.QueryRow("SELECT name FROM domains WHERE id = ?", req.DomainID).Scan(&domainName)
	if err != nil {
		return nil, fmt.Errorf("domain not found")
	}

	result, err := s.db.Exec(
		"INSERT INTO mail_domains (domain_id, name, enabled, dkim_enabled) VALUES (?, ?, 1, 1)",
		req.DomainID, domainName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mail domain: %w", err)
	}

	id, _ := result.LastInsertId()

	// Also insert into MariaDB for Postfix virtual domains
	if mdb, err := s.connectMailDB(); err == nil {
		defer mdb.Close()
		mdb.Exec("INSERT IGNORE INTO virtual_domains (name) VALUES (?)", domainName)
	} else {
		slog.Warn("Failed to connect to mail MariaDB (non-fatal)", "error", err)
	}

	// Create mailbox directory
	mailDir := filepath.Join(MailboxesBaseDir, domainName)
	os.MkdirAll(mailDir, 0755)

	// Generate DKIM key pair
	if err := s.generateDKIMKey(domainName); err != nil {
		slog.Warn("Failed to generate DKIM key (non-fatal)", "error", err, "domain", domainName)
	}

	// Update Dovecot userdb and reload
	s.updateDovecotUserdb()
	s.ReloadDovecot()
	s.ReloadPostfix()

	slog.Info("Mail domain created", "domain", domainName, "id", id)
	return s.GetMailDomain(int(id))
}

// GetMailDomain returns a mail domain by ID
func (s *MailService) GetMailDomain(id int) (*model.MailDomain, error) {
	var m model.MailDomain
	err := s.db.QueryRow(
		"SELECT id, domain_id, name, enabled, dkim_enabled, dkim_selector, created_at, updated_at FROM mail_domains WHERE id = ?", id,
	).Scan(&m.ID, &m.DomainID, &m.Name, &m.Enabled, &m.DKIMEnabled, &m.DKIMSelector, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("mail domain not found")
		}
		return nil, fmt.Errorf("failed to fetch mail domain: %w", err)
	}
	return &m, nil
}

// GetMailDomainByName returns a mail domain by name
func (s *MailService) GetMailDomainByName(name string) (*model.MailDomain, error) {
	var m model.MailDomain
	err := s.db.QueryRow(
		"SELECT id, domain_id, name, enabled, dkim_enabled, dkim_selector, created_at, updated_at FROM mail_domains WHERE name = ?", name,
	).Scan(&m.ID, &m.DomainID, &m.Name, &m.Enabled, &m.DKIMEnabled, &m.DKIMSelector, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("mail domain not found")
		}
		return nil, fmt.Errorf("failed to fetch mail domain: %w", err)
	}
	return &m, nil
}

// ListMailDomains returns all mail domains
func (s *MailService) ListMailDomains() ([]model.MailDomain, error) {
	rows, err := s.db.Query(
		"SELECT id, domain_id, name, enabled, dkim_enabled, dkim_selector, created_at, updated_at FROM mail_domains ORDER BY name",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query mail domains: %w", err)
	}
	defer rows.Close()

	var domains []model.MailDomain
	for rows.Next() {
		var m model.MailDomain
		if err := rows.Scan(&m.ID, &m.DomainID, &m.Name, &m.Enabled, &m.DKIMEnabled, &m.DKIMSelector, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan mail domain: %w", err)
		}
		domains = append(domains, m)
	}
	if domains == nil {
		domains = []model.MailDomain{}
	}
	return domains, nil
}

// DeleteMailDomain removes a mail domain
func (s *MailService) DeleteMailDomain(id int) error {
	mailDomain, err := s.GetMailDomain(id)
	if err != nil {
		return err
	}

	// Delete mail accounts first
	s.db.Exec("DELETE FROM mail_accounts WHERE domain_id = ?", id)

	// Delete from SQLite
	s.db.Exec("DELETE FROM mail_domains WHERE id = ?", id)

	// Also clean MariaDB
	if mdb, err := s.connectMailDB(); err == nil {
		defer mdb.Close()
		mdb.Exec("DELETE FROM virtual_users WHERE domain_id = (SELECT id FROM virtual_domains WHERE name = ?)", mailDomain.Name)
		mdb.Exec("DELETE FROM virtual_aliases WHERE domain_id = (SELECT id FROM virtual_domains WHERE name = ?)", mailDomain.Name)
		mdb.Exec("DELETE FROM virtual_domains WHERE name = ?", mailDomain.Name)
	}

	// Remove DKIM key
	dkimPrivKey := filepath.Join(DKIMKeyDir, fmt.Sprintf("%s.%s.key", DKIMSelector, mailDomain.Name))
	os.Remove(dkimPrivKey)
	dkimPubKey := filepath.Join(DKIMKeyDir, fmt.Sprintf("%s.%s.pub", DKIMSelector, mailDomain.Name))
	os.Remove(dkimPubKey)

	// Update Dovecot userdb and reload
	s.updateDovecotUserdb()
	s.ReloadDovecot()
	s.ReloadPostfix()

	slog.Info("Mail domain deleted", "domain", mailDomain.Name, "id", id)
	return nil
}

// --- Mail Accounts ---

// CreateMailAccount creates an email account
func (s *MailService) CreateMailAccount(domainID int, req *model.CreateMailAccountRequest) (*model.MailAccount, error) {
	// Check domain exists
	var domainName string
	err := s.db.QueryRow("SELECT name FROM mail_domains WHERE id = ?", domainID).Scan(&domainName)
	if err != nil {
		return nil, fmt.Errorf("mail domain not found")
	}

	quota := req.Quota
	if quota == 0 {
		quota = 1024 * 1024 * 1024 // 1GB default
	}

	// Check for duplicate
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM mail_accounts WHERE domain_id = ? AND username = ?", domainID, req.Username).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check account: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("email account already exists")
	}

	// Hash password for Dovecot
	hashedPw := hashPassword(req.Password)

	result, err := s.db.Exec(
		"INSERT INTO mail_accounts (domain_id, username, password, quota, enabled) VALUES (?, ?, ?, ?, 1)",
		domainID, req.Username, hashedPw, quota,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mail account: %w", err)
	}

	id, _ := result.LastInsertId()

	// Also insert into MariaDB for Postfix virtual users
	if mdb, err := s.connectMailDB(); err == nil {
		defer mdb.Close()
		// Get the domain_id in MariaDB
		var mdbDomainID int
		mdb.QueryRow("SELECT id FROM virtual_domains WHERE name = ?", domainName).Scan(&mdbDomainID)
		if mdbDomainID > 0 {
			mdb.Exec("INSERT INTO virtual_users (domain_id, username, password, quota, enabled) VALUES (?, ?, ?, ?, 1)",
				mdbDomainID, req.Username, hashedPw, quota)
		}
	} else {
		slog.Warn("Failed to connect to mail MariaDB (non-fatal)", "error", err)
	}

	// Create mailbox directory
	fullEmail := req.Username + "@" + domainName
	mailDir := filepath.Join(MailboxesBaseDir, domainName, req.Username, "Maildir")
	os.MkdirAll(filepath.Join(mailDir, "cur"), 0755)
	os.MkdirAll(filepath.Join(mailDir, "new"), 0755)
	os.MkdirAll(filepath.Join(mailDir, "tmp"), 0755)

	// Update Dovecot userdb
	s.updateDovecotUserdb()

	// Reload Dovecot
	s.ReloadDovecot()

	slog.Info("Mail account created", "email", fullEmail, "id", id)
	return s.GetMailAccount(int(id))
}

// GetMailAccount returns a mail account by ID
func (s *MailService) GetMailAccount(id int) (*model.MailAccount, error) {
	var m model.MailAccount
	err := s.db.QueryRow(
		"SELECT id, domain_id, username, password, quota, used, enabled, created_at, updated_at FROM mail_accounts WHERE id = ?", id,
	).Scan(&m.ID, &m.DomainID, &m.Username, &m.Password, &m.Quota, &m.Used, &m.Enabled, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("mail account not found")
		}
		return nil, fmt.Errorf("failed to fetch mail account: %w", err)
	}
	return &m, nil
}

// ListMailAccounts returns all mail accounts for a mail domain
func (s *MailService) ListMailAccounts(domainID int) ([]model.MailAccount, error) {
	rows, err := s.db.Query(
		"SELECT id, domain_id, username, '', quota, used, enabled, created_at, updated_at FROM mail_accounts WHERE domain_id = ? ORDER BY username", domainID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query mail accounts: %w", err)
	}
	defer rows.Close()

	var accounts []model.MailAccount
	for rows.Next() {
		var m model.MailAccount
		if err := rows.Scan(&m.ID, &m.DomainID, &m.Username, &m.Password, &m.Quota, &m.Used, &m.Enabled, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan mail account: %w", err)
		}
		accounts = append(accounts, m)
	}
	if accounts == nil {
		accounts = []model.MailAccount{}
	}
	return accounts, nil
}

// UpdateMailAccount updates a mail account
func (s *MailService) UpdateMailAccount(id int, req *model.UpdateMailAccountRequest) (*model.MailAccount, error) {
	setClauses := []string{}
	args := []interface{}{}
	var hashedPw string

	if req.Password != "" {
		hashedPw = hashPassword(req.Password)
		setClauses = append(setClauses, "password = ?")
		args = append(args, hashedPw)
	}
	if req.Quota != nil {
		setClauses = append(setClauses, "quota = ?")
		args = append(args, *req.Quota)
	}
	if req.Enabled != nil {
		setClauses = append(setClauses, "enabled = ?")
		args = append(args, *req.Enabled)
	}

	if len(setClauses) == 0 {
		return s.GetMailAccount(id)
	}

	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE mail_accounts SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update mail account: %w", err)
	}

	// Sync to MariaDB
	if hashedPw != "" {
		account, _ := s.GetMailAccount(id)
		if account != nil {
			var domainName string
			s.db.QueryRow("SELECT name FROM mail_domains WHERE id = ?", account.DomainID).Scan(&domainName)
			if mdb, err := s.connectMailDB(); err == nil {
				defer mdb.Close()
				mdb.Exec("UPDATE virtual_users SET password = ? WHERE username = ? AND domain_id = (SELECT id FROM virtual_domains WHERE name = ?)",
					hashedPw, account.Username, domainName)
			}
		}
	}

	// Update Dovecot userdb
	s.updateDovecotUserdb()
	s.ReloadDovecot()

	return s.GetMailAccount(id)
}

// DeleteMailAccount removes a mail account
func (s *MailService) DeleteMailAccount(id int) error {
	account, err := s.GetMailAccount(id)
	if err != nil {
		return err
	}

	// Get domain name
	var domainName string
	s.db.QueryRow("SELECT name FROM mail_domains WHERE id = ?", account.DomainID).Scan(&domainName)

	// Delete from SQLite
	s.db.Exec("DELETE FROM mail_accounts WHERE id = ?", id)

	// Also clean MariaDB
	if mdb, err := s.connectMailDB(); err == nil {
		defer mdb.Close()
		mdb.Exec("DELETE FROM virtual_users WHERE username = ? AND domain_id = (SELECT id FROM virtual_domains WHERE name = ?)",
			account.Username, domainName)
	}

	// Remove mailbox directory
	if domainName != "" {
		mailDir := filepath.Join(MailboxesBaseDir, domainName, account.Username)
		os.RemoveAll(mailDir)
	}

	// Update Dovecot userdb
	s.updateDovecotUserdb()
	s.ReloadDovecot()

	slog.Info("Mail account deleted", "id", id)
	return nil
}

// --- Autoconfiguration ---

// GetAutoconfig returns email client autoconfiguration data
func (s *MailService) GetAutoconfig(domainName, email string) (*model.MailAutoconfigResponse, error) {
	var enabled int
	err := s.db.QueryRow("SELECT enabled FROM mail_domains WHERE name = ?", domainName).Scan(&enabled)
	if err != nil || enabled == 0 {
		return nil, fmt.Errorf("mail not enabled for domain")
	}

	return &model.MailAutoconfigResponse{
		Domain:   domainName,
		IMAPHost: "mail." + domainName,
		IMAPPort: 993,
		IMAPSSL:  993,
		IMAPTLS:  143,
		SMTPHost: "mail." + domainName,
		SMTPPort: 465,
		SMTPSSL:  465,
		SMTPTLS:  587,
		SMTPAuth: true,
		Username: email,
	}, nil
}

// --- DKIM ---

// generateDKIMKey generates a DKIM key pair using rspamd
func (s *MailService) generateDKIMKey(domainName string) error {
	if err := os.MkdirAll(DKIMKeyDir, 0755); err != nil {
		return fmt.Errorf("failed to create DKIM key directory: %w", err)
	}

	privKeyPath := filepath.Join(DKIMKeyDir, fmt.Sprintf("%s.%s.key", DKIMSelector, domainName))
	pubKeyPath := filepath.Join(DKIMKeyDir, fmt.Sprintf("%s.%s.pub", DKIMSelector, domainName))

	// Generate RSA key pair using openssl
	cmd := exec.Command("openssl", "genrsa", "-out", privKeyPath, "2048")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate DKIM private key: %s: %w", string(output), err)
	}

	// Extract public key
	cmd = exec.Command("openssl", "rsa", "-in", privKeyPath, "-pubout", "-out", pubKeyPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to extract DKIM public key: %s: %w", string(output), err)
	}

	// Set permissions
	os.Chmod(privKeyPath, 0600)

	slog.Info("Generated DKIM key pair", "domain", domainName, "selector", DKIMSelector)
	return nil
}

// GetDKIMPublicKey returns the DKIM public key for a domain (for DNS TXT record)
func (s *MailService) GetDKIMPublicKey(domainName string) (string, error) {
	pubKeyPath := filepath.Join(DKIMKeyDir, fmt.Sprintf("%s.%s.pub", DKIMSelector, domainName))
	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", fmt.Errorf("DKIM public key not found: %w", err)
	}

	// Extract just the key material (remove PEM headers and newlines)
	key := string(data)
	key = strings.ReplaceAll(key, "-----BEGIN PUBLIC KEY-----", "")
	key = strings.ReplaceAll(key, "-----END PUBLIC KEY-----", "")
	key = strings.ReplaceAll(key, "\n", "")
	key = strings.TrimSpace(key)

	dnsRecord := fmt.Sprintf("v=DKIM1; k=rsa; p=%s", key)
	return dnsRecord, nil
}

// --- Config Management ---

// updateDovecotConfig updates Dovecot configuration for virtual users
func (s *MailService) updateDovecotConfig() {
	config := `# Auto-generated by OPanel
# Do not edit manually

protocols = imap pop3

listen = *,::

# Auth configuration
disable_plaintext_auth = yes
auth_mechanisms = plain login

# Passdb: check passwords
passdb {
  driver = passwd-file
  args = /etc/dovecot/users
}

# Userdb: user settings
userdb {
  driver = passwd-file
  args = /etc/dovecot/users
}

# Mail location
mail_location = maildir:/var/vmail/%d/%n/Maildir

# Run as vmail user
mail_uid = vmail
mail_gid = vmail
first_valid_uid = 8
last_valid_uid = 8

# SSL
ssl = required
ssl_cert = </etc/ssl/certs/ssl-cert-snakeoil.pem
ssl_key = </etc/ssl/private/ssl-cert-snakeoil.key

# Logging
log_path = /var/log/dovecot.log

# Sieve
protocol imap {
  mail_max_userip_connections = 20
}

protocol pop3 {
  pop3_uidl_format = %08Xu%08Xv
}

service auth {
  unix_listener /var/spool/postfix/private/dovecot-auth {
    mode = 0660
    user = postfix
    group = postfix
  }
}

service dict {
  unix_listener /run/dovecot/dict {
  }
}
`
	os.MkdirAll(DovecotConfDir, 0755)
	os.WriteFile(filepath.Join(DovecotConfDir, "10-mail.conf"), []byte(config), 0644)
}

// updatePostfixConfig updates Postfix configuration for virtual domains
func (s *MailService) updatePostfixConfig() {
	config := `# Auto-generated by OPanel
# Do not edit manually

# Basic settings
myhostname = mail.example.com
mydomain = example.com
myorigin = $mydomain
mydestination = $myhostname, localhost.$mydomain, localhost
mynetworks = 127.0.0.0/8
inet_interfaces = all
inet_protocols = all

# Virtual domains
virtual_mailbox_domains = mysql:/etc/postfix/mysql-virtual-mailbox-domains.cf
virtual_mailbox_maps = mysql:/etc/postfix/mysql-virtual-mailbox-maps.cf
virtual_alias_maps = mysql:/etc/postfix/mysql-virtual-alias-maps.cf

# Mailbox delivery
virtual_mailbox_base = /var/vmail
virtual_uid_maps = static:8
virtual_gid_maps = static:8

# Authentication
smtpd_sasl_auth_enable = yes
smtpd_sasl_type = dovecot
smtpd_sasl_path = private/dovecot-auth
smtpd_sasl_security_options = noanonymous

# TLS
smtpd_tls_cert_file = /etc/ssl/certs/ssl-cert-snakeoil.pem
smtpd_tls_key_file = /etc/ssl/private/ssl-cert-snakeoil.key
smtpd_tls_security_level = may
smtp_tls_security_level = may

# Restrictions
smtpd_helo_required = yes
smtpd_helo_restrictions = permit_mynetworks, reject_invalid_helo_hostname, reject_non_fqdn_helo_hostname
smtpd_sender_restrictions = permit_mynetworks, reject_non_fqdn_sender, reject_unknown_sender_domain
smtpd_recipient_restrictions = permit_mynetworks, permit_sasl_authenticated, reject_unauth_destination, reject_non_fqdn_recipient, reject_unknown_recipient_domain

# Milter (Rspamd)
milter_protocol = 6
milter_default_action = accept
smtpd_milters = inet:localhost:11332
non_smtpd_milters = inet:localhost:11332

# Message size limit
message_size_limit = 52428800
`
	os.MkdirAll(PostfixConfDir, 0755)
	os.WriteFile(filepath.Join(PostfixConfDir, "main.cf"), []byte(config), 0644)

	// Generate MySQL lookup configs for Postfix
	s.generatePostfixMySQLConfigs()
}

// generatePostfixMySQLConfigs is kept for backward compatibility but the configs
// are now managed by entrypoint.sh which points to MariaDB
func (s *MailService) generatePostfixMySQLConfigs() {
	// Configs are now managed by entrypoint.sh pointing to MariaDB opanel_mail database
	// This method is kept for backward compatibility
}

// updateDovecotUserdb updates the Dovecot users file from mail_accounts table
func (s *MailService) updateDovecotUserdb() {
	rows, err := s.db.Query(`
		SELECT ma.username, md.name, ma.password, ma.quota
		FROM mail_accounts ma
		JOIN mail_domains md ON ma.domain_id = md.id
		WHERE ma.enabled = 1 AND md.enabled = 1
	`)
	if err != nil {
		slog.Warn("Failed to query mail accounts for userdb", "error", err)
		return
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("# Auto-generated by OPanel\n")
	sb.WriteString("# Do not edit manually\n\n")

	for rows.Next() {
		var username, domain, password string
		var quota int64
		if err := rows.Scan(&username, &domain, &password, &quota); err != nil {
			continue
		}

		fullEmail := username + "@" + domain

		sb.WriteString(fmt.Sprintf("%s:%s:5000:5000::/var/vmail/%s/%s::\n", fullEmail, password, domain, username))
	}

	os.MkdirAll(DovecotConfDir, 0755)
	os.WriteFile(filepath.Join(DovecotConfDir, "users"), []byte(sb.String()), 0644)
}

// --- Service Reload ---

// ReloadDovecot reloads the Dovecot service
func (s *MailService) ReloadDovecot() error {
	cmd := exec.Command("doveadm", "reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("Failed to reload Dovecot (non-fatal)", "output", string(output), "error", err)
		return fmt.Errorf("dovecot reload failed: %s: %w", string(output), err)
	}
	slog.Info("Reloaded Dovecot")
	return nil
}

// ReloadPostfix reloads the Postfix service
func (s *MailService) ReloadPostfix() error {
	cmd := exec.Command("postfix", "reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("Failed to reload Postfix (non-fatal)", "output", string(output), "error", err)
		return fmt.Errorf("postfix reload failed: %s: %w", string(output), err)
	}
	slog.Info("Reloaded Postfix")
	return nil
}
