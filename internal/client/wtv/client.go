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
			delay := client.RetryDelay(attempt)

			c.log.Warn(
				"Failed to get W.TV stream, retrying",
				zap.String("Channel", channel),
				zap.Int("Attempt", attempt),
				zap.Int("Max attempts", maxAttempts),
				zap.Duration("Retry in", delay),
				zap.Error(err),
			)

			time.Sleep(delay)
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
		return nil, fmt.Errorf(
			"resolve W.TV user ID: %w",
			err,
		)
	}

	data, err := c.getChannel(userID)
	if err != nil {
		return nil, fmt.Errorf(
			"get W.TV channel: %w",
			err,
		)
	}

	streamResult := &client.Stream{
		Channel: channel,
		URL:     baseUrl + channel,
		IsLive:  false,
	}

	lastStreamTimestamp, err := c.getLastStreamTimestamp(channel)
	if err != nil {
		c.log.Warn(
			"Failed to get last W.TV stream timestamp",
			zap.String("Channel", channel),
			zap.Error(err),
		)

	} else if lastStreamTimestamp != "" {
		lastStreamAt, parseErr := time.Parse(
			time.RFC3339Nano,
			lastStreamTimestamp,
		)

		if parseErr != nil {
			c.log.Warn(
				"Failed to parse last W.TV stream timestamp",
				zap.String("Channel", channel),
				zap.String(
					"Timestamp",
					lastStreamTimestamp,
				),
				zap.Error(parseErr),
			)

		} else {
			streamResult.LastStreamAt = lastStreamAt
		}
	}

	if !data.Channel.Live || data.Channel.LiveStream == nil {
		c.log.Info(
			"W.TV channel is offline",
			zap.String("Channel", channel),
			zap.Time(
				"Last stream",
				streamResult.LastStreamAt,
			),
		)

		return streamResult, nil
	}

	stream := data.Channel.LiveStream
	streamResult.IsLive = stream.State == "started"

	if !streamResult.IsLive {
		c.log.Info(
			"W.TV channel is offline",
			zap.String("Channel", channel),
			zap.Time(
				"Last stream",
				streamResult.LastStreamAt,
			),
		)

		return streamResult, nil
	}

	streamResult.Title = stream.Title
	if stream.Subcategory != nil {
		streamResult.Subcategory = stream.Subcategory.Name
	}

	if stream.StartedAt != "" {
		startedAt, err := time.Parse(
			time.RFC3339Nano,
			stream.StartedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"parse W.TV stream start time %q: %w",
				stream.StartedAt,
				err,
			)
		}

		streamResult.StartedAt = startedAt
	}

	c.log.Info(
		"W.TV channel is live",
		zap.String("Channel", channel),
		zap.String("Subcategory", streamResult.Subcategory),
		zap.String("Title", streamResult.Title),
		zap.Time("Last stream", streamResult.LastStreamAt),
		zap.Time("Started at", streamResult.StartedAt),
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

func (c *Client) getLastStreamTimestamp(
	channel string,
) (string, error) {
	userID, err := c.resolveUserID(channel)
	if err != nil {
		return "", fmt.Errorf("resolve W.TV user ID: %w", err)
	}

	requestURL := channelUrl +
		url.PathEscape(userID) +
		"/streams" +
		userParam

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(requestURL)
	request.Header.SetMethod(fasthttp.MethodGet)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", userAgent)

	c.log.Info(
		"Sending W.TV streams request",
		zap.String("URL", requestURL),
		zap.String("Channel", channel),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return "", fmt.Errorf("send W.TV streams request: %w", err)
	}

	c.log.Info(
		"W.TV streams response received",
		zap.Int("StatusCode", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"W.TV streams endpoint returned an unexpected response",
			zap.Int("StatusCode", response.StatusCode()),
			zap.ByteString("Body", response.Body()),
		)

		return "", fmt.Errorf(
			"W.TV streams endpoint returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var streamsResponse streamsResponse
	if err := json.Unmarshal(
		response.Body(),
		&streamsResponse,
	); err != nil {
		return "", fmt.Errorf(
			"decode W.TV streams response: %w",
			err,
		)
	}

	for _, stream := range streamsResponse.Data {
		if stream.State == "finished" &&
			stream.StartedAt != "" {
			return stream.StartedAt, nil
		}
	}

	return "", nil
}
