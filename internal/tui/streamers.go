package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"tway/internal/client"
	"tway/internal/i18n"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	loadingViewWidth      = 50
	loadingViewHeight     = 5
	errorViewWidth        = 70
	errorViewHeight       = 7
	platformMenuWidth     = 18
	statusMenuWidth       = 16
	statusBarHeight       = 1
	loadingFrameInterval  = 80 * time.Millisecond
	streamRefreshInterval = 5 * time.Second
	tableHeaderRow        = 0
	tableFirstDataRow     = 1
	platformMenuFirstRow  = 1
	streamerColumn        = 0
	statusColumn          = 1
	lastStreamColumn      = 2
	liveForColumn         = 3
	columnExpansion       = 1
)

type Loader func() ([]*client.Stream, error)

type TUI struct {
	application *tview.Application
}

var moscowLocation = time.FixedZone(
	"MSK",
	3*60*60,
)

func NewTUI() *TUI {
	return &TUI{
		application: tview.NewApplication(),
	}
}

func (u *TUI) ShowStreamers(
	load Loader,
	platforms []string,
	texts i18n.Texts,
) error {
	loading := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	loading.SetBorder(true).
		SetTitle(
			fmt.Sprintf(
				" %s ",
				texts.Streams,
			),
		).
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(
			loading,
			loadingViewWidth,
			loadingViewHeight,
		),
		true,
	)

	go u.loadStreams(
		loading,
		load,
		platforms,
		texts,
	)

	return u.application.
		EnableMouse(true).
		Run()
}

func (u *TUI) loadStreams(
	loading *tview.TextView,
	load Loader,
	platforms []string,
	texts i18n.Texts,
) {
	frames := []string{
		"|",
		"/",
		"-",
		"\\",
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(loadingFrameInterval)
		defer ticker.Stop()
		frame := 0

		for {
			select {
			case <-done:
				return

			case <-ticker.C:
				currentFrame := frames[frame%len(frames)]
				frame++
				u.application.QueueUpdateDraw(
					func() {
						loading.SetText(
							fmt.Sprintf(
								"\n%s %s",
								texts.Loading,
								currentFrame,
							),
						)
					},
				)
			}
		}
	}()

	states, err := load()
	close(done)

	if err != nil {
		u.application.QueueUpdateDraw(
			func() {
				errorView := buildErrorView(err, texts)
				u.application.SetRoot(
					centerPrimitive(
						errorView,
						errorViewWidth,
						errorViewHeight,
					),
					true,
				)
			},
		)

		return
	}

	view := buildStreamsView(
		u.application,
		states,
		load,
		platforms,
		texts,
	)

	u.application.QueueUpdateDraw(
		func() {
			u.application.SetRoot(
				view,
				true,
			)
		},
	)
}

func buildStreamsView(
	application *tview.Application,
	states []*client.Stream,
	load Loader,
	platforms []string,
	texts i18n.Texts,
) tview.Primitive {
	activePlatform := 0
	if len(platforms) == 0 {
		empty := tview.NewTextView()
		empty.SetTextAlign(tview.AlignCenter)
		empty.SetText(
			fmt.Sprintf(
				"\n%s\n\n%s",
				texts.NoPlatforms,
				texts.EnablePlatformInSettings,
			),
		)

		empty.SetBorder(true).
			SetTitle(
				fmt.Sprintf(
					" %s ",
					texts.Streams,
				),
			).
			SetTitleAlign(tview.AlignCenter)

		empty.SetInputCapture(
			func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Key() {
				case tcell.KeyEscape:
					application.Stop()
					return nil
				}

				switch event.Rune() {
				case 'q', 'Q', 'й', 'Й':
					application.Stop()
					return nil
				}

				return event
			},
		)

		return empty
	}

	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(false, false)

	table.SetBorder(true)
	platformMenu := tview.NewTable().
		SetSelectable(false, false)

	platformMenu.SetBorder(true).
		SetTitle(
			fmt.Sprintf(
				" %s ",
				texts.Platforms,
			),
		).
		SetTitleAlign(tview.AlignCenter)

	statusBar := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	var update func()
	update = func() {
		platform := platforms[activePlatform]
		filteredStates := filterStreams(states, platform)

		updateTable(
			table,
			filteredStates,
			platform,
			texts,
		)

		updatePlatformMenu(
			platformMenu,
			platforms,
			activePlatform,
			func(index int) {
				activePlatform = index
				update()
			},
		)

		updateStatusBar(
			statusBar,
			filteredStates,
			texts,
		)
	}

	update()
	go func() {
		ticker := time.NewTicker(streamRefreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			newStates, err := load()
			if err != nil {
				continue
			}

			application.QueueUpdateDraw(
				func() {
					states = newStates
					update()
				},
			)
		}
	}()

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(platformMenu, platformMenuWidth, 0, false).
		AddItem(table, 0, 1, false)

	statusLayout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, statusMenuWidth, 0, false).
		AddItem(statusBar, 0, 1, false)

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mainLayout, 0, 1, false).
		AddItem(statusLayout, statusBarHeight, 0, false)

	layout.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyTAB:
				activePlatform++
				if activePlatform >= len(platforms) {
					activePlatform = 0
				}

				update()
				return nil

			case tcell.KeyBacktab:
				activePlatform--
				if activePlatform < 0 {
					activePlatform = len(platforms) - 1
				}

				update()
				return nil

			case tcell.KeyEscape:
				application.Stop()
				return nil

			case tcell.KeyRune:
				switch event.Rune() {
				case 'q', 'Q':
					application.Stop()
					return nil
				}
			}

			return event
		},
	)

	return layout
}

