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
	pidFile := "/var/run/php/php8.4-fpm.pid"

	// Test config first
	cmd := exec.Command(binary, "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("phpfpm config test failed (non-fatal)", "output", string(output), "error", err)
	}

	// Try graceful reload via PID file
	if pidData, readErr := os.ReadFile(pidFile); readErr == nil {
		pid := strings.TrimSpace(string(pidData))
		reloadCmd := exec.Command("kill", "-USR2", pid)
		reloadOutput, reloadErr := reloadCmd.CombinedOutput()
		if reloadErr == nil {
			slog.Info("Reloaded phpfpm via SIGUSR2", "version", s.PHPVersion, "pid", pid)
			return nil
		}
		slog.Warn("phpfpm SIGUSR2 failed", "pid", pid, "output", string(reloadOutput), "error", reloadErr)
	} else {
		slog.Warn("phpfpm PID file not found", "path", pidFile, "error", readErr)
	}

	// Fallback: kill old master and start new one detached
	if pidData, readErr := os.ReadFile(pidFile); readErr == nil {
		pid := strings.TrimSpace(string(pidData))
		exec.Command("kill", "-TERM", pid).Run()
	}

	startCmd := exec.Command("setsid", binary, "--allow-to-run-as-root", "--nodaemonize")
	startCmd.Stdout = nil
	startCmd.Stderr = nil
	startErr := startCmd.Start()
	if startErr != nil {
		slog.Warn("phpfpm restart failed (non-fatal)", "error", startErr)
	} else {
		startCmd.Process.Release()
		slog.Info("Restarted phpfpm (fallback)", "version", s.PHPVersion)
	}

	return nil
}

// GetSocketPath returns the socket path for a domain
func (s *PHPFPMService) GetSocketPath(domain, phpVersion string) string {
	username := "op_" + strings.ReplaceAll(domain, ".", "-")
	return filepath.Join(s.SocketDir, fmt.Sprintf("php%s-fpm-%s.sock", phpVersion, username))
}
