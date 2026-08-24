package tray

import (
	_ "embed"
	"tway/internal/tui"

	"github.com/getlantern/systray"
	"go.uber.org/zap"
)

//go:embed tway.ico
var icon []byte

type Tray struct {
	log       *zap.Logger
	onRefresh func()
	onSummary func()
	onExit    func()
}

func NewTray(
	log *zap.Logger,
	onRefresh func(),
	onSummary func(),
	onExit func(),
) *Tray {
	return &Tray{
		log:       log,
		onRefresh: onRefresh,
		onSummary: onSummary,
		onExit:    onExit,
	}
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExitHandler)
}

func (t *Tray) showStreamers() {
	if err := tui.OpenTerminal(); err != nil {
		t.log.Error(
			"Tray.showStreamers.OpenTerminal",
			zap.Error(err),
		)
	}
}

func (t *Tray) showStreamsSummary() {
	if t.onSummary != nil {
		t.onSummary()
	}
}

func (t *Tray) refreshStreamsStatus() {
	if t.onRefresh != nil {
		t.onRefresh()
	}
}

func (t *Tray) onReady() {
	systray.SetTitle("tway")
	systray.SetTooltip("tway")
	systray.SetIcon(icon)

	show := systray.AddMenuItem(
		"Show",
		"Show streamers status",
	)

	summary := systray.AddMenuItem(
		"Summary",
		"Show streams summary",
	)

	refresh := systray.AddMenuItem(
		"Refresh",
		"Refresh streams status",
	)

	systray.AddSeparator()
	exit := systray.AddMenuItem(
		"Exit",
		"Exit application",
	)

	go func() {
		for range show.ClickedCh {
			t.log.Info("Tray.showStreamers",
				zap.String("Clicked", "Tray show streamers status requested"),
			)

			t.showStreamers()
		}
	}()

	go func() {
		for range summary.ClickedCh {
			t.log.Info("Tray.showSummary",
				zap.String("Clicked", "Tray show streams summary requested"))

			t.showStreamsSummary()
		}
	}()

	go func() {
		for range refresh.ClickedCh {
			t.log.Info("Tray.refreshStatus",
				zap.String("Clicked", "Tray refresh streams status"),
			)

			t.refreshStreamsStatus()
		}
	}()

	go func() {
		<-exit.ClickedCh
		t.log.Info("Tray.onReady",
			zap.String("Clicked", "Tray exit requested"))
		systray.Quit()
	}()
}

func (t *Tray) onExitHandler() {
	if t.onExit != nil {
		t.onExit()
	}
}
