package youtube

const (
	maxAttempts             = 3
	maxLastStreamCandidates = 2

	baseUrl           = "https://www.youtube.com/@"
	baseWatchUrl      = "https://www.youtube.com/watch?v="
	channelUrl        = "https://www.youtube.com/channel/"
	resolveUrl        = "https://www.youtube.com/youtubei/v1/navigation/resolve_url?"
	playerUrl         = "https://www.youtube.com/youtubei/v1/player?"
	browseUrl         = "https://www.youtube.com/youtubei/v1/browse?"
	acceptHeader      = "application/json"
	contentTypeHeader = "application/json"
	userAgent         = "Mozilla/5.0"
	clientName        = "WEB"
	clientNameID      = "1"
	clientVersion     = "2.20260708.00.00"
	innertubeAPIKey   = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	streamsParams     = "EgdzdHJlYW1z8gYECgJ6AA%3D%3D"
)

type browseRequest struct {
	Context  innertubeContext `json:"context"`
	BrowseID string           `json:"browseId"`
	Params   string           `json:"params"`
}

type innertubeClient struct {
	ClientName    string `json:"clientName"`
	ClientVersion string `json:"clientVersion"`
	HL            string `json:"hl,omitempty"`
	GL            string `json:"gl,omitempty"`
}

type innertubeContext struct {
	Client innertubeClient `json:"client"`
}

type resolveRequest struct {
	Context innertubeContext `json:"context"`
	URL     string           `json:"url"`
}

type resolveResponse struct {
	Endpoint struct {
		WatchEndpoint struct {
			VideoID string `json:"videoId"`
		} `json:"watchEndpoint"`

		BrowseEndpoint struct {
			BrowseID string `json:"browseId"`
		} `json:"browseEndpoint"`
	} `json:"endpoint"`
}

type playerRequest struct {
	Context innertubeContext `json:"context"`
	VideoID string           `json:"videoId"`
}

type playerResponse struct {
	VideoDetails struct {
		VideoID string `json:"videoId"`
		Title   string `json:"title"`
		Author  string `json:"author"`
	} `json:"videoDetails"`

	Microformat struct {
		PlayerMicroformatRenderer struct {
			Category string `json:"category"`

			LiveBroadcastDetails struct {
				IsLiveNow      bool   `json:"isLiveNow"`
				StartTimestamp string `json:"startTimestamp"`
			} `json:"liveBroadcastDetails"`
		} `json:"playerMicroformatRenderer"`
	} `json:"microformat"`
}
