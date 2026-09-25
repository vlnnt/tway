//go:build linux

package tui

import (
	"os"

	"golang.org/x/sys/unix"
)

func AttachConsole() error {
	if _, err := unix.IoctlGetTermios(
		int(os.Stdin.Fd()),
		unix.TCGETS,
	); err != nil {
		return ErrNoConsole
	}

	if _, err := unix.IoctlGetTermios(
		int(os.Stdout.Fd()),
		unix.TCGETS,
	); err != nil {
		return ErrNoConsole
	}

	return nil
}
