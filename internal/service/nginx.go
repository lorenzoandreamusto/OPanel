package service

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

type NginxTemplateData struct {
	Domain        string
	DocumentRoot  string
	LogDir        string
	PHPFPM_SOCKET string
	SSLCertPath   string
	SSLKeyPath    string
	IsSSL         bool
}

type NginxService struct {
	TemplateDir string
	ConfigDir   string
}

func NewNginxService(templateDir, configDir string) *NginxService {
	return &NginxService{
		TemplateDir: templateDir,
		ConfigDir:   configDir,
	}
}

// GenerateConfig generates an Nginx virtual host config for a domain
func (n *NginxService) GenerateConfig(data NginxTemplateData) error {
	templatePath := filepath.Join(n.TemplateDir, "nginx", "default.conf.template")

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse nginx template: %w", err)
	}

	outputPath := filepath.Join(n.ConfigDir, fmt.Sprintf("%s.conf", data.Domain))

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create nginx config: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("failed to execute nginx template: %w", err)
	}

	slog.Info("Generated nginx config", "domain", data.Domain, "path", outputPath)
	return nil
}

// RemoveConfig removes the Nginx config for a domain
func (n *NginxService) RemoveConfig(domain string) error {
	configPath := filepath.Join(n.ConfigDir, fmt.Sprintf("%s.conf", domain))

	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove nginx config: %w", err)
	}

	slog.Info("Removed nginx config", "domain", domain)
	return nil
}

// TestConfig tests the Nginx configuration
func (n *NginxService) TestConfig() error {
	cmd := exec.Command("nginx", "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx config test failed: %s: %w", string(output), err)
	}
	return nil
}

// Reload reloads the Nginx service
func (n *NginxService) Reload() error {
	if err := n.TestConfig(); err != nil {
		return err
	}

	// Try nginx -s reload directly (works with or without systemd)
	cmd := exec.Command("nginx", "-s", "reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If nginx is not running, try to start it
		cmd = exec.Command("nginx")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("nginx start/reload failed: %s: %w", string(output), err)
		}
	}

	slog.Info("Reloaded nginx")
	return nil
}
