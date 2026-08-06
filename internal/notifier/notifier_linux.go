//go:build linux

package notifier

import (
	"context"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
)

type LinuxNotifier struct {
	conn *dbus.Conn
}

func New() (Notifier, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to session D-Bus: %w", err)
	}

	return &LinuxNotifier{conn: conn}, nil
}

func (n *LinuxNotifier) Send(
	notification Notification,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	notifications := n.conn.Object(
		"org.freedesktop.Notifications",
		dbus.ObjectPath("/org/freedesktop/Notifications"),
	)

	call := notifications.CallWithContext(
		ctx,
		"org.freedesktop.Notifications.Notify",
		dbus.Flags(0),
		"tway",
		uint32(0),
		"",
		notification.Title,
		notification.Message,
		[]string{},
		map[string]dbus.Variant{
			"urgency": dbus.MakeVariant(byte(1)),
		},
		int32(5000),
	)

	if call.Err != nil {
		return fmt.Errorf("send notification: %w", call.Err)
	}

	return nil
}

func (n *LinuxNotifier) Close() error {
	if n.conn == nil {
		return nil
	}

	return n.conn.Close()
}
