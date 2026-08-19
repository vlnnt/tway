package wtv

const (
	maxAttempts = 3
	userParam   = "?user_lang=en&platform=web"
	baseUrl     = "https://w.tv/"
	profileUrl  = "https://profiles-service.w.tv/api/v1/profiles/by-nickname/"
	channelUrl  = "https://streams-search-service.w.tv/api/v1/channels/"
	userAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"
)

type profileResponse struct {
	Profile profile  `json:"profile"`
	Tags    []string `json:"tags"`
}

type profile struct {
	UserID   string `json:"userId"`
	Nickname string `json:"nickname"`
}

type channelResponse struct {
	Channel channel `json:"channel"`
}

type subcategory struct {
	SubcategoryTag string `json:"subcategoryTag"`
	Name           string `json:"name"`
	ImageURL       string `json:"imageUrl"`
	Viewers        int    `json:"viewers"`
	Followers      int    `json:"followers"`
}

type liveStream struct {
	StreamID     string       `json:"streamId"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	State        string       `json:"state"`
	StartedAt    string       `json:"startedAt"`
	FinishedAt   *string      `json:"finishedAt"`
	ThumbnailURL string       `json:"thumbnailUrl"`
	PlaybackURL  string       `json:"playbackUrl"`
	Subcategory  *subcategory `json:"subcategory"`
	Viewers      int          `json:"viewers"`
	Views        int          `json:"views"`
	Bitrate      int64        `json:"bitrate"`
}

type channel struct {
	ChannelID    string      `json:"channelId"`
	Name         string      `json:"name"`
	ImageURL     string      `json:"imageUrl"`
	Followers    int         `json:"followers"`
	Live         bool        `json:"live"`
	LiveStreamID string      `json:"liveStreamId"`
	LiveStream   *liveStream `json:"liveStream"`
	Verified     bool        `json:"verified"`
}
