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
	streamerColumn       = 0
	statusColumn         = 1
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

	view := buildStreamsView(states)
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
	states []*client.Stream,
) tview.Primitive {
	activePlatform := 0
	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(false, false)

	table.SetBorder(true)
	platformMenu := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	platformMenu.SetBorder(true).
		SetTitle(" Platforms ").
		SetTitleAlign(tview.AlignCenter)

	update := func() {
		platform := platforms[activePlatform]
		updateTable(
			table,
			filterStreams(states, platform),
			platform,
		)

		updatePlatformMenu(
			platformMenu,
			activePlatform,
		)
	}

	update()
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

		url := state.URL
		streamerCell := tview.NewTableCell(
			fmt.Sprintf(
				"[::u:%s]%s[-:-:-:-]",
				url,
				tview.Escape(state.Channel),
			),
		).SetAlign(tview.AlignCenter)

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
				SetTextColor(statusColor).
				SetAttributes(tcell.AttrBold),
		)

		row++
	}
}

func updatePlatformMenu(
	menu *tview.TextView,
	activePlatform int,
) {
	menu.Clear()
	for index, platform := range platforms {
		if index == activePlatform {
			fmt.Fprintf(
				menu,
				"\n  [yellow::b]> %s[-:-:-]",
				platform,
			)

			continue
		}

		fmt.Fprintf(
			menu,
			"\n    %s",
			platform,
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
	case strings.Contains(
		url,
		"twitch.tv/",
	):
		return "Twitch"

	case strings.Contains(
		url,
		"kick.com/",
	):
		return "Kick"

	case strings.Contains(
		url,
		"youtube.com/",
	):
		return "YouTube"

	case strings.Contains(
		url,
		"youtu.be/",
	):
		return "YouTube"

	case strings.Contains(
		url,
		"w.tv/",
	):
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