func updateTable(
	table *tview.Table,
	states []*client.Stream,
	platform string,
	texts i18n.Texts,
) {
	table.Clear()
	table.SetTitle(
		fmt.Sprintf(
			" %s ",
			fmt.Sprintf(
				texts.PlatformStreams,
				platform,
			),
		),
	)

	table.SetCell(
		tableHeaderRow,
		streamerColumn,
		tview.NewTableCell(texts.Streamer).
			SetAlign(tview.AlignCenter).
			SetExpansion(columnExpansion).
			SetAttributes(tcell.AttrBold),
	)

	table.SetCell(
		tableHeaderRow,
		statusColumn,
		tview.NewTableCell(texts.Status).
			SetAlign(tview.AlignCenter).
			SetExpansion(columnExpansion).
			SetAttributes(tcell.AttrBold),
	)

	table.SetCell(
		tableHeaderRow,
		lastStreamColumn,
		tview.NewTableCell(texts.LastStream).
			SetAlign(tview.AlignCenter).
			SetExpansion(columnExpansion).
			SetAttributes(tcell.AttrBold),
	)

	table.SetCell(
		tableHeaderRow,
		liveForColumn,
		tview.NewTableCell(texts.LiveFor).
			SetAlign(tview.AlignCenter).
			SetExpansion(columnExpansion).
			SetAttributes(tcell.AttrBold),
	)

	row := tableFirstDataRow
	for _, state := range states {
		if state == nil {
			continue
		}

		status := texts.Offline
		statusColor := tcell.ColorRed

		if state.IsLive {
			status = texts.Live
			statusColor = tcell.ColorGreen
		}

		lastStreamAt := "-"
		if !state.LastStreamAt.IsZero() {
			lastStreamAt = state.LastStreamAt.
				In(moscowLocation).
				Format("2006-01-02 15:04:05")
		}

		liveFor := "-"
		if state.IsLive && !state.StartedAt.IsZero() {
			liveFor = formatLiveFor(state.StartedAt, texts)
		}

		url := state.URL
		streamerCell :=
			tview.NewTableCell(
				fmt.Sprintf(
					"[::u:%s]%s[-:-:-:-]",
					url,
					tview.Escape(state.Channel),
				),
			).
				SetAlign(tview.AlignCenter).
				SetExpansion(columnExpansion)

		streamerCell.SetClickedFunc(
			func() bool {
				openURL(url)
				return true
			},
		)

		table.SetCell(
			row,
			streamerColumn,
			streamerCell,
		)

		table.SetCell(
			row,
			statusColumn,
			tview.NewTableCell(status).
				SetAlign(tview.AlignCenter).
				SetExpansion(columnExpansion).
				SetTextColor(statusColor).
				SetAttributes(tcell.AttrBold),
		)

		table.SetCell(
			row,
			lastStreamColumn,
			tview.NewTableCell(lastStreamAt).
				SetAlign(tview.AlignCenter).
				SetExpansion(columnExpansion),
		)

		table.SetCell(
			row,
			liveForColumn,
			tview.NewTableCell(liveFor).
				SetAlign(tview.AlignCenter).
				SetExpansion(columnExpansion),
		)

		row++
	}
}

