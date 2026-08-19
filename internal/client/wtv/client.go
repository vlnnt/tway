package wtv

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
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
	userIDsMu  sync.RWMutex
	userIDs    map[string]string
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
		httpClient.Dial = fasthttpproxy.FasthttpSocksDialer(
			socksProxy,
		)

		log.Info(
			"Using SOCKS proxy for W.TV",
			zap.String("Proxy", socksProxy),
		)

	case httpProxy != "":
		httpClient.Dial = fasthttpproxy.FasthttpHTTPDialerTimeout(
			httpProxy,
			10*time.Second,
		)

		log.Info(
			"Using HTTP proxy for W.TV",
			zap.String("Proxy", httpProxy),
		)

	default:
		log.Info("Using direct connection for W.TV")
	}

	return &Client{
		log:        log,
		httpClient: httpClient,
		timeout:    10 * time.Second,
		userIDs:    make(map[string]string),
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
				"Failed to get W.TV stream, retrying",
				zap.String("Channel", channel),
				zap.Int("Attempt", attempt),
				zap.Int("Max attempts", maxAttempts),
				zap.Error(err),
			)

			time.Sleep(3 * time.Second)
		}
	}

	return nil, fmt.Errorf(
		"failed to get W.TV stream %q after %d attempts: %w",
		channel,
		maxAttempts,
		lastErr,
	)
}

func (c *Client) getStream(
	channel string,
) (*client.Stream, error) {
	c.log.Info(
		"Checking W.TV channel",
		zap.String("Channel", channel),
	)

	userID, err := c.resolveUserID(channel)
	if err != nil {
		return nil, fmt.Errorf("resolve W.TV user ID: %w", err)
	}

	data, err := c.getChannel(userID)
	if err != nil {
		return nil, fmt.Errorf("get W.TV channel: %w", err)
	}

	streamResult := &client.Stream{
		Channel: channel,
		URL:     baseUrl + channel,
		IsLive:  false,
	}

	if !data.Channel.Live || data.Channel.LiveStream == nil {
		c.log.Info(
			"W.TV channel is offline",
			zap.String("Channel", channel),
		)

		return streamResult, nil
	}

	stream := data.Channel.LiveStream
	startedAt, err := time.Parse(time.RFC3339Nano, stream.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("parse W.TV stream start time: %w", err)
	}

	streamResult.ID = stream.StreamID
	streamResult.Title = stream.Title
	streamResult.StartedAt = startedAt
	streamResult.IsLive = stream.State == "started"

	if stream.Subcategory != nil {
		streamResult.Game = stream.Subcategory.Name
	}

	c.log.Info(
		"W.TV channel is live",
		zap.String("Channel", channel),
		zap.String("Game", streamResult.Game),
		zap.String("Title", streamResult.Title),
	)

	return streamResult, nil
}

func (c *Client) resolveUserID(
	channel string,
) (string, error) {
	c.userIDsMu.RLock()
	userID, ok := c.userIDs[channel]
	c.userIDsMu.RUnlock()

	if ok {
		return userID, nil
	}

	url := profileUrl + url.PathEscape(channel) + userParam
	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(url)
	request.Header.SetMethod(fasthttp.MethodGet)
	request.Header.Set("User-Agent", userAgent)

	c.log.Info(
		"Sending W.TV profile request",
		zap.String("URL", url),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return "", fmt.Errorf("send W.TV profile request: %w", err)
	}

	c.log.Info(
		"W.TV profile response received",
		zap.Int("StatusCode", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"W.TV profile endpoint returned an unexpected response",
			zap.Int("StatusCode", response.StatusCode()),
			zap.ByteString("Body", response.Body()),
		)

		return "", fmt.Errorf(
			"W.TV profile endpoint returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var profileResponse profileResponse
	if err := json.Unmarshal(
		response.Body(),
		&profileResponse,
	); err != nil {
		return "", fmt.Errorf("decode W.TV profile response: %w", err)
	}

	if profileResponse.Profile.UserID == "" {
		return "", fmt.Errorf("W.TV channel %q not found", channel)
	}

	c.userIDsMu.Lock()
	c.userIDs[channel] = profileResponse.Profile.UserID
	c.userIDsMu.Unlock()

	return profileResponse.Profile.UserID, nil
}

func (c *Client) getChannel(
	userID string,
) (*channelResponse, error) {
	url := channelUrl + url.PathEscape(userID) + userParam
	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(url)
	request.Header.SetMethod(fasthttp.MethodGet)
	request.Header.Set("User-Agent", userAgent)

	c.log.Info(
		"Sending W.TV channel request",
		zap.String("URL", url),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return nil, fmt.Errorf("send W.TV channel request: %w", err)
	}

	c.log.Info(
		"W.TV channel response received",
		zap.Int("StatusCode", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"W.TV channel endpoint returned an unexpected response",
			zap.Int("StatusCode", response.StatusCode()),
			zap.ByteString("Body", response.Body()),
		)

		return nil, fmt.Errorf(
			"W.TV channel endpoint returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var channelResponse channelResponse
	if err := json.Unmarshal(
		response.Body(),
		&channelResponse,
	); err != nil {
		return nil, fmt.Errorf("decode W.TV channel response: %w", err)
	}

	if channelResponse.Channel.ChannelID == "" {
		return nil, fmt.Errorf("W.TV channel data is missing")
	}

	return &channelResponse, nil
}
