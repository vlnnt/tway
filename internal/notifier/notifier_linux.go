//go:build linux

package notifier

import (
	"context"
	"time"

	"github.com/godbus/dbus/v5"
	"go.uber.org/zap"
)

type LinuxNotifier struct {
	log  *zap.Logger
	conn *dbus.Conn
}

func New(
	log *zap.Logger,
) (Notifier, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Error("LinuxNotifier.New.ConnectSessionBus", zap.Error(err))
		return nil, err
	}

	return &LinuxNotifier{
		conn: conn,
		log:  log,
	}, nil
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
		"dialog-information",
		notification.Title,
		notification.Message,
		[]string{},
		map[string]dbus.Variant{
			"urgency": dbus.MakeVariant(byte(1)),
		},
		int32(5000),
	)

	if call.Err != nil {
		n.log.Error("LinuxNotifier.Send.CallWithContext", zap.Error(call.Err))
		return call.Err
	}

	return nil
}

func (n *LinuxNotifier) Close() error {
	if n.conn == nil {
		return nil
	}

	return n.conn.Close()
}
