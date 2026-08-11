//go:build linux

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

	terminals := []struct {
		command string
		args    []string
	}{
		{
			"kitty",
			[]string{exePath, "--tui"},
		},
		{
			"alacritty",
			[]string{"-e", exePath, "--tui"},
		},
		{
			"gnome-terminal",
			[]string{"--", exePath, "--tui"},
		},
		{
			"konsole",
			[]string{"-e", exePath, "--tui"},
		},
		{
			"xfce4-terminal",
			[]string{"--command", exePath + " --tui"},
		},
		{
			"x-terminal-emulator",
			[]string{"-e", exePath, "--tui"},
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
