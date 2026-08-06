//go:build windows

package notifier

import (
	"fmt"
	"path/filepath"

	"github.com/go-toast/toast"
)

type WindowsNotifier struct{}

func New() (Notifier, error) {
	return &WindowsNotifier{}, nil
}

func (wn *WindowsNotifier) Send(
	notification Notification,
) error {
	icon := notification.Icon
	if icon != "" {
		absoluteIcon, err := filepath.Abs(icon)
		if err != nil {
			return fmt.Errorf("resolve notification icon path: %w", err)
		}

		icon = absoluteIcon
	}

	windowsNotification := toast.Notification{
		AppID:   "Twitch Watcher",
		Title:   notification.Title,
		Message: notification.Message,
		Icon:    icon,
		Audio:   toast.Default,
	}

	if notification.URL != "" {
		windowsNotification.ActivationType = "protocol"
		windowsNotification.ActivationArguments = notification.URL
	}

	if err := windowsNotification.Push(); err != nil {
		return fmt.Errorf("send Windows notification: %w", err)
	}

	return nil
}

func (n *WindowsNotifier) Close() error {
	return nil
}
