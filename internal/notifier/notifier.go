package notifier

type Notification struct {
	Title   string
	Message string
	Icon    string
	URL     string
}

type Notifier interface {
	Send(Notification) error
	Close() error
}
