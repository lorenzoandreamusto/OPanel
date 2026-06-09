package service

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type BackupService struct {
	DB         *sql.DB
	BackupsDir string
	VhostsDir  string
}

type Backup struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	DomainID  int64     `json:"domain_id"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

func NewBackupService(db *sql.DB, backupsDir, vhostsDir string) *BackupService {
	return &BackupService{
		DB:         db,
		BackupsDir: backupsDir,
		VhostsDir:  vhostsDir,
	}
}

func (s *BackupService) CreateBackup(domainID int64, domainName, name string) (*Backup, error) {
	if name == "" {
		name = fmt.Sprintf("%s_%s", domainName, time.Now().Format("20060102_150405"))
	}

	if err := os.MkdirAll(s.BackupsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backups directory: %w", err)
	}

	archivePath := filepath.Join(s.BackupsDir, name+".tar.gz")

	if err := s.createArchive(domainName, archivePath); err != nil {
		return nil, fmt.Errorf("failed to create archive: %w", err)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return nil, err
	}

	result, err := s.DB.Exec(
		"INSERT INTO backups (name, domain_id, size, status) VALUES (?, ?, ?, 'completed')",
		name, domainID, info.Size(),
	)
	if err != nil {
		os.Remove(archivePath)
		return nil, err
	}

	id, _ := result.LastInsertId()

	return &Backup{
		ID:        id,
		Name:      name,
		DomainID:  domainID,
		Size:      info.Size(),
		CreatedAt: time.Now(),
		Status:    "completed",
	}, nil
}

func (s *BackupService) createArchive(domainName, archivePath string) error {
	domainDir := filepath.Join(s.VhostsDir, domainName)

	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(domainDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		rel, err := filepath.Rel(domainDir, path)
		if err != nil {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return nil
		}
		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})
}

func (s *BackupService) ListBackups(domainID int64) ([]Backup, error) {
	rows, err := s.DB.Query(
		"SELECT id, name, domain_id, size, created_at, status FROM backups WHERE domain_id = ? ORDER BY created_at DESC",
		domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []Backup
	for rows.Next() {
		var b Backup
		if err := rows.Scan(&b.ID, &b.Name, &b.DomainID, &b.Size, &b.CreatedAt, &b.Status); err != nil {
			continue
		}
		backups = append(backups, b)
	}
	return backups, nil
}

func (s *BackupService) GetBackupPath(backupID int64) (string, error) {
	var name string
	err := s.DB.QueryRow("SELECT name FROM backups WHERE id = ?", backupID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("backup not found")
	}

	path := filepath.Join(s.BackupsDir, name+".tar.gz")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("backup file not found")
	}

	return path, nil
}

func (s *BackupService) DeleteBackup(backupID int64) error {
	var name string
	err := s.DB.QueryRow("SELECT name FROM backups WHERE id = ?", backupID).Scan(&name)
	if err != nil {
		return fmt.Errorf("backup not found")
	}

	path := filepath.Join(s.BackupsDir, name+".tar.gz")
	os.Remove(path) // Ignore error if file doesn't exist

	_, err = s.DB.Exec("DELETE FROM backups WHERE id = ?", backupID)
	return err
}

func (s *BackupService) RestoreBackup(backupID int64, domainName string) error {
	path, err := s.GetBackupPath(backupID)
	if err != nil {
		return err
	}

	domainDir := filepath.Join(s.VhostsDir, domainName)

	if err := os.MkdirAll(domainDir, 0755); err != nil {
		return fmt.Errorf("failed to create domain directory: %w", err)
	}

	cmd := exec.Command("tar", "-xzf", path, "-C", domainDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}
