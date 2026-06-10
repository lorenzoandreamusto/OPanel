package service

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"opanel/internal/database"
	"opanel/internal/model"
)

const (
	Bind9ZoneDir     = "/etc/bind/zones"
	Bind9ConfDir     = "/etc/bind"
	DefaultTTTL      = 3600
	DefaultSOASerial = 2026060101
	DefaultSOARefresh = 3600
	DefaultSOARetry  = 900
	DefaultSOAExpire = 604800
	DefaultSOAMinimum = 86400
)

type DNSService struct {
	db *database.DB
}

func NewDNSService(db *database.DB) *DNSService {
	return &DNSService{db: db}
}

func (s *DNSService) EnsureZoneDir() error {
	return os.MkdirAll(Bind9ZoneDir, 0755)
}

// CreateZone creates a DNS zone for a domain
func (s *DNSService) CreateZone(req *model.CreateDNSZoneRequest) (*model.DNSZone, error) {
	// Check if zone already exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM dns_zones WHERE domain_id = ?", req.DomainID).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check zone: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("zone already exists for this domain")
	}

	// Get domain name
	var domainName string
	err = s.db.QueryRow("SELECT name FROM domains WHERE id = ?", req.DomainID).Scan(&domainName)
	if err != nil {
		return nil, fmt.Errorf("domain not found")
	}

	result, err := s.db.Exec(
		"INSERT INTO dns_zones (domain_id, name, enabled) VALUES (?, ?, 1)",
		req.DomainID, domainName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create zone: %w", err)
	}

	id, _ := result.LastInsertId()

	// Create default records
	s.createDefaultRecords(int(id), domainName)

	// Generate zone file
	if err := s.GenerateZoneFile(int(id)); err != nil {
		slog.Warn("Failed to generate zone file (non-fatal)", "error", err)
	}

	// Reload Bind9
	s.Reload()

	slog.Info("DNS zone created", "domain", domainName, "id", id)
	return s.GetZone(int(id))
}

func (s *DNSService) createDefaultRecords(zoneID int, domainName string) {
	defaultRecords := []struct {
		Type  string
		Name  string
		Value string
		TTL   int
	}{
		{"NS", "@", "ns1." + domainName, DefaultTTTL},
		{"A", "@", "0.0.0.0", DefaultTTTL},
		{"A", "www", "0.0.0.0", DefaultTTTL},
		{"MX", "@", "mail." + domainName, 3600},
	}

	for _, r := range defaultRecords {
		s.db.Exec(
			"INSERT INTO dns_records (zone_id, type, name, value, ttl, priority, enabled) VALUES (?, ?, ?, ?, ?, 0, 1)",
			zoneID, r.Type, r.Name, r.Value, r.TTL,
		)
	}
}

// GetZone returns a DNS zone by ID
func (s *DNSService) GetZone(id int) (*model.DNSZone, error) {
	var z model.DNSZone
	err := s.db.QueryRow(
		"SELECT id, domain_id, name, enabled, created_at, updated_at FROM dns_zones WHERE id = ?", id,
	).Scan(&z.ID, &z.DomainID, &z.Name, &z.Enabled, &z.CreatedAt, &z.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("zone not found")
		}
		return nil, fmt.Errorf("failed to fetch zone: %w", err)
	}
	return &z, nil
}

// GetZoneByDomain returns the DNS zone for a given domain ID
func (s *DNSService) GetZoneByDomain(domainID int) (*model.DNSZone, error) {
	var z model.DNSZone
	err := s.db.QueryRow(
		"SELECT id, domain_id, name, enabled, created_at, updated_at FROM dns_zones WHERE domain_id = ?", domainID,
	).Scan(&z.ID, &z.DomainID, &z.Name, &z.Enabled, &z.CreatedAt, &z.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("zone not found")
		}
		return nil, fmt.Errorf("failed to fetch zone: %w", err)
	}
	return &z, nil
}

