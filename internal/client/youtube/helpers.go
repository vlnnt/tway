package youtube

import (
	"encoding/json"
	"fmt"
	"time"
	"tway/internal/client"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

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
		Channel: channel,
		URL:     baseWatchUrl + videoID,
		IsLive:  false,
	}

	live := playerResponse.Microformat.
		PlayerMicroformatRenderer.
		LiveBroadcastDetails

	if live.StartTimestamp != "" {
		lastStreamAt, err := time.Parse(
			time.RFC3339Nano,
			live.StartTimestamp,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"parse YouTube stream start time %q: %w",
				live.StartTimestamp,
				err,
			)
		}

		streamResult.LastStreamAt = lastStreamAt
	}

	if !live.IsLiveNow {
		return streamResult, nil
	}

	streamResult.Title = playerResponse.VideoDetails.Title
	streamResult.Subcategory = playerResponse.
		Microformat.PlayerMicroformatRenderer.Category
	streamResult.IsLive = true

	return streamResult, nil
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

func (c *Client) getLastStream(
	channel string,
) (string, error) {
	channelID, err := c.resolveChannelID(channel)
	if err != nil {
		return "", fmt.Errorf("resolve YouTube channel ID: %w", err)
	}

	requestBody := browseRequest{
		Context: innertubeContext{
			Client: innertubeClient{
				ClientName:    clientName,
				ClientVersion: clientVersion,
				HL:            "en",
				GL:            "US",
			},
		},
		BrowseID: channelID,
		Params:   streamsParams,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("encode YouTube browse request: %w", err)
	}

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)

	request.SetRequestURI(browseUrl + "key=" + innertubeAPIKey)
	request.Header.SetMethod(fasthttp.MethodPost)
	request.Header.SetContentType(contentTypeHeader)
	request.Header.Set("Accept", acceptHeader)
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("X-Youtube-Client-Name", clientNameID)
	request.Header.Set("X-Youtube-Client-Version", clientVersion)
	request.SetBody(body)

	c.log.Info(
		"Sending YouTube last stream request",
		zap.String("Channel", channel),
		zap.String("ChannelID", channelID),
	)

	if err := c.httpClient.DoTimeout(
		request,
		response,
		c.timeout,
	); err != nil {
		return "", fmt.Errorf(
			"send YouTube last stream request: %w",
			err,
		)
	}

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"YouTube returned an unexpected response",
			zap.Int("Status code", response.StatusCode()),
			zap.ByteString("Body", response.Body()),
		)

		return "", fmt.Errorf(
			"YouTube returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	videoID, err := findFirstVideoID(response.Body())
	if err != nil {
		return "", fmt.Errorf("find last YouTube stream video ID: %w", err)
	}

	return videoID, nil
}

func findFirstVideoID(
	body []byte,
) (string, error) {
	var value any
	if err := json.Unmarshal(
		body, &value,
	); err != nil {
		return "", fmt.Errorf("decode YouTube browse response: %w", err)
	}

	return findVideoID(value), nil
}

func findVideoID(
	value any,
) string {
	switch value := value.(type) {
	case map[string]any:
		if videoID, ok := value["videoId"].(string); ok &&
			videoID != "" {
			return videoID
		}

		for _, child := range value {
			if videoID := findVideoID(child); videoID != "" {
				return videoID
			}
		}

	case []any:
		for _, child := range value {
			if videoID := findVideoID(child); videoID != "" {
				return videoID
			}
		}
	}

	return ""
}
