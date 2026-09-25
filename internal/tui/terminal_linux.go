//go:build linux

package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
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

	terminals := []struct {
		command string
		args    []string
	}{
		{
			"kitty",
			[]string{exePath, mode},
		},
		{
			"alacritty",
			[]string{"-e", exePath, mode},
		},
		{
			"gnome-terminal",
			[]string{"--", exePath, mode},
		},
		{
			"konsole",
			[]string{"-e", exePath, mode},
		},
		{
			"xfce4-terminal",
			[]string{"--command", fmt.Sprintf("%q %s", exePath, mode)},
		},
		{
			"x-terminal-emulator",
			[]string{"-e", exePath, mode},
		},
	}

	for _, terminal := range terminals {
		if _, err := exec.LookPath(terminal.command); err != nil {
			continue
		}

		cmd := exec.Command(terminal.command, terminal.args...)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start terminal: %w", err)
		}

		return nil
	}

	return fmt.Errorf("no supported terminal emulator found")
}

func StartDetached() error {
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

	cmd := exec.Command(exePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

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
