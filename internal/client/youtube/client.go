package youtube

import (
	"encoding/json"
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
		return nil, fmt.Errorf("resolve YouTube channel ID: %w", err)
	}

	videoID, err := c.resolveLiveVideoID(channelID)
	if err != nil {
		return nil, fmt.Errorf("resolve YouTube live video: %w", err)
	}

	if videoID == "" {
		c.log.Info(
			"YouTube channel is offline",
			zap.String("Channel", channel),
		)

		return &client.Stream{
			Channel: channel,
			URL:     baseUrl + channel,
			IsLive:  false,
		}, nil
	}

	stream, err := c.getPlayerStream(
		channel,
		videoID,
	)
	if err != nil {
		return nil, fmt.Errorf("get YouTube player info: %w", err)
	}

	if !stream.IsLive {
		c.log.Info(
			"YouTube channel is offline",
			zap.String("Channel", channel),
		)

		return stream, nil
	}

	c.log.Info(
		"YouTube channel is live",
		zap.String("Channel", stream.Channel),
		zap.String("Subcategory", stream.Subcategory),
		zap.String("Title", stream.Title),
	)

	return stream, nil
}

func (c *Client) resolveChannelID(
	channel string,
) (string, error) {
	c.channelIDsMu.RLock()
	channelID, ok := c.channelIDs[channel]
	c.channelIDsMu.RUnlock()

	if ok {
		return channelID, nil
	}

	requestBody := resolveRequest{
		Context: innertubeContext{
			Client: innertubeClient{
				ClientName:    clientName,
				ClientVersion: clientVersion,
			},
		},
		URL: baseUrl + channel,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("encode YouTube channel resolve request: %w", err)
	}

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(resolveUrl)
	request.Header.SetMethod(fasthttp.MethodPost)
	request.Header.SetContentType(contentTypeHeader)
	request.Header.Set("Accept", acceptHeader)
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("X-Youtube-Client-Name", clientNameID)
	request.Header.Set("X-Youtube-Client-Version", clientVersion)
	request.SetBody(body)

	c.log.Info(
		"Sending YouTube channel resolve request",
		zap.String("Channel", channel),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return "", fmt.Errorf("send YouTube channel resolve request: %w", err)
	}

	c.log.Info(
		"YouTube channel resolve response received",
		zap.Int("Status code", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"YouTube channel resolve endpoint returned an unexpected response",
			zap.Int("Status code", response.StatusCode()),
			zap.ByteString("Body", response.Body()),
		)

		return "", fmt.Errorf(
			"YouTube channel resolve endpoint returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var resolveResponse resolveResponse
	if err := json.Unmarshal(
		response.Body(), &resolveResponse,
	); err != nil {
		return "", fmt.Errorf("decode YouTube channel resolve response: %w", err)
	}

	channelID = resolveResponse.Endpoint.BrowseEndpoint.BrowseID
	if channelID == "" {
		return "", fmt.Errorf("YouTube channel %q not found", channel)
	}

	c.channelIDsMu.Lock()
	c.channelIDs[channel] = channelID
	c.channelIDsMu.Unlock()

	c.log.Info(
		"YouTube channel ID resolved",
		zap.String("Channel", channel),
		zap.String("ChannelID", channelID),
	)

	return channelID, nil
}

func (c *Client) resolveLiveVideoID(
	channelID string,
) (string, error) {
	requestBody := resolveRequest{
		Context: innertubeContext{
			Client: innertubeClient{
				ClientName:    clientName,
				ClientVersion: clientVersion,
			},
		},
		URL: channelUrl + channelID + "/live",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("encode YouTube resolve request: %w", err)
	}

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(resolveUrl)
	request.Header.SetMethod(fasthttp.MethodPost)
	request.Header.SetContentType(contentTypeHeader)
	request.Header.Set("Accept", acceptHeader)
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("X-Youtube-Client-Name", clientNameID)
	request.Header.Set("X-Youtube-Client-Version", clientVersion)
	request.SetBody(body)

	c.log.Info(
		"Sending YouTube resolve request",
		zap.String("ChannelID", channelID),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return "", fmt.Errorf("send YouTube resolve request: %w", err)
	}

	c.log.Info(
		"YouTube resolve response received",
		zap.Int("Status code", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"YouTube resolve endpoint returned an unexpected response",
			zap.Int("Status code", response.StatusCode()),
			zap.ByteString("Body", response.Body()),
		)

		return "", fmt.Errorf(
			"YouTube resolve endpoint returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var resolveResponse resolveResponse
	if err := json.Unmarshal(
		response.Body(), &resolveResponse,
	); err != nil {
		return "", fmt.Errorf("decode YouTube resolve response: %w", err)
	}

	return resolveResponse.Endpoint.WatchEndpoint.VideoID, nil
}

func (c *Client) getPlayerStream(
	channel, videoID string,
) (*client.Stream, error) {
	requestBody := playerRequest{
		Context: innertubeContext{
			Client: innertubeClient{
				ClientName:    clientName,
				ClientVersion: clientVersion,
				HL:            "en",
			},
		},
		VideoID: videoID,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode YouTube player request: %w", err)
	}

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(playerUrl)
	request.Header.SetMethod(fasthttp.MethodPost)
	request.Header.SetContentType(contentTypeHeader)
	request.Header.Set("Accept", acceptHeader)
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("X-Youtube-Client-Name", clientNameID)
	request.Header.Set("X-Youtube-Client-Version", clientVersion)
	request.SetBody(body)

	c.log.Info(
		"Sending YouTube player request",
		zap.String("URL", playerUrl),
		zap.String("VideoID", videoID),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return nil, fmt.Errorf("send YouTube player request: %w", err)
	}

	c.log.Info(
		"YouTube player response received",
		zap.Int("Status code", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"YouTube player endpoint returned an unexpected response",
			zap.Int("Status code", response.StatusCode()),
			zap.ByteString("Body", response.Body()),
		)

		return nil, fmt.Errorf(
			"YouTube player endpoint returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var playerResponse playerResponse
	if err := json.Unmarshal(
		response.Body(), &playerResponse,
	); err != nil {
		return nil, fmt.Errorf("decode YouTube player response: %w", err)
	}

	streamResult := &client.Stream{
		ID:      videoID,
		Channel: channel,
		URL:     baseWatchUrl + videoID,
		IsLive:  false,
	}

	live := playerResponse.Microformat.
		PlayerMicroformatRenderer.
		LiveBroadcastDetails

	if !live.IsLiveNow {
		return streamResult, nil
	}

	startedAt, err := time.Parse(time.RFC3339, live.StartTimestamp)
	if err != nil {
		return nil, fmt.Errorf("parse YouTube stream start time: %w", err)
	}

	streamResult.ID = playerResponse.VideoDetails.VideoID
	streamResult.Title = playerResponse.VideoDetails.Title
	streamResult.Subcategory = playerResponse.Microformat.
		PlayerMicroformatRenderer.Category
	streamResult.StartedAt = startedAt
	streamResult.IsLive = true

	return streamResult, nil
}
