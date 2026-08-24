package kick

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
			"Using SOCKS proxy for Kick",
			zap.String("Proxy", socksProxy),
		)

	case httpProxy != "":
		httpClient.Dial = fasthttpproxy.FasthttpHTTPDialerTimeout(
			httpProxy,
			10*time.Second,
		)

		log.Info(
			"Using HTTP proxy for Kick",
			zap.String("Proxy", httpProxy),
		)

	default:
		log.Info("Using direct connection for Kick")
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
				"Failed to get Kick stream, retrying",
				zap.String("Channel", channel),
				zap.Int("Attempt", attempt),
				zap.Int("Max attempts", maxAttempts),
				zap.Error(err),
			)

			time.Sleep(3 * time.Second)
		}
	}

	return nil, fmt.Errorf(
		"failed to get Kick stream %q after %d attempts: %w",
		channel,
		maxAttempts,
		lastErr,
	)
}

func (c *Client) getStream(
	channel string,
) (*client.Stream, error) {
	url := kickApiChannelsV2Route + channel
	c.log.Info(
		"Checking Kick channel",
		zap.String("Channel", channel),
	)

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(url)
	request.Header.SetMethod(fasthttp.MethodGet)
	request.Header.Set("User-Agent", kickUserAgent)
	request.Header.Set("Accept", kickAcceptHeader)
	request.Header.Set("Referer", kickBaseUrl+channel)

	c.log.Info(
		"Sending Kick request",
		zap.String("URL", url),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return nil, fmt.Errorf(
			"send Kick request: %w",
			err,
		)
	}

	c.log.Info(
		"Kick response received",
		zap.Int("Status code", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"Kick returned an unexpected response",
			zap.Int("Status code", response.StatusCode()),
			zap.ByteString("Body", response.Body()),
		)

		return nil, fmt.Errorf(
			"Kick returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var channelResponse channelResponse
	if err := json.Unmarshal(
		response.Body(),
		&channelResponse,
	); err != nil {
		return nil, fmt.Errorf(
			"decode Kick response: %w",
			err,
		)
	}

	streamResult := &client.Stream{
		Channel: channel,
		URL:     kickBaseUrl + channel,
		IsLive:  channelResponse.Livestream != nil,
	}

	lastStreamTimestamp, err := c.getLastStreamTimestamp(channel)
	if err != nil {
		c.log.Warn(
			"Failed to get last Kick stream timestamp",
			zap.String("Channel", channel),
			zap.Error(err),
		)
	}

	if lastStreamTimestamp != "" {
		lastStreamAt, parseErr := time.Parse(
			"2006-01-02 15:04:05",
			lastStreamTimestamp,
		)
		if parseErr != nil {
			c.log.Warn(
				"Failed to parse last Kick stream timestamp",
				zap.String("Channel", channel),
				zap.String("Timestamp", lastStreamTimestamp),
				zap.Error(parseErr),
			)
		} else {
			streamResult.LastStreamAt = lastStreamAt
		}
	}

	if channelResponse.Livestream == nil {
		c.log.Info(
			"Kick channel is offline",
			zap.String("Channel", channel),
			zap.Time("Last stream", streamResult.LastStreamAt),
		)

		return streamResult, nil
	}

	streamResult.Title = channelResponse.Livestream.Title
	if streamResult.LastStreamAt.IsZero() {
		currentStreamAt, err := time.Parse(
			"2006-01-02 15:04:05",
			channelResponse.Livestream.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"parse current Kick stream timestamp %q: %w",
				channelResponse.Livestream.CreatedAt,
				err,
			)
		}

		streamResult.LastStreamAt = currentStreamAt
	}

	if channelResponse.Livestream.Category != nil {
		streamResult.Subcategory = channelResponse.Livestream.Category.Name
	}

	c.log.Info(
		"Kick channel is live",
		zap.String("Channel", channel),
		zap.String("Subcategory", streamResult.Subcategory),
		zap.String("Title", streamResult.Title),
		zap.Time("Last stream", streamResult.LastStreamAt),
	)

	return streamResult, nil
}

func (c *Client) getLastStreamTimestamp(
	channel string,
) (string, error) {
	url := kickApiChannelsV1Route + channel
	c.log.Info(
		"Checking last Kick stream timestamp",
		zap.String("Channel", channel),
	)

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(url)
	request.Header.SetMethod(fasthttp.MethodGet)
	request.Header.Set("User-Agent", kickUserAgent)
	request.Header.Set("Accept", kickAcceptHeader)
	request.Header.Set("Referer", kickBaseUrl+channel)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return "", fmt.Errorf("send Kick request: %w", err)
	}

	c.log.Info(
		"Kick last stream timestamp response received",
		zap.Int("Status code", response.StatusCode()),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"Kick returned an unexpected response",
			zap.Int("Status code", response.StatusCode()),
			zap.ByteString("Body", response.Body()),
		)

		return "", fmt.Errorf(
			"Kick returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var previousLivestreamResponse previousLivestreamsResponse
	if err := json.Unmarshal(
		response.Body(),
		&previousLivestreamResponse,
	); err != nil {
		return "", fmt.Errorf(
			"decode Kick previous livestreams response: %w",
			err,
		)
	}

	if len(previousLivestreamResponse.PreviousLivestreams) == 0 {
		return "", nil
	}

	return previousLivestreamResponse.
		PreviousLivestreams[0].
		StartTime, nil
}
