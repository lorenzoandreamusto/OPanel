package service

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

type TerminalService struct {
	mu sync.Mutex
}

type TerminalSession struct {
	Pty *os.File
	Cmd *exec.Cmd
}

func NewTerminalService() *TerminalService {
	return &TerminalService{}
}

func (s *TerminalService) CreateSession(width, height uint16) (*TerminalSession, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}

	if err := pty.Setsize(ptmx, &pty.Winsize{
		Rows: height,
		Cols: width,
	}); err != nil {
		ptmx.Close()
		return nil, fmt.Errorf("failed to set pty size: %w", err)
	}

	return &TerminalSession{
		Pty: ptmx,
		Cmd: cmd,
	}, nil
}

func (s *TerminalService) Resize(ptmx *os.File, width, height uint16) error {
	return pty.Setsize(ptmx, &pty.Winsize{
		Rows: height,
		Cols: width,
	})
}

func (s *TerminalSession) Close() error {
	if s.Pty != nil {
		s.Pty.Close()
	}
	if s.Cmd != nil && s.Cmd.Process != nil {
		s.Cmd.Process.Kill()
		s.Cmd.Wait()
	}
	return nil
}

// Pipe proxies between a reader/writer (websocket) and the pty
func (s *TerminalSession) Pipe(ws io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)

	// PTY -> WebSocket
	go func() {
		defer wg.Done()
		io.Copy(ws, s.Pty)
	}()

	// WebSocket -> PTY
	go func() {
		defer wg.Done()
		io.Copy(s.Pty, ws)
	}()

	wg.Wait()
}
