package kick

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

type category struct {
	Name string `json:"name"`
}
