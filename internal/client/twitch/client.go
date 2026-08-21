package twitch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tway/internal/client"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"go.uber.org/zap"
)

type Client struct {
	log        *zap.Logger
	httpClient *fasthttp.Client
	timeout    time.Duration
}

func NewClient(
	log *zap.Logger,
	httpProxy, socksProxy string,
) client.Client {
	httpClient := &fasthttp.Client{
		Name: "tway",
	}

	httpProxy = strings.TrimSpace(httpProxy)
	socksProxy = strings.TrimSpace(socksProxy)

	switch {
	case socksProxy != "":
		httpClient.Dial = fasthttpproxy.FasthttpSocksDialer(socksProxy)

		log.Info(
			"Using SOCKS proxy for Twitch",
			zap.String("Proxy", socksProxy),
		)

	case httpProxy != "":
		httpClient.Dial = fasthttpproxy.FasthttpHTTPDialerTimeout(
			httpProxy,
			10*time.Second,
		)

		log.Info(
			"Using HTTP proxy for Twitch",
			zap.String("Proxy", httpProxy),
		)

	default:
		log.Info("Using direct connection for Twitch")
	}

	return &Client{
		log:        log,
		httpClient: httpClient,
		timeout:    10 * time.Second,
	}
}

func (c *Client) GetStream(
	channel string,
) (*client.Stream, error) {
	channel = strings.TrimSpace(strings.ToLower(channel))
	if channel == "" {
		return nil, fmt.Errorf(
			"channel cannot be empty",
		)
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
				"Failed to get Twitch stream, retrying",
				zap.String("Channel", channel),
				zap.Int("Attempt", attempt),
				zap.Int("Max attempts", maxAttempts),
				zap.Error(err),
			)

			time.Sleep(3 * time.Second)
		}
	}

	return nil, fmt.Errorf(
		"failed to get Twitch stream %q after %d attempts: %w",
		channel,
		maxAttempts,
		lastErr,
	)
}

func (c *Client) getStream(
	channel string,
) (*client.Stream, error) {
	c.log.Info(
		"Checking Twitch channel",
		zap.String("Channel", channel),
	)

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
		return nil, fmt.Errorf("encode Twitch request: %w", err)
	}

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(url)
	request.Header.SetMethod(fasthttp.MethodPost)
	request.Header.SetContentType("application/json")
	request.Header.Set("Client-ID", clientID)
	request.Header.Set("Accept", "application/json")
	request.SetBody(body)

	c.log.Info(
		"Sending Twitch request",
		zap.String("URL", url),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return nil, fmt.Errorf("send Twitch request: %w", err)
	}

	c.log.Info(
		"Twitch response received",
		zap.Int("Status code", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"Twitch returned an unexpected response",
			zap.Int("Status code", response.StatusCode()),
			zap.ByteString("Body", response.Body()),
		)

		return nil, fmt.Errorf(
			"Twitch returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var streamMetadataResponse streamMetadataResponse
	if err := json.Unmarshal(
		response.Body(),
		&streamMetadataResponse,
	); err != nil {
		return nil, fmt.Errorf("decode Twitch response: %w", err)
	}

	if len(streamMetadataResponse.Errors) > 0 {
		return nil, fmt.Errorf(
			"Twitch GraphQL error: %s",
			streamMetadataResponse.Errors[0].Message,
		)
	}

	if streamMetadataResponse.Data.User == nil {
		return nil, fmt.Errorf("Twitch channel %q not found", channel)
	}

	streamResult := &client.Stream{
		Channel: channel,
		URL:     baseUrl + channel,
		IsLive:  false,
	}

	if streamMetadataResponse.Data.User.Stream == nil {
		c.log.Info(
			"Twitch channel is offline",
			zap.String("Channel", channel),
		)

		return streamResult, nil
	}

	stream := streamMetadataResponse.Data.User.Stream
	startedAt, err := time.Parse(
		time.RFC3339,
		stream.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("parse Twitch stream start time: %w", err)
	}

	streamResult.ID = stream.ID
	streamResult.Title = streamMetadataResponse.Data.User.LastBroadcast.Title
	streamResult.Subcategory = stream.Category.Name
	streamResult.StartedAt = startedAt
	streamResult.IsLive = stream.Type == "live"

	c.log.Info(
		"Twitch channel is live",
		zap.String("Channel", channel),
		zap.String("Subcategory", streamResult.Subcategory),
		zap.String("Title", streamResult.Title),
	)

	return streamResult, nil
}
