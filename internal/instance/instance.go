package instance

import "errors"

var ErrAlreadyRunning = errors.New("tway is already running")

type Lock struct {
	release func() error
}

func (l *Lock) Close() error {
	if l == nil || l.release == nil {
		return nil
	}

	release := l.release
	l.release = nil

	return release()
}
