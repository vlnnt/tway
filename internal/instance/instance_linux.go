//go:build linux

package instance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func Acquire() (*Lock, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf(
			"get user cache directory: %w",
			err,
		)
	}

	lockDir := filepath.Join(cacheDir, "tway")
	if err := os.MkdirAll(
		lockDir,
		0700,
	); err != nil {
		return nil, fmt.Errorf(
			"create lock directory: %w",
			err,
		)
	}

	lockPath := filepath.Join(
		lockDir,
		"instance.lock",
	)

	file, err := os.OpenFile(
		lockPath,
		os.O_CREATE|os.O_RDWR,
		0600,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open instance lock: %w",
			err,
		)
	}

	if err := unix.Flock(
		int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB,
	); err != nil {
		_ = file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) ||
			errors.Is(err, unix.EAGAIN) {
			return nil, ErrAlreadyRunning
		}

		return nil, fmt.Errorf(
			"lock instance file: %w",
			err,
		)
	}

	return &Lock{
		release: func() error {
			unlockErr := unix.Flock(int(file.Fd()), unix.LOCK_UN)
			closeErr := file.Close()

			if unlockErr != nil {
				return fmt.Errorf(
					"unlock instance file: %w",
					unlockErr,
				)
			}

			if closeErr != nil {
				return fmt.Errorf(
					"close instance lock: %w",
					closeErr,
				)
			}

			return nil
		},
	}, nil
}
