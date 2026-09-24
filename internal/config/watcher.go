package config

import (
	"context"
	"os"
	"time"
)

func Watch(
	ctx context.Context,
	path string,
	interval time.Duration,
) (<-chan struct{}, <-chan error) {
	changes := make(chan struct{})
	errors := make(chan error)

	go func() {
		defer close(changes)
		defer close(errors)

		info, err := os.Stat(path)
		if err != nil {
			errors <- err
			return
		}

		lastModified := info.ModTime()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				info, err := os.Stat(path)
				if err != nil {
					select {
					case errors <- err:
					case <-ctx.Done():
						return
					}

					continue
				}

				modified := info.ModTime()
				if modified.Equal(lastModified) {
					continue
				}

				lastModified = modified
				select {
				case changes <- struct{}{}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return changes, errors
}
