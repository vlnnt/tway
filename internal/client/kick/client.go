package kick

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tway/internal/client"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
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
		log.Info("Using SOCKS proxy for Kick")

	case httpProxy != "":
		httpClient.Dial = fasthttpproxy.FasthttpHTTPDialerTimeout(
			httpProxy,
			10*time.Second,
		)
		log.Info("Using HTTP proxy for Kick")

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
				"Kick GetStream failed, retrying ...",
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
	apiURL := fmt.Sprintf(
		"https://kick.com/api/v2/channels/%s",
		channel,
	)

	c.log.Info(
		"Checking Kick channel ...",
		zap.String("Channel", channel),
	)

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(apiURL)
	request.Header.SetMethod(fasthttp.MethodGet)

	request.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0 Safari/537.36",
	)

	request.Header.Set(
		"Accept",
		"application/json, text/plain, */*",
	)

	request.Header.Set(
		"Referer",
		"https://kick.com/"+channel,
	)

	c.log.Info(
		"Sending request",
		zap.String("URL", apiURL),
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
			zap.ByteString(
				"Body",
				response.Body(),
			),
		)

		return nil, fmt.Errorf(
			"kick returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var data channelResponse
	if err := json.Unmarshal(
		response.Body(),
		&data,
	); err != nil {
		return nil, fmt.Errorf(
			"decode response: %w",
			err,
		)
	}

	streamResult := &client.Stream{
		Channel: channel,
		URL:     "https://kick.com/" + channel,
		IsLive:  data.Livestream != nil,
	}

	if data.Livestream == nil {
		c.log.Info(
			"Channel is offline",
			zap.String("Channel", channel),
		)

		return streamResult, nil
	}

	streamResult.ID = strconv.FormatInt(
		data.Livestream.ID,
		10,
	)

	streamResult.Title = data.Livestream.Title
	startedAt, err := time.Parse(
		"2006-01-02 15:04:05",
		data.Livestream.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"parse stream start time: %w",
			err,
		)
	}

	streamResult.StartedAt = startedAt
	if data.Livestream.Category != nil {
		streamResult.Game = data.Livestream.Category.Name
	}

	c.log.Info(
		"LIVE",
		zap.String("Channel", channel),
		zap.String("Game", streamResult.Game),
		zap.String("Title", streamResult.Title),
	)

	return streamResult, nil
}