func updateStatusBar(
	statusBar *tview.TextView,
	states []*client.Stream,
	texts i18n.Texts,
) {
	liveCount := 0
	for _, state := range states {
		if state != nil && state.IsLive {
			liveCount++
		}
	}

	statusBar.SetText(
		fmt.Sprintf(
			texts.StreamsStatusBar,
			liveCount,
			len(states),
		),
	)
}

func formatLiveFor(
	startedAt time.Time,
	texts i18n.Texts,
) string {
	duration := time.Since(startedAt)
	if duration < 0 {
		return "-"
	}

	totalMinutes := int(duration / time.Minute)
	if totalMinutes < 60 {
		return fmt.Sprintf(
			"%d%s",
			totalMinutes,
			texts.MinuteShort,
		)
	}

	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	return fmt.Sprintf(
		"%d%s %d%s",
		hours,
		texts.HourShort,
		minutes,
		texts.MinuteShort,
	)
}

func updatePlatformMenu(
	menu *tview.Table,
	platforms []string,
	activePlatform int,
	onSelect func(int),
) {
	menu.Clear()
	for index, platform := range platforms {
		platformIndex := index
		text := fmt.Sprintf(
			"    %s",
			platform,
		)

		cell := tview.NewTableCell(text).
			SetAlign(tview.AlignLeft).
			SetExpansion(1)

		if index == activePlatform {
			cell.SetText(
				fmt.Sprintf(
					"  > %s",
					platform,
				)).
				SetTextColor(tcell.ColorYellow).
				SetAttributes(tcell.AttrBold)
		}

		cell.SetClickedFunc(
			func() bool {
				onSelect(platformIndex)
				return true
			},
		)

		menu.SetCell(
			platformMenuFirstRow+index,
			0,
			cell,
		)
	}
}

func filterStreams(
	states []*client.Stream,
	platform string,
) []*client.Stream {
	filtered := make(
		[]*client.Stream,
		0,
		len(states),
	)

	for _, state := range states {
		if state == nil {
			continue
		}

		if streamPlatform(state) != platform {
			continue
		}

		filtered = append(
			filtered,
			state,
		)
	}

	sort.SliceStable(
		filtered,
		func(i, j int) bool {
			if filtered[i].IsLive == filtered[j].IsLive {
				return false
			}
			return filtered[i].IsLive
		},
	)

	return filtered
}

func streamPlatform(
	stream *client.Stream,
) string {
	url := strings.ToLower(stream.URL)
	switch {
	case strings.Contains(url, "twitch.tv/"):
		return "Twitch"

	case strings.Contains(url, "kick.com/"):
		return "Kick"

	case strings.Contains(url, "youtube.com/"):
		return "YouTube"

	case strings.Contains(url, "youtu.be/"):
		return "YouTube"

	case strings.Contains(url, "w.tv/"):
		return "W.TV"

	default:
		return ""
	}
}

func centerPrimitive(
	primitive tview.Primitive,
	width, height int,
) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(primitive, height, 1, true).
				AddItem(nil, 0, 1, false),
			width,
			1,
			true,
		).
		AddItem(nil, 0, 1, false)
}

func buildErrorView(
	err error,
	texts i18n.Texts,
) *tview.TextView {
	view := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	view.SetBorder(true).
		SetTitle(
			fmt.Sprintf(
				" %s ",
				texts.Error,
			),
		).
		SetTitleAlign(tview.AlignCenter)

	view.SetText(
		fmt.Sprintf(
			"\n[red]%s[-]\n\n%s",
			texts.FailedToLoadStreams,
			err.Error(),
		),
	)

	return view
}

func openURL(
	url string,
) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command(
			"rundll32",
			"url.dll,FileProtocolHandler",
			url,
		).Start()

	case "linux":
		_ = exec.Command(
			"xdg-open",
			url,
		).Start()
	}
}
