package tui

import (
	"fmt"
	"strings"
	"tway/internal/config"

	"github.com/rivo/tview"
)

const (
	setupViewWidth          = 70
	setupViewHeight         = 16
	platformSetupViewHeight = 18
	channelsSetupViewHeight = 12
	proxySetupViewHeight    = 14
	addChannelViewHeight    = 10
)

type Setup struct {
	Name     string
	Settings *config.Platform
}

func (u *TUI) ShowSetup(
	config *config.Config,
) (bool, error) {
	saved := false

	var showMainSetup func()
	showMainSetup = func() {
		form := tview.NewForm()
		form.AddInputField(
			"Check interval",
			config.Check,
			20,
			nil,
			func(value string) {
				config.Check = strings.TrimSpace(value)
			},
		)

		addCheckbox(
			form,
			"Summary",
			config.Summary.Enable,
			func(checked bool) {
				config.Summary.Enable = checked
			},
		)

		form.AddInputField(
			"Summary interval",
			config.Summary.Interval,
			20,
			nil,
			func(value string) {
				config.Summary.Interval = strings.TrimSpace(value)
			},
		)

		addCheckbox(
			form,
			"Show streamers after save",
			config.UI.ShowStreamers,
			func(checked bool) {
				config.UI.ShowStreamers = checked
			},
		)

		form.AddButton(
			"Platforms",
			func() {
				u.showPlatformSetup(
					config,
					&saved,
					showMainSetup,
					0,
				)
			},
		)

		form.AddButton(
			"Save",
			func() {
				saved = true
				u.application.Stop()
			},
		)

		form.AddButton(
			"Cancel",
			func() {
				u.application.Stop()
			},
		)

		form.SetBorder(true).
			SetTitle(" Tway Configuration ").
			SetTitleAlign(tview.AlignCenter)

		u.application.SetRoot(
			centerPrimitive(
				form,
				setupViewWidth,
				setupViewHeight,
			),
			true,
		)
	}

	showMainSetup()
	if err := u.application.EnableMouse(true).Run(); err != nil {
		return false, err
	}

	return saved, nil
}

func (u *TUI) showPlatformSetup(
	config *config.Config,
	saved *bool,
	showMainSetup func(),
	activePlatform int,
) {
	platforms := setupPlatforms(config)
	if activePlatform < 0 || activePlatform >= len(platforms) {
		activePlatform = 0
	}

	platform := platforms[activePlatform]
	form := tview.NewForm()

	names := make(
		[]string,
		0,
		len(platforms),
	)

	for _, platform := range platforms {
		names = append(names, platform.Name)
	}

	form.AddDropDown(
		"Platform",
		names,
		activePlatform,
		func(
			_ string, index int,
		) {
			if index == activePlatform {
				return
			}

			u.showPlatformSetup(
				config,
				saved,
				showMainSetup,
				index,
			)
		},
	)

	addCheckbox(
		form,
		"Enable",
		platform.Settings.Enable,
		func(checked bool) {
			platform.Settings.Enable = checked
		},
	)

	form.AddButton(
		fmt.Sprintf(
			"Channels (%d)",
			len(platform.Settings.Channels),
		),
		func() {
			currentPlatform := activePlatform
			u.showChannelsSetup(
				platform.Name,
				platform.Settings,
				func() {
					u.showPlatformSetup(
						config,
						saved,
						showMainSetup,
						currentPlatform,
					)
				},
			)
		},
	)

	form.AddButton(
		proxyButtonLabel(platform.Settings.Proxy),
		func() {
			currentPlatform := activePlatform
			u.showProxySetup(
				platform.Name,
				&platform.Settings.Proxy,
				func() {
					u.showPlatformSetup(
						config,
						saved,
						showMainSetup,
						currentPlatform,
					)
				},
			)
		},
	)

	form.AddButton(
		"Back",
		func() {
			showMainSetup()
		},
	)

	form.AddButton(
		"Save",
		func() {
			*saved = true
			u.application.Stop()
		},
	)

	form.SetBorder(true).
		SetTitle(" Platforms ").
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(
			form,
			setupViewWidth,
			platformSetupViewHeight,
		),
		true,
	)
}

func (u *TUI) showChannelsSetup(
	platformName string,
	platform *config.Platform,
	back func(),
) {
	u.showChannelsSetupAt(
		platformName,
		platform,
		back,
		0,
	)
}

