// Package notify shows the app's system notifications.
package notify

import (
	"errors"
	"log/slog"

	"github.com/egoist/mygo"
)

// Notifier shows and closes notifications, once each. It is used on the main
// thread.
type Notifier struct {
	log     *slog.Logger
	current *mygo.Notification
	known   bool
	warned  bool
}

// New returns a notifier for the app.
func New(log *slog.Logger) *Notifier {
	return &Notifier{log: log, known: mygo.NotificationsSupported()}
}

// Show displays a notification, replacing the last one. IDs are fixed
// ("break-mini", "break-long") so that a repeat replaces the old notice.
func (n *Notifier) Show(id, title, body string, silent bool) {
	if !n.known {
		if !n.warned {
			n.warned = true
			n.log.Warn("notifications are not available; skipping notices")
		}
		return
	}
	note := mygo.NewNotification(mygo.NotificationOptions{
		ID:     id,
		Title:  title,
		Body:   body,
		Silent: silent,
	})
	if err := note.Show(); err != nil {
		if errors.Is(err, mygo.ErrNotificationsDenied) {
			if !n.warned {
				n.warned = true
				n.log.Warn("notifications are not allowed")
			}
			return
		}
		n.log.Warn("show notification", "error", err)
		return
	}
	n.current = note
}

// Close takes down the notification still shown, if any.
func (n *Notifier) Close() {
	if n.current == nil {
		return
	}
	n.current.Close()
	n.current = nil
}
