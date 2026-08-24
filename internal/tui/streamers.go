package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"tway/internal/client"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	loadingViewWidth     = 50
	loadingViewHeight    = 5
	errorViewWidth       = 70
	errorViewHeight      = 7
	platformMenuWidth    = 18
	loadingFrameInterval = 80 * time.Millisecond
	tableHeaderRow       = 0
	tableFirstDataRow    = 1
	platformMenuFirstRow = 1
	streamerColumn       = 0
	statusColumn         = 1
	lastStreamColumn     = 2
	liveForColumn        = 3
	columnExpansion      = 1
)

type Loader func() ([]*client.Stream, error)

type TUI struct {
	application *tview.Application
}

var platforms = []string{
	"Twitch",
	"Kick",
	"YouTube",
	"W.TV",
}

func NewTUI() *TUI {
	return &TUI{
		application: tview.NewApplication(),
	}
}

func (u *TUI) ShowStreamers(
	load Loader,
) error {
	loading := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	loading.SetBorder(true).
		SetTitle(" Streams ").
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(
			loading,
			loadingViewWidth,
			loadingViewHeight,
		),
		true,
	)

	go u.loadStreams(loading, load)
	return u.application.
		EnableMouse(true).
		Run()
}

func (u *TUI) loadStreams(
	loading *tview.TextView,
	load Loader,
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
								"\nLoading streams status %s",
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
				errorView := buildErrorView(err)
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

	view := buildStreamsView(u.application, states)
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
) tview.Primitive {
	activePlatform := 0
	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(false, false)

	table.SetBorder(true)
	platformMenu := tview.NewTable().
		SetSelectable(false, false)

	platformMenu.SetBorder(true).
		SetTitle(" Platforms ").
		SetTitleAlign(tview.AlignCenter)

	var update func()
	update = func() {
		platform := platforms[activePlatform]
		updateTable(
			table,
			filterStreams(states, platform),
			platform,
		)

		updatePlatformMenu(
			platformMenu,
			activePlatform,
			func(index int) {
				activePlatform = index
				update()
			},
		)
	}

	update()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			application.QueueUpdateDraw(
				func() {
					update()
				},
			)
		}
	}()

	layout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(platformMenu, platformMenuWidth, 0, false).
		AddItem(table, 0, 1, false)

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
) {
	table.Clear()
	table.SetTitle(
		fmt.Sprintf(
			" %s Streams ",
			platform,
		),
	)

	table.SetCell(
		tableHeaderRow,
		streamerColumn,
		tview.NewTableCell("Streamer").
			SetAlign(tview.AlignCenter).
			SetExpansion(columnExpansion).
			SetAttributes(tcell.AttrBold),
	)

	table.SetCell(
		tableHeaderRow,
		statusColumn,
		tview.NewTableCell("Status").
			SetAlign(tview.AlignCenter).
			SetExpansion(columnExpansion).
			SetAttributes(tcell.AttrBold),
	)

	table.SetCell(
		tableHeaderRow,
		lastStreamColumn,
		tview.NewTableCell("Last Stream").
			SetAlign(tview.AlignCenter).
			SetExpansion(columnExpansion).
			SetAttributes(tcell.AttrBold),
	)

	table.SetCell(
		tableHeaderRow,
		liveForColumn,
		tview.NewTableCell("Live For").
			SetAlign(tview.AlignCenter).
			SetExpansion(columnExpansion).
			SetAttributes(tcell.AttrBold),
	)

	row := tableFirstDataRow
	for _, state := range states {
		if state == nil {
			continue
		}

		status := "OFFLINE"
		statusColor := tcell.ColorRed

		if state.IsLive {
			status = "LIVE"
			statusColor = tcell.ColorGreen
		}

		lastStreamAt := "-"
		if !state.LastStreamAt.IsZero() {
			lastStreamAt = state.LastStreamAt.Format("2006-01-02 15:04:05")
		}

		liveFor := "-"
		if state.IsLive && !state.LastStreamAt.IsZero() {
			liveFor = formatLiveFor(state.LastStreamAt)
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

func formatLiveFor(
	startedAt time.Time,
) string {
	duration := time.Since(startedAt)
	if duration < 0 {
		return "-"
	}

	totalMinutes := int(duration / time.Minute)
	if totalMinutes < 60 {
		return fmt.Sprintf(
			"%dm",
			totalMinutes,
		)
	}

	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	return fmt.Sprintf(
		"%dh %dm",
		hours,
		minutes,
	)
}

func updatePlatformMenu(
	menu *tview.Table,
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
) *tview.TextView {
	view := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	view.SetBorder(true).
		SetTitle(" Error ").
		SetTitleAlign(tview.AlignCenter)

	view.SetText(
		fmt.Sprintf(
			"\n[red]Failed to load streams status[-]\n\n%s",
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
