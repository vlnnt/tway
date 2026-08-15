package youtube

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

const (
	maxAttempts = 3

	resolveURL = "https://www.youtube.com/youtubei/v1/navigation/resolve_url?prettyPrint=false"
	playerURL  = "https://www.youtube.com/youtubei/v1/player?prettyPrint=false"

	clientName    = "WEB"
	clientNameID  = "1"
	clientVersion = "2.20260708.00.00"
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
		log.Info("Using SOCKS proxy for YouTube")

	case httpProxy != "":
		httpClient.Dial = fasthttpproxy.FasthttpHTTPDialerTimeout(
			httpProxy,
			10*time.Second,
		)
		log.Info("Using HTTP proxy for YouTube")

	default:
		log.Info("Using direct connection for YouTube")
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
	channel = strings.TrimSpace(channel)
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
				"YouTube GetStream failed, retrying ...",
				zap.String("Channel", channel),
				zap.Int("Attempt", attempt),
				zap.Int("MaxAttempts", maxAttempts),
				zap.Error(err),
			)

			time.Sleep(3 * time.Second)
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
	c.log.Info(
		"Checking YouTube channel ...",
		zap.String("Channel", channel),
	)

	videoID, err := c.resolveLiveVideoID(channel)
	if err != nil {
		return nil, fmt.Errorf("resolve live video: %w", err)
	}

	if videoID == "" {
		c.log.Info(
			"Channel is offline",
			zap.String("Channel", channel),
		)

		return &client.Stream{
			Channel: channel,
			URL:     "https://www.youtube.com/channel/" + channel,
			IsLive:  false,
		}, nil
	}

	stream, err := c.getPlayerStream(
		channel,
		videoID,
	)
	if err != nil {
		return nil, fmt.Errorf("get player info: %w", err)
	}

	if !stream.IsLive {
		c.log.Info(
			"Channel is offline", zap.String("Channel", channel))
		return stream, nil
	}

	c.log.Info(
		"LIVE",
		zap.String("Channel", stream.Channel),
		zap.String("Game", stream.Game),
		zap.String("Title", stream.Title),
	)

	return stream, nil
}

func (c *Client) resolveLiveVideoID(
	channel string,
) (string, error) {
	requestBody := resolveRequest{
		Context: innertubeContext{
			Client: innertubeClient{
				ClientName:    clientName,
				ClientVersion: clientVersion,
			},
		},
		URL: "https://www.youtube.com/channel/" + channel + "/live",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf(
			"encode request: %w",
			err,
		)
	}

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(resolveURL)
	request.Header.SetMethod(fasthttp.MethodPost)
	request.Header.SetContentType("application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Mozilla/5.0")
	request.Header.Set("X-Youtube-Client-Name", clientNameID)
	request.Header.Set("X-Youtube-Client-Version", clientVersion)
	request.SetBody(body)

	c.log.Info(
		"Sending request",
		zap.String("URL", resolveURL),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}

	c.log.Info(
		"HTTP Status",
		zap.Int("Code", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Info("Response", zap.ByteString("Body", response.Body()))
		return "", fmt.Errorf(
			"youtube resolve returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var result resolveResponse
	if err := json.Unmarshal(
		response.Body(),
		&result,
	); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Endpoint.WatchEndpoint.VideoID, nil
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
		return nil, fmt.Errorf(
			"encode request: %w",
			err,
		)
	}

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(playerURL)
	request.Header.SetMethod(fasthttp.MethodPost)
	request.Header.SetContentType("application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Mozilla/5.0")
	request.Header.Set("X-Youtube-Client-Name", clientNameID)
	request.Header.Set("X-Youtube-Client-Version", clientVersion)
	request.SetBody(body)

	c.log.Info(
		"Sending request",
		zap.String("URL", playerURL),
		zap.String("VideoID", videoID),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return nil, fmt.Errorf(
			"send request: %w",
			err,
		)
	}

	c.log.Info(
		"HTTP Status",
		zap.Int("Code", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Info(
			"Response",
			zap.ByteString("Body", response.Body()),
		)

		return nil, fmt.Errorf(
			"youtube player returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var result playerResponse
	if err := json.Unmarshal(
		response.Body(),
		&result,
	); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	streamResult := &client.Stream{
		ID:      videoID,
		Channel: channel,
		URL:     "https://www.youtube.com/watch?v=" + videoID,
		IsLive:  false,
	}

	if result.VideoDetails.Author != "" {
		streamResult.Channel = result.VideoDetails.Author
	}

	live := result.Microformat.
		PlayerMicroformatRenderer.
		LiveBroadcastDetails

	if !live.IsLiveNow {
		return streamResult, nil
	}

	startedAt, err := time.Parse(
		time.RFC3339,
		live.StartTimestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("parse stream start time: %w", err)
	}

	streamResult.ID = result.VideoDetails.VideoID
	streamResult.Title = result.VideoDetails.Title
	streamResult.Game = result.Microformat.
		PlayerMicroformatRenderer.Category
	streamResult.StartedAt = startedAt
	streamResult.IsLive = true

	return streamResult, nil
}
