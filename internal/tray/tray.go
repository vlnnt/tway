package tray

import (
	_ "embed"
	"sync"
	"tway/internal/i18n"
	"tway/internal/tui"

	"github.com/getlantern/systray"
	"go.uber.org/zap"
)

//go:embed tway.ico
var icon []byte

type Tray struct {
	log        *zap.Logger
	onRefresh  func()
	onSummary  func()
	onSettings func()
	onLogs     func()
	onExit     func()

	mu    sync.RWMutex
	texts i18n.Texts

	showItem     *systray.MenuItem
	summaryItem  *systray.MenuItem
	refreshItem  *systray.MenuItem
	settingsItem *systray.MenuItem
	logsItem     *systray.MenuItem
	exitItem     *systray.MenuItem
}

func NewTray(
	log *zap.Logger,
	texts i18n.Texts,
	onRefresh func(),
	onSummary func(),
	onSettings func(),
	onLogs func(),
	onExit func(),
) *Tray {
	return &Tray{
		log:        log,
		texts:      texts,
		onRefresh:  onRefresh,
		onSummary:  onSummary,
		onSettings: onSettings,
		onLogs:     onLogs,
		onExit:     onExit,
	}
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExitHandler)
}

func (t *Tray) Quit() {
	systray.Quit()
}

func (t *Tray) SetTexts(
	texts i18n.Texts,
) {
	t.mu.Lock()

	t.texts = texts

	show := t.showItem
	summary := t.summaryItem
	refresh := t.refreshItem
	settings := t.settingsItem
	logs := t.logsItem
	exit := t.exitItem

	t.mu.Unlock()

	if show != nil {
		show.SetTitle(texts.Show)
	}

	if summary != nil {
		summary.SetTitle(texts.Summary)
	}

	if refresh != nil {
		refresh.SetTitle(texts.Refresh)
	}

	if settings != nil {
		settings.SetTitle(texts.Settings)
	}

	if logs != nil {
		logs.SetTitle(texts.OpenLogs)
	}

	if exit != nil {
		exit.SetTitle(texts.Exit)
	}
}

func (t *Tray) showStreamers() {
	if err := tui.OpenTerminal("--tui"); err != nil {
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

func (t *Tray) showSettings() {
	if t.onSettings != nil {
		t.onSettings()
	}
}

func (t *Tray) showLogs() {
	if t.onLogs != nil {
		t.onLogs()
	}
}

func (t *Tray) onReady() {
	systray.SetTitle("tway")
	systray.SetTooltip("tway")
	systray.SetIcon(icon)

	t.mu.RLock()
	texts := t.texts
	t.mu.RUnlock()

	show := systray.AddMenuItem(
		texts.Show,
		texts.ShowStreamersStatus,
	)

	summary := systray.AddMenuItem(
		texts.Summary,
		texts.ShowStreamsSummary,
	)

	refresh := systray.AddMenuItem(
		texts.Refresh,
		texts.RefreshStreamsStatus,
	)

	settings := systray.AddMenuItem(
		texts.Settings,
		texts.OpenTwaySettings,
	)

	logs := systray.AddMenuItem(
		texts.OpenLogs,
		texts.OpenTwayLogFile,
	)

	systray.AddSeparator()
	exit := systray.AddMenuItem(
		texts.Exit,
		texts.ExitApplication,
	)

	t.mu.Lock()

	t.showItem = show
	t.summaryItem = summary
	t.refreshItem = refresh
	t.settingsItem = settings
	t.logsItem = logs
	t.exitItem = exit

	t.mu.Unlock()

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
		for range settings.ClickedCh {
			t.log.Info(
				"Tray.showSettings",
				zap.String(
					"Clicked",
					"Tray settings requested",
				),
			)

			t.showSettings()
		}
	}()

	go func() {
		for range logs.ClickedCh {
			t.log.Info(
				"Tray.showLogs",
				zap.String(
					"Clicked",
					"Tray open logs requested",
				),
			)

			t.showLogs()
		}
	}()

	go func() {
		<-exit.ClickedCh
		t.log.Info("Tray.onReady",
			zap.String("Clicked", "Tray exit requested"))

		t.Quit()
	}()
}

func (t *Tray) onExitHandler() {
	if t.onExit != nil {
		t.onExit()
	}
}