// ListZones returns all DNS zones
func (s *DNSService) ListZones() ([]model.DNSZone, error) {
	rows, err := s.db.Query("SELECT id, domain_id, name, enabled, created_at, updated_at FROM dns_zones ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to query zones: %w", err)
	}
	defer rows.Close()

	var zones []model.DNSZone
	for rows.Next() {
		var z model.DNSZone
		if err := rows.Scan(&z.ID, &z.DomainID, &z.Name, &z.Enabled, &z.CreatedAt, &z.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan zone: %w", err)
		}
		zones = append(zones, z)
	}
	if zones == nil {
		zones = []model.DNSZone{}
	}
	return zones, nil
}

// DeleteZone removes a DNS zone
func (s *DNSService) DeleteZone(id int) error {
	zone, err := s.GetZone(id)
	if err != nil {
		return err
	}

	// Delete records
	s.db.Exec("DELETE FROM dns_records WHERE zone_id = ?", id)

	// Remove zone file
	zoneFile := filepath.Join(Bind9ZoneDir, fmt.Sprintf("db.%s", zone.Name))
	os.Remove(zoneFile)

	// Delete from DB
	s.db.Exec("DELETE FROM dns_zones WHERE id = ?", id)

	// Reload Bind9
	s.Reload()

	slog.Info("DNS zone deleted", "domain", zone.Name, "id", id)
	return nil
}

// ListRecords returns all records for a zone
func (s *DNSService) ListRecords(zoneID int) ([]model.DNSRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, zone_id, type, name, value, ttl, priority, enabled FROM dns_records WHERE zone_id = ? ORDER BY type, name", zoneID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var records []model.DNSRecord
	for rows.Next() {
		var r model.DNSRecord
		if err := rows.Scan(&r.ID, &r.ZoneID, &r.Type, &r.Name, &r.Value, &r.TTL, &r.Priority, &r.Enabled); err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}
		records = append(records, r)
	}
	if records == nil {
		records = []model.DNSRecord{}
	}
	return records, nil
}

// CreateRecord adds a DNS record to a zone
func (s *DNSService) CreateRecord(zoneID int, req *model.CreateDNSRecordRequest) (*model.DNSRecord, error) {
	if req.TTL == 0 {
		req.TTL = DefaultTTTL
	}

	result, err := s.db.Exec(
		"INSERT INTO dns_records (zone_id, type, name, value, ttl, priority, enabled) VALUES (?, ?, ?, ?, ?, ?, 1)",
		zoneID, req.Type, req.Name, req.Value, req.TTL, req.Priority,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	id, _ := result.LastInsertId()

	// Regenerate zone file and reload
	zone, _ := s.GetZone(zoneID)
	if zone != nil {
		s.GenerateZoneFile(zoneID)
		s.Reload()
	}

	slog.Info("DNS record created", "zone_id", zoneID, "type", req.Type, "name", req.Name)
	return s.GetRecord(int(id))
}

// GetRecord returns a DNS record by ID
func (s *DNSService) GetRecord(id int) (*model.DNSRecord, error) {
	var r model.DNSRecord
	err := s.db.QueryRow(
		"SELECT id, zone_id, type, name, value, ttl, priority, enabled FROM dns_records WHERE id = ?", id,
	).Scan(&r.ID, &r.ZoneID, &r.Type, &r.Name, &r.Value, &r.TTL, &r.Priority, &r.Enabled)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("record not found")
		}
		return nil, fmt.Errorf("failed to fetch record: %w", err)
	}
	return &r, nil
}

