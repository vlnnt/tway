package twitch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type Client struct {
	httpClient *fasthttp.Client
	timeout    time.Duration
}

func NewClient() *Client {
	return &Client{
		httpClient: &fasthttp.Client{
			Name: "TwitchWatcher",
		},
		timeout: 10 * time.Second,
	}
}

func (c *Client) GetStream(
	channel string,
) (*Stream, error) {
	channel = strings.TrimSpace(strings.ToLower(channel))
	if channel == "" {
		return nil, fmt.Errorf("channel cannot be empty")
	}

	requestBody := streamMetadataRequest{
		OperationName: "StreamMetadata",
		Variables: streamMetadataVariables{
			ChannelLogin: channel,
			IncludeIsDJ:  true,
		},
		Extensions: streamMetadataExtensions{
			PersistedQuery: persistedQuery{
				Version:    1,
				SHA256Hash: streamMetadataHash,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(apiURL)
	request.Header.SetMethod(fasthttp.MethodPost)
	request.Header.SetContentType("application/json")
	request.Header.Set("Client-ID", clientID)
	request.Header.Set("Accept", "application/json")
	request.SetBody(body)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if response.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf(
			"twitch returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var result streamMetadataResponse
	if err := json.Unmarshal(response.Body(), &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf(
			"twitch GraphQL error: %s",
			result.Errors[0].Message,
		)
	}

	if result.Data.User == nil {
		return nil, fmt.Errorf("channel %q not found", channel)
	}

	if result.Data.User.Stream == nil {
		return &Stream{
			Channel: channel,
			Title:   result.Data.User.LastBroadcast.Title,
			IsLive:  false,
		}, nil
	}

	stream := result.Data.User.Stream
	startedAt, err := time.Parse(time.RFC3339, stream.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse stream start time: %w", err)
	}

	return &Stream{
		ID:        stream.ID,
		Channel:   channel,
		Title:     result.Data.User.LastBroadcast.Title,
		Game:      stream.Game.Name,
		StartedAt: startedAt,
		IsLive:    stream.Type == "live",
	}, nil
}
