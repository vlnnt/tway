package kick

const (
	maxAttempts            = 3
	kickBaseUrl            = "https://kick.com/"
	kickAcceptHeader       = "application/json, text/plain, */*"
	kickUserAgent          = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0 Safari/537.36"
	kickApiChannelsV1Route = "https://kick.com/api/v1/channels/"
	kickApiChannelsV2Route = "https://kick.com/api/v2/channels/"
)

type category struct {
	Name string `json:"name"`
}

type channelResponse struct {
	Slug       string      `json:"slug"`
	Livestream *livestream `json:"livestream"`
}

type livestream struct {
	CreatedAt string    `json:"created_at"`
	Title     string    `json:"session_title"`
	Category  *category `json:"category"`
}

type previousLivestreamsResponse struct {
	PreviousLivestreams []previousLivestream `json:"previous_livestreams"`
}

type previousLivestream struct {
	StartTime string `json:"start_time"`
}
