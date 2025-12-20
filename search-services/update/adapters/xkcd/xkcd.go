package xkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"yadro.com/course/update/core"
)

type Client struct {
	log    *slog.Logger
	client http.Client
	url    string
}

func NewClient(url string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	return &Client{
		client: http.Client{Timeout: timeout},
		log:    log,
		url:    url,
	}, nil
}

func (c Client) Get(ctx context.Context, id int) (core.XKCDInfo, error) {
	url := c.url + "/" + strconv.Itoa(id) + "/info.0.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return core.XKCDInfo{}, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return core.XKCDInfo{}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("closing body error", "error", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return core.XKCDInfo{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.XKCDInfo{}, fmt.Errorf("read response body: %v", err)
	}
	var res struct {
		ID          int    `json:"num"`
		URL         string `json:"img"`
		Title       string `json:"title"`
		Description string `json:"transcript"`
		Alt         string `json:"alt"`
		SafeTitle   string `json:"safe_title"`
	}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		c.log.Error("JSON decode error", "id", id, "body", string(bodyBytes), "error", err)
		return core.XKCDInfo{}, fmt.Errorf("decode Json: %v", err)
	}
	title := res.Title
	if title == "" {
		title = res.SafeTitle
	}
	return core.XKCDInfo{
		ID:          res.ID,
		URL:         res.URL,
		Title:       title,
		Description: res.Description,
		Alt:         res.Alt,
	}, nil
}

func (c Client) LastID(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/info.0.json", nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("closing body error", "error", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var res struct {
		Num int `json:"num"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	return res.Num, nil
}
