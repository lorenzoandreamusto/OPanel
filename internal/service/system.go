package service

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

type SystemService struct{}

func NewSystemService() *SystemService {
	return &SystemService{}
}

// CreateUser creates a Linux user with useradd
// username: e.g. "op_demo" for domain "demo.com"
// homeDir: e.g. "/var/www/vhosts/demo.com"
func (s *SystemService) CreateUser(username, homeDir string) error {
	// Check if user already exists
	if _, err := user.Lookup(username); err == nil {
		return fmt.Errorf("user %s already exists", username)
	}

	// Create user with useradd
	// -d: home directory
	// -s /bin/false: no shell access
	// -M: don't create home dir (we create it ourselves)
	// -r: create system user
	cmd := exec.Command("useradd", "-d", homeDir, "-s", "/bin/false", "-M", "-r", username)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("useradd failed: %s: %w", string(output), err)
	}

	slog.Info("Created Linux user", "username", username, "home", homeDir)
	return nil
}

// DeleteUser deletes a Linux user with userdel
func (s *SystemService) DeleteUser(username string) error {
	cmd := exec.Command("userdel", "-r", username)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("userdel failed: %s: %w", string(output), err)
	}

	slog.Info("Deleted Linux user", "username", username)
	return nil
}

// EnsureGroup ensures a system group exists
func (s *SystemService) EnsureGroup(groupName string) error {
	// Check if group exists
	_, err := user.LookupGroup(groupName)
	if err == nil {
		return nil // group already exists
	}

	cmd := exec.Command("groupadd", groupName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("groupadd failed: %s: %w", string(output), err)
	}

	slog.Info("Created group", "group", groupName)
	return nil
}

// AddUserToGroup adds a user to a group
func (s *SystemService) AddUserToGroup(username, groupName string) error {
	cmd := exec.Command("usermod", "-aG", groupName, username)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("usermod failed: %s: %w", string(output), err)
	}

	slog.Info("Added user to group", "username", username, "group", groupName)
	return nil
}

// SetupSFTPChroot configures /etc/ssh/sshd_config for SFTP chroot
// It adds a Match Group block for opanel_users
func (s *SystemService) SetupSFTPChroot(sshdConfigPath, groupName string) error {
	// Read existing sshd_config
	data, err := os.ReadFile(sshdConfigPath)
	if err != nil {
		// If file doesn't exist, create it
		if os.IsNotExist(err) {
			data = []byte{}
		} else {
			return fmt.Errorf("failed to read sshd_config: %w", err)
		}
	}

	content := string(data)

	// Check if our block already exists
	matchMarker := fmt.Sprintf("Match Group %s", groupName)
	if strings.Contains(content, matchMarker) {
		slog.Info("SSHD chroot config already exists", "group", groupName)
		return nil
	}

	// Build the chroot block
	chrootBlock := fmt.Sprintf(`
# OPanel SFTP Chroot Configuration
Match Group %s
    ChrootDirectory /var/www/vhosts/%%u
    ForceCommand internal-sftp
    AllowTcpForwarding no
    X11Forwarding no
`, groupName)

	// Append to sshd_config
	f, err := os.OpenFile(sshdConfigPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open sshd_config for appending: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(chrootBlock); err != nil {
		return fmt.Errorf("failed to append to sshd_config: %w", err)
	}

	slog.Info("Added SFTP chroot config to sshd_config", "group", groupName)
	return nil
}

// ReloadSSHD restarts the sshd service
func (s *SystemService) ReloadSSHD() error {
	cmd := exec.Command("systemctl", "reload", "sshd")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try sshd without systemctl (for containers)
		cmd = exec.Command("sshd", "-t")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("sshd reload/test failed: %s: %w", string(output), err)
		}
	}

	slog.Info("Reloaded sshd service")
	return nil
}

// SetOwnership recursively changes ownership of a directory
func (s *SystemService) SetOwnership(path, username string) error {
	// Get UID/GID of user
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("failed to lookup user %s: %w", username, err)
	}

	uid := u.Uid
	gid := u.Gid

	cmd := exec.Command("chown", "-R", uid+":"+gid, path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("chown failed: %s: %w", string(output), err)
	}

	slog.Info("Set ownership", "path", path, "owner", username)
	return nil
}

// SetPermissions sets directory permissions
func (s *SystemService) SetPermissions(path string, mode os.FileMode) error {
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod failed: %w", err)
	}
	return nil
}

// GetUserHomeDir returns the home directory for a user
func (s *SystemService) GetUserHomeDir(username string) (string, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return "", fmt.Errorf("failed to lookup user %s: %w", username, err)
	}
	return u.HomeDir, nil
}

// UserExists checks if a Linux user exists
func (s *SystemService) UserExists(username string) bool {
	_, err := user.Lookup(username)
	return err == nil
}

// GetIPInterface gets the IP address of a network interface
func (s *SystemService) GetIPInterface(iface string) (string, error) {
	cmd := exec.Command("hostname", "-I")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get IP: %w", err)
	}

	ips := strings.Fields(strings.TrimSpace(string(output)))
	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses found")
	}

	// Return first non-loopback IP
	for _, ip := range ips {
		if ip != "127.0.0.1" {
			return ip, nil
		}
	}

	return ips[0], nil
}

// ReloadNginx reloads the nginx service
func (s *SystemService) ReloadNginx() error {
	// Test config first
	cmd := exec.Command("nginx", "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx config test failed: %s: %w", string(output), err)
	}

	// Try nginx -s reload directly (works with or without systemd)
	cmd = exec.Command("nginx", "-s", "reload")
	output, err = cmd.CombinedOutput()
	if err != nil {
		// If nginx is not running, try to start it
		cmd = exec.Command("nginx")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("nginx start/reload failed: %s: %w", string(output), err)
		}
	}

	slog.Info("Reloaded nginx service")
	return nil
}
