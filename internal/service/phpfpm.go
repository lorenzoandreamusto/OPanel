package service

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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
func (s *PHPFPMService) CreatePool(domain, username, documentRoot string) error {
	templatePath := filepath.Join(s.TemplateDir, "phpfpm", "pool.conf.template")

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse phpfpm template: %w", err)
	}

	socketPath := filepath.Join(s.SocketDir, fmt.Sprintf("php%s-fpm-%s.sock", s.PHPVersion, username))

	data := PHPFPMTemplateData{
		Username:     username,
		SocketPath:   socketPath,
		Domain:       domain,
		DocumentRoot: documentRoot,
		PHPVersion:   s.PHPVersion,
	}

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

// Reload restarts PHP-FPM service
func (s *PHPFPMService) Reload() error {
	binary := fmt.Sprintf("php-fpm%s", s.PHPVersion)

	// Test config first
	cmd := exec.Command(binary, "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("phpfpm config test failed (non-fatal)", "output", string(output), "error", err)
	}

	// Try killall + start (works in Docker without systemd)
	cmd = exec.Command("killall", binary)
	_ = cmd.Run() // ignore error if not running

	// Start PHP-FPM in background
	cmd = exec.Command(binary)
	if err := cmd.Start(); err != nil {
		slog.Warn("phpfpm start failed (non-fatal)", "error", err)
	} else {
		go cmd.Wait()
	}

	slog.Info("Reloaded phpfpm", "version", s.PHPVersion)
	return nil
}

// GetSocketPath returns the socket path for a domain
func (s *PHPFPMService) GetSocketPath(domain string) string {
	username := "op_" + domain
	return filepath.Join(s.SocketDir, fmt.Sprintf("php%s-fpm-%s.sock", s.PHPVersion, username))
}
