package youtube

type innertubeContext struct {
	Client innertubeClient `json:"client"`
}

type innertubeClient struct {
	ClientName    string `json:"clientName"`
	ClientVersion string `json:"clientVersion"`
	HL            string `json:"hl,omitempty"`
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
