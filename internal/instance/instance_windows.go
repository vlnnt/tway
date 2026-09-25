//go:build windows

package instance

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

func Acquire() (*Lock, error) {
	name, err := windows.UTF16PtrFromString("Local\\TwaySingleInstance")
	if err != nil {
		return nil, fmt.Errorf("create mutex name: %w", err)
	}

	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		if errors.Is(
			err, windows.ERROR_ALREADY_EXISTS,
		) {
			if handle != 0 {
				_ = windows.CloseHandle(handle)
			}

			return nil, ErrAlreadyRunning
		}

		if handle != 0 {
			_ = windows.CloseHandle(handle)
		}

		return nil, fmt.Errorf("create instance mutex: %w", err)
	}

	return &Lock{
		release: func() error {
			if err := windows.CloseHandle(handle); err != nil {
				return fmt.Errorf("close instance mutex: %w", err)
			}
			return nil
		},
	}, nil
}
