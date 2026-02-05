package xkcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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

type xkcdResp struct {
	Num        int    `json:"num"`
	Img        string `json:"img"`
	Title      string `json:"title"`
	SafeTitle  string `json:"safe_title"`
	Alt        string `json:"alt"`
	Transcript string `json:"transcript"`
}

func (c Client) Get(ctx context.Context, id int) (core.XKCDInfo, error) {

	u := fmt.Sprintf("%s/%d/info.0.json", c.url, id)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return core.XKCDInfo{}, err
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			c.log.Debug("close body failed", "error", e)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return core.XKCDInfo{}, fmt.Errorf("xkcd get %d: http %d", id, resp.StatusCode)
	}

	var xr xkcdResp
	if err := json.NewDecoder(resp.Body).Decode(&xr); err != nil {
		return core.XKCDInfo{}, err
	}

	parts := []string{}
	if strings.TrimSpace(xr.SafeTitle) != "" {
		parts = append(parts, xr.SafeTitle)
	}
	if strings.TrimSpace(xr.Transcript) != "" {
		parts = append(parts, xr.Transcript)
	}
	if strings.TrimSpace(xr.Alt) != "" {
		parts = append(parts, xr.Alt)
	}
	desc := strings.Join(parts, " ")

	return core.XKCDInfo{
		ID:          xr.Num,
		URL:         xr.Img,
		Title:       xr.Title,
		Description: desc,
	}, nil
}

func (c Client) LastID(ctx context.Context) (int, error) {

	u := fmt.Sprintf("%s/info.0.json", c.url)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			c.log.Debug("close body failed", "error", e)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("xkcd last id: http %d", resp.StatusCode)
	}

	var xr xkcdResp
	if err := json.NewDecoder(resp.Body).Decode(&xr); err != nil {
		return 0, err
	}
	if xr.Num <= 0 {
		return 0, errors.New("bad num from xkcd")
	}
	return xr.Num, nil
}
