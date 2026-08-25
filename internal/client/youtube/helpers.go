package youtube

import (
	"bytes"
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
		return nil, fmt.Errorf(
			"encode YouTube player request: %w",
			err,
		)
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
		return nil, fmt.Errorf(
			"send YouTube player request: %w",
			err,
		)
	}

	c.log.Info(
		"YouTube player response received",
		zap.Int(
			"Status code",
			response.StatusCode(),
		),
	)

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"YouTube player endpoint returned an unexpected response",
			zap.Int(
				"Status code",
				response.StatusCode(),
			),
			zap.ByteString(
				"Body",
				response.Body(),
			),
		)

		return nil, fmt.Errorf(
			"YouTube player endpoint returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	var playerResponse playerResponse
	if err := json.Unmarshal(
		response.Body(),
		&playerResponse,
	); err != nil {
		return nil, fmt.Errorf(
			"decode YouTube player response: %w",
			err,
		)
	}

	streamResult := &client.Stream{
		Channel: channel,
		URL:     baseWatchUrl + videoID,
		IsLive:  false,
	}

	live := playerResponse.
		Microformat.
		PlayerMicroformatRenderer.
		LiveBroadcastDetails

	if live.StartTimestamp != "" {
		startedAt, err := time.Parse(
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

		if live.IsLiveNow {
			streamResult.StartedAt = startedAt
		} else {
			streamResult.LastStreamAt = startedAt
		}
	}

	if !live.IsLiveNow {
		return streamResult, nil
	}

	streamResult.Title =
		playerResponse.VideoDetails.Title

	streamResult.Subcategory =
		playerResponse.
			Microformat.
			PlayerMicroformatRenderer.
			Category

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
) (*client.Stream, error) {
	channelID, err := c.resolveChannelID(channel)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve YouTube channel ID: %w",
			err,
		)
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
		return nil, fmt.Errorf(
			"encode YouTube browse request: %w",
			err,
		)
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
		return nil, fmt.Errorf(
			"send YouTube last stream request: %w",
			err,
		)
	}

	if response.StatusCode() != fasthttp.StatusOK {
		c.log.Warn(
			"YouTube returned an unexpected response",
			zap.Int(
				"Status code",
				response.StatusCode(),
			),
			zap.ByteString(
				"Body",
				response.Body(),
			),
		)

		return nil, fmt.Errorf(
			"YouTube returned status %d: %s",
			response.StatusCode(),
			string(response.Body()),
		)
	}

	videoIDs, err := findVideoIDsInOrder(response.Body())
	if err != nil {
		return nil, fmt.Errorf(
			"find YouTube stream video IDs: %w",
			err,
		)
	}

	if len(videoIDs) == 0 {
		c.log.Info(
			"No YouTube stream candidates found",
			zap.String("Channel", channel),
		)

		return nil, nil
	}

	limit := len(videoIDs)
	if limit > maxLastStreamCandidates {
		limit = maxLastStreamCandidates
	}

	c.log.Info(
		"YouTube stream candidates found",
		zap.String("Channel", channel),
		zap.Int("Candidates", len(videoIDs)),
		zap.Int("Checking", limit),
	)

	now := time.Now()
	for _, videoID := range videoIDs[:limit] {
		stream, err := c.getPlayerStream(
			channel,
			videoID,
		)
		if err != nil {
			c.log.Warn(
				"Failed to inspect YouTube stream candidate",
				zap.String("Channel", channel),
				zap.String("VideoID", videoID),
				zap.Error(err),
			)

			continue
		}

		if stream.IsLive {
			continue
		}

		if stream.LastStreamAt.IsZero() {
			continue
		}

		if stream.LastStreamAt.After(now) {
			continue
		}

		c.log.Info(
			"Last YouTube stream found",
			zap.String("Channel", channel),
			zap.String("VideoID", videoID),
			zap.Time(
				"Last stream",
				stream.LastStreamAt,
			),
		)

		return stream, nil
	}

	c.log.Info(
		"No valid YouTube stream found among candidates",
		zap.String("Channel", channel),
		zap.Int("Checked", limit),
	)

	return nil, nil
}

func findVideoIDsInOrder(
	body []byte,
) ([]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))

	videoIDs := make([]string, 0)
	seen := make(map[string]struct{})

	addVideoID := func(videoID string) {

		if videoID == "" {
			return
		}

		if _, exists := seen[videoID]; exists {
			return
		}

		seen[videoID] = struct{}{}
		videoIDs = append(
			videoIDs,
			videoID,
		)
	}

	var walk func() error
	walk = func() error {

		token, err := decoder.Token()
		if err != nil {
			return err
		}

		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}

		switch delim {
		case '{':
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}

				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("unexpected JSON object key")
				}

				switch key {
				case "videoRenderer",
					"gridVideoRenderer",
					"playlistVideoRenderer",
					"compactVideoRenderer":
					var renderer struct {
						VideoID string `json:"videoId"`
					}

					if err := decoder.Decode(&renderer); err != nil {
						return err
					}

					addVideoID(renderer.VideoID)
					continue

				case "lockupViewModel":
					var viewModel struct {
						ContentID   string `json:"contentId"`
						ContentType string `json:"contentType"`
					}

					if err := decoder.Decode(&viewModel); err != nil {
						return err
					}

					if viewModel.ContentType ==
						"LOCKUP_CONTENT_TYPE_VIDEO" {
						addVideoID(viewModel.ContentID)
					}

					continue
				}

				if err := walk(); err != nil {
					return err
				}
			}

			_, err := decoder.Token()
			return err

		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}

			_, err := decoder.Token()
			return err
		}

		return nil
	}

	if err := walk(); err != nil {
		return nil, fmt.Errorf(
			"decode YouTube browse response: %w",
			err,
		)
	}

	return videoIDs, nil
}
