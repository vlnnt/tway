//go:build windows

package notifier

import (
	"path/filepath"

	"github.com/go-toast/toast"
	"go.uber.org/zap"
)

type WindowsNotifier struct {
	log *zap.Logger
}

func New(
	log *zap.Logger,
) (Notifier, error) {
	return &WindowsNotifier{log: log}, nil
}

func (wn *WindowsNotifier) Send(
	notification Notification,
) error {
	icon := notification.Icon
	if icon != "" {
		absoluteIcon, err := filepath.Abs(icon)
		if err != nil {
			wn.log.Error("WindowsNotifier.Send.Abs", zap.Error(err))
			return err
		}

		icon = absoluteIcon
	}

	windowsNotification := toast.Notification{
		AppID:   "tway",
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
		wn.log.Error("WindowsNotifier.Send.Push", zap.Error(err))
		return err
	}

	return nil
}

func (n *WindowsNotifier) Close() error {
	return nil
}
