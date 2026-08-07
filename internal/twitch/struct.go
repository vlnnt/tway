package twitch

import "time"

const (
	apiURL             = "https://gql.twitch.tv/gql"
	clientID           = "kimne78kx3ncx6brgo4mv6wki5h1ko"
	streamMetadataHash = "b57f9b910f8cd1a4659d894fe7550ccc81ec9052c01e438b290fd66a040b9b93"
)

type Stream struct {
	ID        string
	Channel   string
	Title     string
	Game      string
	StartedAt time.Time
	IsLive    bool
}

type persistedQuery struct {
	Version    int    `json:"version"`
	SHA256Hash string `json:"sha256Hash"`
}

type streamMetadataVariables struct {
	ChannelLogin string `json:"channelLogin"`
	IncludeIsDJ  bool   `json:"includeIsDJ"`
}

type streamMetadataExtensions struct {
	PersistedQuery persistedQuery `json:"persistedQuery"`
}

type streamMetadataRequest struct {
	OperationName string                   `json:"operationName"`
	Variables     streamMetadataVariables  `json:"variables"`
	Extensions    streamMetadataExtensions `json:"extensions"`
}

type gameResponse struct {
	Name string `json:"name"`
}

type broadcastResponse struct {
	Title string `json:"title"`
}

type streamResponse struct {
	ID        string       `json:"id"`
	Type      string       `json:"type"`
	CreatedAt string       `json:"createdAt"`
	Game      gameResponse `json:"game"`
}

type graphqlError struct {
	Message string `json:"message"`
}

type userResponse struct {
	LastBroadcast broadcastResponse `json:"lastBroadcast"`
	Stream        *streamResponse   `json:"stream"`
}

type streamMetadataResponse struct {
	Data struct {
		User *userResponse `json:"user"`
	} `json:"data"`

	Errors []graphqlError `json:"errors"`
}
