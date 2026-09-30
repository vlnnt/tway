//go:build windows

package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func OpenTerminal(
	mode string,
) error {
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
		"/c",
		exePath,
		mode,
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start terminal: %w", err)
	}

	return nil
}

func StartDetached(
	args ...string,
) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf(
			"get executable path: %w",
			err,
		)
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf(
			"get absolute executable path: %w",
			err,
		)
	}

	cmd := exec.Command(
		exePath,
		args...,
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf(
			"start detached tway: %w",
			err,
		)
	}

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf(
			"release detached tway process: %w",
			err,
		)
	}

	return nil
}
