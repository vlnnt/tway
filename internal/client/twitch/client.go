package twitch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"tway/internal/client"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const maxAttempts = 3

type Client struct {
	log        *zap.Logger
	httpClient *fasthttp.Client
	timeout    time.Duration
}

func NewClient(
	log *zap.Logger,
) client.Client {
	return &Client{
		log: log,
		httpClient: &fasthttp.Client{
			Name: "tway",
		},
		timeout: 10 * time.Second,
	}
}

func (c *Client) GetStream(
	channel string,
) (*client.Stream, error) {
	channel = strings.TrimSpace(strings.ToLower(channel))
	if channel == "" {
		return nil, fmt.Errorf("channel cannot be empty")
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		stream, err := c.getStream(channel)
		if err == nil {
			return stream, nil
		}

		lastErr = err
		if attempt < maxAttempts {
			c.log.Warn(
				"Twitch GetStream failed, retrying ...",
				zap.String("Channel", channel),
				zap.Int("Attempt", attempt),
				zap.Int("MaxAttempts", maxAttempts),
				zap.Error(err),
			)

			time.Sleep(time.Second * 3)
		}
	}

	return nil, fmt.Errorf(
		"failed to get stream %q after %d attempts: %w",
		channel,
		maxAttempts,
		lastErr,
	)
}

func (c *Client) getStream(
	channel string,
) (*client.Stream, error) {
	c.log.Info("Checking Twitch channel ...",
		zap.String("Channel", channel))
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

	c.log.Info("Sending request",
		zap.String("URL", apiURL))
	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	c.log.Info("HTTP Status",
		zap.Int("Code", response.StatusCode()))
	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Info("Response", zap.ByteString("Body", response.Body()))
		return nil, fmt.Errorf(
			"twitch returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var result streamMetadataResponse
	if err := json.Unmarshal(
		response.Body(),
		&result,
	); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf(
			"twitch GraphQL error: %s", result.Errors[0].Message,
		)
	}

	if result.Data.User == nil {
		return nil, fmt.Errorf("channel %q not found", channel)
	}

	if result.Data.User.Stream == nil {
		c.log.Info(
			"Channel is offline", zap.String("Channel", channel))
		return &client.Stream{
			Channel: channel,
			Title:   result.Data.User.LastBroadcast.Title,
			IsLive:  false,
		}, nil
	}

	stream := result.Data.User.Stream
	startedAt, err := time.Parse(
		time.RFC3339,
		stream.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("parse stream start time: %w", err)
	}

	c.log.Info(
		"LIVE",
		zap.String("Channel", channel),
		zap.String("Game", stream.Game.Name),
		zap.String("Title", result.Data.User.LastBroadcast.Title),
	)

	return &client.Stream{
		ID:        stream.ID,
		Channel:   channel,
		Title:     result.Data.User.LastBroadcast.Title,
		Game:      stream.Game.Name,
		StartedAt: startedAt,
		IsLive:    stream.Type == "live",
	}, nil
}
