package i18n

var english = Texts{
	Language: "Language",

	SetupTitle:             "Tway Configuration",
	CheckInterval:          "Check interval",
	Summary:                "Summary",
	SummaryInterval:        "Summary interval",
	ShowStreamersAfterSave: "Show streamers window after save",
	Platforms:              "Platforms",
	Platform:               "Platform",
	Save:                   "Save",
	Cancel:                 "Cancel",

	Enable:        "Enable",
	Channels:      "Channels",
	Channel:       "Channel",
	Add:           "Add",
	AddChannel:    "Add Channel",
	RemoveChannel: "Remove Channel",
	Back:          "Back",
	OK:            "OK",

	NoChannelsAdded: "No channels added",

	HTTPProxy:  "HTTP Proxy",
	SocksProxy: "SOCKS Proxy",
	ClearProxy: "Clear Proxy",
	AddProxy:   "Add Proxy",
	EditProxy:  "Edit Proxy",

	PlatformChannels:   "%s Channels",
	AddPlatformChannel: "Add %s Channel",
	PlatformProxy:      "%s Proxy",

	IntervalCannotBeEmpty:         "Interval cannot be empty",
	InvalidInterval:               "Invalid interval: use values like 30s, 2m or 1h",
	IntervalMustBeGreaterThanZero: "Interval must be greater than 0",
	IntervalMustBeAtLeast:         "Interval must be at least %s",

	Streams:                  "Streams",
	Streamer:                 "Streamer",
	Status:                   "Status",
	LastStream:               "Last Stream",
	LiveFor:                  "Live For",
	Live:                     "LIVE",
	Offline:                  "OFFLINE",
	Loading:                  "Loading streams status",
	NoPlatforms:              "No platforms enabled.",
	EnablePlatformInSettings: "Enable a platform in settings.",
	PlatformStreams:          "%s Streams",
	StreamsStatusBar:         "Live: %d / %d | Tab/Shift+Tab: platform | Q/Esc: quit",
	Error:                    "Error",
	FailedToLoadStreams:      "Failed to load streams status",
	MinuteShort:              "m",
	HourShort:                "h",

	LogsTitle: "Tway Logs",

	Interval: "Interval",
	Pause:    "Pause",
	Resume:   "Resume",
	Clear:    "Clear",
	Close:    "Close",

	Auto: "Auto",
	On:   "ON",
	Off:  "OFF",

	ClearAllLogs:               "Clear all logs?",
	FailedToReadLogFile:        "Failed to read log file",
	FailedToClearLogs:          "Failed to clear logs",
	IntervalSeconds:            "Interval (seconds): ",
	AutoRefresh:                "Auto Refresh",
	EnterNumberGreaterThanZero: "Enter a number greater than 0",

	Refresh:  "Refresh",
	Logs:     "Logs",
	Settings: "Settings",
	Exit:     "Exit",
	Show:     "Show",
	OpenLogs: "Open Logs",

	ShowStreamersStatus:  "Show streamers status",
	ShowStreamsSummary:   "Show streams summary",
	RefreshStreamsStatus: "Refresh streams status",
	OpenTwaySettings:     "Open Tway settings",
	OpenTwayLogFile:      "Open Tway log file",
	ExitApplication:      "Exit application",

	StreamStartedTitle:   "%s is now live!",
	StreamStartedMessage: "%s\nCategory: %s",
	StreamEndedTitle:     "%s is no longer live!",
	StreamEndedMessage:   "The streamer has left the broadcast!",

	StreamManualRefreshAlreadyRunningMessage: "Manual stream refresh is already running!",
	StreamManualRefreshRequestMessage:        "Processing streams status refresh started!",
	StreamManualRefreshStatusReadyMessage:    "Streams status refreshed!",

	StreamOnlineMessage:  "Online",
	StreamOfflineMessage: "Offline",

	StreamMonitoringClosedMessage: "Stream monitoring stopped due to a fatal error. Tway will be closed.",

	StreamSingleInstanceMessage: "Tway is already running. Only one instance can run at a time.",

	StreamInitializationMessage:  "Initializing services and connecting to streaming platforms...",
	StreamMonitoringReadyMessage: "Initialization completed. All services are connected and stream monitoring is active.",
}