// UpdateRecord updates a DNS record
func (s *DNSService) UpdateRecord(id int, req *model.UpdateDNSRecordRequest) (*model.DNSRecord, error) {
	setClauses := []string{}
	args := []interface{}{}

	if req.Type != "" {
		setClauses = append(setClauses, "type = ?")
		args = append(args, req.Type)
	}
	if req.Name != "" {
		setClauses = append(setClauses, "name = ?")
		args = append(args, req.Name)
	}
	if req.Value != "" {
		setClauses = append(setClauses, "value = ?")
		args = append(args, req.Value)
	}
	if req.TTL != 0 {
		setClauses = append(setClauses, "ttl = ?")
		args = append(args, req.TTL)
	}
	if req.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *req.Priority)
	}
	if req.Enabled != nil {
		setClauses = append(setClauses, "enabled = ?")
		args = append(args, *req.Enabled)
	}

	if len(setClauses) == 0 {
		return s.GetRecord(id)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE dns_records SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}

	// Get zone ID for regeneration
	var zoneID int
	s.db.QueryRow("SELECT zone_id FROM dns_records WHERE id = ?", id).Scan(&zoneID)
	if zoneID > 0 {
		s.GenerateZoneFile(zoneID)
		s.Reload()
	}

	return s.GetRecord(id)
}

// DeleteRecord removes a DNS record
func (s *DNSService) DeleteRecord(id int) error {
	var zoneID int
	s.db.QueryRow("SELECT zone_id FROM dns_records WHERE id = ?", id).Scan(&zoneID)

	_, err := s.db.Exec("DELETE FROM dns_records WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	if zoneID > 0 {
		s.GenerateZoneFile(zoneID)
		s.Reload()
	}

	slog.Info("DNS record deleted", "id", id)
	return nil
}

type zoneFileData struct {
	Domain  string
	Serial  int
	Refresh int
	Retry   int
	Expire  int
	Minimum int
	NS      string
	Records []model.DNSRecord
}

// GenerateZoneFile writes a Bind9 zone file for a zone
func (s *DNSService) GenerateZoneFile(zoneID int) error {
	zone, err := s.GetZone(zoneID)
	if err != nil {
		return err
	}

	records, err := s.ListRecords(zoneID)
	if err != nil {
		return err
	}

	serial := time.Now().Format("20060102") + "01"

	data := zoneFileData{
		Domain:  zone.Name,
		Serial:  DefaultSOASerial,
		Refresh: DefaultSOARefresh,
		Retry:   DefaultSOARetry,
		Expire:  DefaultSOAExpire,
		Minimum: DefaultSOAMinimum,
		Records: records,
	}
	fmt.Sscanf(serial, "%d", &data.Serial)

	tmplStr := `$TTL 3600
@   IN  SOA ns1.{{.Domain}}. admin.{{.Domain}}. (
        {{.Serial}}   ; Serial
        {{.Refresh}}  ; Refresh
        {{.Retry}}    ; Retry
        {{.Expire}}   ; Expire
        {{.Minimum}}  ; Minimum TTL
)

; Nameservers
@   IN  NS  ns1.{{.Domain}}.

; DNS Records
{{- range .Records}}
{{- if eq .Type "A"}}
{{.Name}}  IN  A   {{.Value}}
{{- else if eq .Type "AAAA"}}
{{.Name}}  IN  AAAA  {{.Value}}
{{- else if eq .Type "CNAME"}}
{{.Name}}  IN  CNAME  {{.Value}}.
{{- else if eq .Type "MX"}}
{{.Name}}  IN  MX  {{.Priority}}  {{.Value}}.
{{- else if eq .Type "TXT"}}
{{.Name}}  IN  TXT  "{{.Value}}"
{{- else if eq .Type "SRV"}}
{{.Name}}  IN  SRV  {{.Priority}}  0  0  {{.Value}}
{{- else if eq .Type "NS"}}
{{.Name}}  IN  NS  {{.Value}}.
{{- end}}
{{- end}}
`
	tmpl, err := template.New("zone").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("failed to parse zone template: %w", err)
	}

	if err := s.EnsureZoneDir(); err != nil {
		return fmt.Errorf("failed to create zone directory: %w", err)
	}

	zoneFile := filepath.Join(Bind9ZoneDir, fmt.Sprintf("db.%s", zone.Name))
	f, err := os.Create(zoneFile)
	if err != nil {
		return fmt.Errorf("failed to create zone file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		os.Remove(zoneFile)
		return fmt.Errorf("failed to execute zone template: %w", err)
	}

	slog.Info("Generated Bind9 zone file", "domain", zone.Name, "path", zoneFile)
	return nil
}

