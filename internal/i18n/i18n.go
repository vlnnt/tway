package i18n

const (
	English = "en"
	Russian = "ru"
)

type Texts struct {
	Language string

	SetupTitle             string
	CheckInterval          string
	Summary                string
	SummaryInterval        string
	ShowStreamersAfterSave string
	Platforms              string
	Platform               string
	Save                   string
	Cancel                 string

	Enable        string
	Channels      string
	Channel       string
	Add           string
	AddChannel    string
	RemoveChannel string
	Back          string
	OK            string

	NoChannelsAdded string

	HTTPProxy  string
	SocksProxy string
	ClearProxy string
	AddProxy   string
	EditProxy  string

	PlatformChannels   string
	AddPlatformChannel string
	PlatformProxy      string

	IntervalCannotBeEmpty         string
	InvalidInterval               string
	IntervalMustBeGreaterThanZero string
	IntervalMustBeAtLeast         string

	Streams                  string
	Streamer                 string
	Status                   string
	LastStream               string
	LiveFor                  string
	Live                     string
	Offline                  string
	Loading                  string
	NoPlatforms              string
	EnablePlatformInSettings string
	PlatformStreams          string
	StreamsStatusBar         string
	Error                    string
	FailedToLoadStreams      string
	MinuteShort              string
	HourShort                string

	LogsTitle string

	Interval string
	Pause    string
	Resume   string
	Clear    string
	Close    string

	Auto string
	On   string
	Off  string

	ClearAllLogs               string
	FailedToReadLogFile        string
	FailedToClearLogs          string
	IntervalSeconds            string
	AutoRefresh                string
	EnterNumberGreaterThanZero string

	Refresh  string
	Logs     string
	Settings string
	Exit     string
	Show     string
	OpenLogs string

	ShowStreamersStatus  string
	ShowStreamsSummary   string
	RefreshStreamsStatus string
	OpenTwaySettings     string
	OpenTwayLogFile      string
	ExitApplication      string

	StreamStartedTitle   string
	StreamStartedMessage string
	StreamEndedTitle     string
	StreamEndedMessage   string

	StreamManualRefreshAlreadyRunningMessage string
	StreamManualRefreshRequestMessage        string
	StreamManualRefreshStatusReadyMessage    string

	StreamOnlineMessage  string
	StreamOfflineMessage string

	StreamMonitoringClosedMessage string

	StreamSingleInstanceMessage string

	StreamInitializationMessage  string
	StreamMonitoringReadyMessage string
}

func Get(
	language string,
) Texts {
	switch language {
	case Russian:
		return russian

	default:
		return english
	}
}
