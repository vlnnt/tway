//go:build windows

package tui

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

var (
	kernel32          = windows.NewLazySystemDLL("kernel32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
)

const attachParentProcess = ^uint32(0)

func AttachConsole() error {
	r1, _, err := procAttachConsole.Call(
		uintptr(attachParentProcess),
	)

	if r1 == 0 {
		if err != windows.ERROR_ACCESS_DENIED {
			return fmt.Errorf("attach console: %w", err)
		}
	}

	stdout, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open console output: %w", err)
	}

	stderr, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		stdout.Close()
		return fmt.Errorf("open console error output: %w", err)
	}

	stdin, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		stdout.Close()
		stderr.Close()
		return fmt.Errorf("open console input: %w", err)
	}

	os.Stdout = stdout
	os.Stderr = stderr
	os.Stdin = stdin

	return nil
}