// Reload reloads Bind9
func (s *DNSService) Reload() error {
	cmd := exec.Command("rndc", "reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try named restart
		cmd = exec.Command("service", "bind9", "restart")
		output, err = cmd.CombinedOutput()
		if err != nil {
			slog.Warn("Failed to reload/restart Bind9 (non-fatal)", "output", string(output), "error", err)
			return fmt.Errorf("bind9 reload failed: %s: %w", string(output), err)
		}
	}
	slog.Info("Reloaded Bind9")
	return nil
}

// GenerateNamedConf generates named.conf.local with zone declarations
func (s *DNSService) GenerateNamedConf() error {
	zones, err := s.ListZones()
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("// Auto-generated by OPanel\n")
	sb.WriteString("// Do not edit manually\n\n")

	for _, zone := range zones {
		if !zone.Enabled {
			continue
		}
		sb.WriteString(fmt.Sprintf(`zone "%s" {
    type master;
    file "/etc/bind/zones/db.%s";
    allow-query { any; };
    allow-transfer { none; };
};

`, zone.Name, zone.Name))
	}

	namedConfLocal := filepath.Join(Bind9ConfDir, "named.conf.local")
	f, err := os.Create(namedConfLocal)
	if err != nil {
		return fmt.Errorf("failed to create named.conf.local: %w", err)
	}
	defer f.Close()

	f.WriteString(sb.String())
	slog.Info("Generated named.conf.local", "zones", len(zones))
	return nil
}

// CreateMailRecords adds mail-related DNS records (MX, SPF, DKIM, A for mail subdomain)
func (s *DNSService) CreateMailRecords(zoneID int, domainName, serverIP, dkimPublicKey string) error {
	mailRecords := []model.CreateDNSRecordRequest{
		{Type: "A", Name: "mail", Value: serverIP, TTL: DefaultTTTL},
		{Type: "MX", Name: "@", Value: "mail." + domainName, TTL: DefaultTTTL, Priority: 10},
		{Type: "TXT", Name: "@", Value: "v=spf1 a mx ip4:" + serverIP + " -all", TTL: DefaultTTTL},
	}

	if dkimPublicKey != "" {
		mailRecords = append(mailRecords, model.CreateDNSRecordRequest{
			Type: "TXT", Name: "default._domainkey", Value: dkimPublicKey, TTL: DefaultTTTL,
		})
	}

	for _, req := range mailRecords {
		if _, err := s.CreateRecord(zoneID, &req); err != nil {
			slog.Warn("Failed to create mail DNS record (non-fatal)", "type", req.Type, "error", err)
		}
	}

	slog.Info("Created mail DNS records", "zone_id", zoneID, "domain", domainName)
	return nil
}

// RemoveMailRecords removes mail-related DNS records from a zone
func (s *DNSService) RemoveMailRecords(zoneID int) error {
	records, err := s.ListRecords(zoneID)
	if err != nil {
		return err
	}

	for _, r := range records {
		if r.Type == "MX" ||
			(r.Type == "TXT" && (strings.Contains(r.Value, "v=spf1") || strings.Contains(r.Value, "v=DKIM1"))) ||
			(r.Name == "mail" && r.Type == "A") {
			s.DeleteRecord(r.ID)
		}
	}

	slog.Info("Removed mail DNS records", "zone_id", zoneID)
	return nil
}

// GetServerIP returns the server's primary IP address
func GetServerIP() string {
	cmd := exec.Command("hostname", "-I")
	output, err := cmd.Output()
	if err != nil {
		return "127.0.0.1"
	}
	ips := strings.Fields(string(output))
	if len(ips) > 0 {
		return ips[0]
	}
	return "127.0.0.1"
}
