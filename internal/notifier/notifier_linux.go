//go:build linux

package notifier

import (
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"go.uber.org/zap"
)

type LinuxNotifier struct {
	log  *zap.Logger
	conn *dbus.Conn

	mu   sync.RWMutex
	urls map[uint32]string
}

func New(
	log *zap.Logger,
) (Notifier, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Error(
			"LinuxNotifier.New.ConnectSessionBus", zap.Error(err))
		return nil, err
	}

	n := &LinuxNotifier{
		log:  log,
		conn: conn,
		urls: make(map[uint32]string),
	}

	if err := n.setupActionHandler(); err != nil {
		conn.Close()
		log.Error(
			"LinuxNotifier.New.setupActionHandler", zap.Error(err))
		return nil, err
	}

	return n, nil
}

func (n *LinuxNotifier) Send(
	notification Notification,
) error {
	ctx, cancel := context.WithTimeout(
		context.Background(), 3*time.Second)
	defer cancel()

	notifications := n.conn.Object(
		"org.freedesktop.Notifications",
		dbus.ObjectPath("/org/freedesktop/Notifications"),
	)

	actions := []string{}
	if notification.URL != "" {
		actions = []string{
			"open",
			"Open Twitch",
		}
	}

	call := notifications.CallWithContext(
		ctx,
		"org.freedesktop.Notifications.Notify",
		dbus.Flags(0),
		"tway",
		uint32(0),
		notification.Icon,
		notification.Title,
		notification.Message,
		actions,
		map[string]dbus.Variant{
			"urgency": dbus.MakeVariant(byte(1)),
		},
		int32(5000),
	)

	if call.Err != nil {
		n.log.Error("LinuxNotifier.Send.CallWithContext", zap.Error(call.Err))
		return call.Err
	}

	if notification.URL != "" && len(call.Body) > 0 {
		notificationID, ok := call.Body[0].(uint32)
		if ok {
			n.mu.Lock()
			n.urls[notificationID] = notification.URL
			n.mu.Unlock()
		}
	}

	return nil
}

func (n *LinuxNotifier) setupActionHandler() error {
	if err := n.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.Notifications"),
		dbus.WithMatchMember("ActionInvoked"),
	); err != nil {
		return err
	}

	signals := make(chan *dbus.Signal, 10)
	n.conn.Signal(signals)

	go func() {

		for signal := range signals {

			if len(signal.Body) < 2 {
				continue
			}

			notificationID, ok := signal.Body[0].(uint32)
			if !ok {
				continue
			}

			action, ok := signal.Body[1].(string)
			if !ok {
				continue
			}

			if action != "open" {
				continue
			}

			n.mu.RLock()
			url := n.urls[notificationID]
			n.mu.RUnlock()

			if url == "" {
				n.log.Warn(
					"LinuxNotifier.ActionInvoked.URLNotFound",
					zap.Uint32("NotificationID", notificationID),
				)
				continue
			}

			n.log.Info(
				"LinuxNotifier.ActionInvoked",
				zap.Uint32("NotificationID", notificationID),
				zap.String("URL", url),
			)

			if err := exec.Command("xdg-open", url).Start(); err != nil {
				n.log.Error(
					"LinuxNotifier.ActionInvoked.OpenURL",
					zap.String("URL", url),
					zap.Error(err),
				)
			}
		}
	}()

	return nil
}

func (n *LinuxNotifier) Close() error {
	if n.conn == nil {
		return nil
	}

	return n.conn.Close()
}
