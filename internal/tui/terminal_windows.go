//go:build windows

package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func OpenTerminal() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("get absolute executable path: %w", err)
	}

	cmd := exec.Command(
		"cmd.exe",
		"/c",
		"start",
		"tway - Streamers",
		"cmd.exe",
		"/k",
		exePath,
		"--tui",
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start terminal: %w", err)
	}

	return nil
}
