package service

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type PHPFPMTemplateData struct {
	Username     string
	SocketPath   string
	Domain       string
	DocumentRoot string
	PHPVersion   string
}

type PHPFPMService struct {
	TemplateDir string
	PHPVersion  string
	PoolDir     string
	SocketDir   string
}

func NewPHPFPMService(templateDir, phpVersion, poolDir, socketDir string) *PHPFPMService {
	return &PHPFPMService{
		TemplateDir: templateDir,
		PHPVersion:  phpVersion,
		PoolDir:     poolDir,
		SocketDir:   socketDir,
	}
}

// CreatePool generates a PHP-FPM pool config for a domain
func (s *PHPFPMService) CreatePool(domain, username, documentRoot, phpVersion string) error {
	templatePath := filepath.Join(s.TemplateDir, "phpfpm", "pool.conf.template")

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse phpfpm template: %w", err)
	}

	socketPath := filepath.Join(s.SocketDir, fmt.Sprintf("php%s-fpm-%s.sock", phpVersion, username))

	data := PHPFPMTemplateData{
		Username:     username,
		SocketPath:   socketPath,
		Domain:       domain,
		DocumentRoot: documentRoot,
		PHPVersion:   phpVersion,
	}

	// Create log directory for this PHP version if it doesn't exist
	logDir := fmt.Sprintf("/var/log/php%s-fpm", phpVersion)
	_ = os.MkdirAll(logDir, 0755)

	outputPath := filepath.Join(s.PoolDir, fmt.Sprintf("%s.pool.conf", domain))

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create phpfpm pool config: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("failed to execute phpfpm template: %w", err)
	}

	slog.Info("Generated phpfpm pool config", "domain", domain, "path", outputPath)
	return nil
}

// RemovePool removes the PHP-FPM pool config for a domain
func (s *PHPFPMService) RemovePool(domain string) error {
	configPath := filepath.Join(s.PoolDir, fmt.Sprintf("%s.pool.conf", domain))

	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove phpfpm pool config: %w", err)
	}

	slog.Info("Removed phpfpm pool config", "domain", domain)
	return nil
}

// Reload sends SIGUSR2 to PHP-FPM master to gracefully reload config
func (s *PHPFPMService) Reload() error {
	binary := filepath.Join("/usr/sbin", fmt.Sprintf("php-fpm%s", s.PHPVersion))

	// Test config first
	cmd := exec.Command(binary, "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("phpfpm config test failed (non-fatal)", "output", string(output), "error", err)
	}

	// Try graceful reload: send SIGUSR2 to master process
	reloadCmd := exec.Command("sh", "-c", fmt.Sprintf("pkill -USR2 -f 'php-fpm: master process' 2>/dev/null"))
	reloadOutput, reloadErr := reloadCmd.CombinedOutput()
	if reloadErr == nil && len(strings.TrimSpace(string(reloadOutput))) == 0 {
		slog.Info("Reloaded phpfpm via SIGUSR2", "version", s.PHPVersion)
		return nil
	}
	slog.Warn("phpfpm SIGUSR2 failed, trying restart", "output", string(reloadOutput), "error", reloadErr)

	// Fallback: kill and restart via sh -c (avoids zombie processes from Go child)
	restartScript := fmt.Sprintf(
		"pkill -TERM -f 'php-fpm: master' 2>/dev/null; sleep 0.5; nohup %s --nodaemonize >/dev/null 2>&1 &",
		binary,
	)
	restartCmd := exec.Command("sh", "-c", restartScript)
	restartOutput, restartErr := restartCmd.CombinedOutput()
	if restartErr != nil {
		slog.Warn("phpfpm restart failed (non-fatal)", "output", string(restartOutput), "error", restartErr)
	}

	slog.Info("Reloaded phpfpm (restart fallback)", "version", s.PHPVersion)
	return nil
}

// GetSocketPath returns the socket path for a domain
func (s *PHPFPMService) GetSocketPath(domain, phpVersion string) string {
	username := "op_" + strings.ReplaceAll(domain, ".", "-")
	return filepath.Join(s.SocketDir, fmt.Sprintf("php%s-fpm-%s.sock", phpVersion, username))
}
