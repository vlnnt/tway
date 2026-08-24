package youtube

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"tway/internal/client"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"go.uber.org/zap"
)

type Client struct {
	log          *zap.Logger
	httpClient   *fasthttp.Client
	timeout      time.Duration
	channelIDsMu sync.RWMutex
	channelIDs   map[string]string
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
			"Using SOCKS proxy for YouTube",
			zap.String("Proxy", socksProxy),
		)

	case httpProxy != "":
		httpClient.Dial = fasthttpproxy.FasthttpHTTPDialerTimeout(
			httpProxy,
			10*time.Second,
		)

		log.Info(
			"Using HTTP proxy for YouTube",
			zap.String("Proxy", httpProxy),
		)

	default:
		log.Info("Using direct connection for YouTube")
	}

	return &Client{
		log:        log,
		httpClient: httpClient,
		timeout:    10 * time.Second,
		channelIDs: make(map[string]string),
	}
}

func (c *Client) GetStream(
	channel string,
) (*client.Stream, error) {
	channel = strings.TrimSpace(channel)
	channel = strings.TrimPrefix(channel, "@")
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
				"Failed to get YouTube stream, retrying",
				zap.String("Channel", channel),
				zap.Int("Attempt", attempt),
				zap.Int("Max attempts", maxAttempts),
				zap.Error(err),
			)

			time.Sleep(3 * time.Second)
		}
	}

	return nil, fmt.Errorf(
		"failed to get YouTube stream %q after %d attempts: %w",
		channel,
		maxAttempts,
		lastErr,
	)
}

func (c *Client) getStream(
	channel string,
) (*client.Stream, error) {
	c.log.Info(
		"Checking YouTube channel",
		zap.String("Channel", channel),
	)

	channelID, err := c.resolveChannelID(channel)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve YouTube channel ID: %w",
			err,
		)
	}

	streamResult := &client.Stream{
		Channel: channel,
		URL:     baseUrl + channel,
		IsLive:  false,
	}

	lastStream, err := c.getLastStream(channel)
	if err != nil {
		c.log.Warn(
			"Failed to get last YouTube stream",
			zap.String("Channel", channel),
			zap.Error(err),
		)

	} else if lastStream != nil {
		streamResult.LastStreamAt = lastStream.LastStreamAt
	}

	videoID, err := c.resolveLiveVideoID(channelID)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve YouTube live video: %w",
			err,
		)
	}

	if videoID == "" {
		c.log.Info(
			"YouTube channel is offline",
			zap.String("Channel", channel),
			zap.Time(
				"Last stream",
				streamResult.LastStreamAt,
			),
		)

		return streamResult, nil
	}

	stream, err := c.getPlayerStream(channel, videoID)
	if err != nil {
		return nil, fmt.Errorf(
			"get YouTube player info: %w",
			err,
		)
	}

	if !stream.IsLive {
		c.log.Info(
			"YouTube channel is offline",
			zap.String("Channel", channel),
			zap.Time(
				"Last stream",
				streamResult.LastStreamAt,
			),
		)

		return streamResult, nil
	}

	streamResult.Title = stream.Title
	streamResult.Subcategory = stream.Subcategory
	streamResult.URL = stream.URL
	streamResult.IsLive = true
	streamResult.StartedAt = stream.StartedAt

	c.log.Info(
		"YouTube channel is live",
		zap.String("Channel", streamResult.Channel),
		zap.String("Subcategory", streamResult.Subcategory),
		zap.String("Title", streamResult.Title),
		zap.Time("Last stream", streamResult.LastStreamAt),
		zap.Time("Started at", streamResult.StartedAt),
	)

	return streamResult, nil
}
