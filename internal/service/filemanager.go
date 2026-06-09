package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileManagerService struct {
	VhostsDir string
}

type FileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	Mode    string `json:"mode"`
	ModTime string `json:"mod_time"`
	Path    string `json:"path"`
}

func NewFileManagerService(vhostsDir string) *FileManagerService {
	return &FileManagerService{VhostsDir: vhostsDir}
}

// safePath resolves a path and ensures it's within the allowed base directory
func (s *FileManagerService) safePath(domain, path string) (string, error) {
	base := filepath.Join(s.VhostsDir, domain)
	full := filepath.Join(base, filepath.Clean(path))
	// Ensure the resolved path is within the base
	rel, err := filepath.Rel(base, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path outside allowed directory")
	}
	return full, nil
}

func (s *FileManagerService) ListFiles(domain, path string) ([]FileInfo, error) {
	full, err := s.safePath(domain, path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			Path:    filepath.Join(path, entry.Name()),
		})
	}
	return files, nil
}

func (s *FileManagerService) ReadFile(domain, path string) ([]byte, error) {
	full, err := s.safePath(domain, path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}
	return data, nil
}

func (s *FileManagerService) WriteFile(domain, path string, content []byte) error {
	full, err := s.safePath(domain, path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}
	return os.WriteFile(full, content, 0644)
}

func (s *FileManagerService) CreateDir(domain, path string) error {
	full, err := s.safePath(domain, path)
	if err != nil {
		return err
	}
	return os.MkdirAll(full, 0755)
}

func (s *FileManagerService) Delete(domain, path string) error {
	full, err := s.safePath(domain, path)
	if err != nil {
		return err
	}
	return os.RemoveAll(full)
}

func (s *FileManagerService) Rename(domain, oldPath, newPath string) error {
	oldFull, err := s.safePath(domain, oldPath)
	if err != nil {
		return err
	}
	newFull, err := s.safePath(domain, newPath)
	if err != nil {
		return err
	}
	return os.Rename(oldFull, newFull)
}

func (s *FileManagerService) UploadFile(domain, path string, reader io.Reader) error {
	full, err := s.safePath(domain, path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}
	f, err := os.Create(full)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer f.Close()
	_, err = io.Copy(f, reader)
	return err
}