func (u *TUI) showChannelsSetupAt(
	platformName string,
	platform *config.Platform,
	back func(),
	selectedChannel int,
) {
	form := tview.NewForm()
	if len(platform.Channels) > 0 {
		if selectedChannel < 0 {
			selectedChannel = 0
		}

		if selectedChannel >= len(platform.Channels) {
			selectedChannel = len(platform.Channels) - 1
		}

		form.AddDropDown(
			"Channel",
			platform.Channels,
			selectedChannel,
			func(
				_ string,
				index int,
			) {
				selectedChannel = index
			},
		)

	} else {
		form.AddTextView(
			"Channels",
			"No channels added",
			40,
			1,
			false,
			false,
		)
	}

	form.AddButton(
		"Add Channel",
		func() {
			u.showAddChannelSetup(
				platformName,
				platform,
				func() {
					selected := len(platform.Channels) - 1
					u.showChannelsSetupAt(
						platformName,
						platform,
						back,
						selected,
					)
				},
			)
		},
	)

	form.AddButton(
		"Remove Channel",
		func() {
			if len(platform.Channels) == 0 {
				return
			}

			platform.Channels = append(
				platform.Channels[:selectedChannel],
				platform.Channels[selectedChannel+1:]...,
			)

			if selectedChannel >= len(platform.Channels) {
				selectedChannel = len(platform.Channels) - 1
			}

			if selectedChannel < 0 {
				selectedChannel = 0
			}

			u.showChannelsSetupAt(
				platformName,
				platform,
				back,
				selectedChannel,
			)
		},
	)

	form.AddButton(
		"Back",
		back,
	)

	form.SetBorder(true).
		SetTitle(
			fmt.Sprintf(
				" %s Channels ",
				platformName,
			),
		).
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(
			form,
			setupViewWidth,
			channelsSetupViewHeight,
		),
		true,
	)
}

func (u *TUI) showAddChannelSetup(
	platformName string,
	platform *config.Platform,
	back func(),
) {
	form := tview.NewForm()
	channel := ""
	form.AddInputField(
		"Channel",
		"",
		50,
		nil,
		func(value string) {
			channel = strings.TrimSpace(value)
		},
	)

	form.AddButton(
		"Add",
		func() {
			if channel == "" {
				return
			}

			if channelExists(
				platform.Channels,
				channel,
			) {
				return
			}

			platform.Channels = append(
				platform.Channels,
				channel,
			)

			back()
		},
	)

	form.AddButton(
		"Cancel",
		back,
	)

	form.SetBorder(true).
		SetTitle(
			fmt.Sprintf(
				" Add %s Channel ",
				platformName,
			),
		).
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(
			form,
			setupViewWidth,
			addChannelViewHeight,
		),
		true,
	)
}

func (u *TUI) showProxySetup(
	platformName string,
	proxy *config.Proxy,
	back func(),
) {
	form := tview.NewForm()
	form.AddInputField(
		"HTTP Proxy",
		proxy.HTTP,
		50,
		nil,
		func(value string) {
			proxy.HTTP = strings.TrimSpace(value)
		},
	)

	form.AddInputField(
		"SOCKS Proxy",
		proxy.Socks,
		50,
		nil,
		func(value string) {
			proxy.Socks = strings.TrimSpace(value)
		},
	)

	form.AddButton(
		"Clear Proxy",
		func() {
			proxy.HTTP = ""
			proxy.Socks = ""
			u.showProxySetup(
				platformName,
				proxy,
				back,
			)
		},
	)

	form.AddButton(
		"Back",
		back,
	)

	form.SetBorder(true).
		SetTitle(
			fmt.Sprintf(
				" %s Proxy ",
				platformName,
			),
		).
		SetTitleAlign(tview.AlignCenter)

	u.application.SetRoot(
		centerPrimitive(
			form,
			setupViewWidth,
			proxySetupViewHeight,
		),
		true,
	)
}

func addCheckbox(
	form *tview.Form,
	label string,
	checked bool,
	changed func(bool),
) {
	checkbox := tview.NewCheckbox().
		SetLabel(label).
		SetChecked(checked).
		SetCheckedString(tview.Escape("[x]")).
		SetUncheckedString(tview.Escape("[ ]")).
		SetChangedFunc(changed)
	form.AddFormItem(checkbox)
}

func setupPlatforms(
	config *config.Config,
) []Setup {
	return []Setup{
		{
			Name:     "Twitch",
			Settings: &config.Twitch.Platform,
		},
		{
			Name:     "Kick",
			Settings: &config.Kick.Platform,
		},
		{
			Name:     "YouTube",
			Settings: &config.Youtube.Platform,
		},
		{
			Name:     "W.TV",
			Settings: &config.WTV.Platform,
		},
	}
}

func proxyButtonLabel(
	proxy config.Proxy,
) string {
	if strings.TrimSpace(proxy.HTTP) == "" &&
		strings.TrimSpace(proxy.Socks) == "" {
		return "Add Proxy"
	}

	return "Edit Proxy"
}

func channelExists(
	channels []string,
	channel string,
) bool {
	for _, existing := range channels {
		if strings.EqualFold(existing, channel) {
			return true
		}
	}

	return false
}
