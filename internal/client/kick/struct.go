package kick

const (
	maxAttempts          = 3
	kickBaseUrl          = "https://kick.com/"
	kickAcceptHeader     = "application/json, text/plain, */*"
	kickUserAgent        = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0 Safari/537.36"
	kickApiChannelsRoute = "https://kick.com/api/v2/channels/"
)

type category struct {
	Name string `json:"name"`
}

type channelResponse struct {
	Slug       string      `json:"slug"`
	Livestream *livestream `json:"livestream"`
}

type livestream struct {
	ID        int64     `json:"id"`
	Title     string    `json:"session_title"`
	CreatedAt string    `json:"created_at"`
	Category  *category `json:"category"`
}
