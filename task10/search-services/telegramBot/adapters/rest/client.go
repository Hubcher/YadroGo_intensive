package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"yadro.com/course/telegramBot/core"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger

	adminUser string
	adminPass string
}

func NewClient(baseURL string, log *slog.Logger, adminUser string, adminPass string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Minute * 1,
		},
		log:       log,
		adminUser: adminUser,
		adminPass: adminPass,
	}
}

// Endpoint: POST /api/Login
func (c *Client) login(ctx context.Context) (string, error) {

	body := map[string]string{
		"name":     c.adminUser,
		"password": c.adminPass,
	}

	b, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/login", bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			c.log.Debug("close body failed", "error", e)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed: status: %d", resp.StatusCode)
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(tokenBytes)), nil
}

// Endpoint POST /api/db/update + authorization
func (c *Client) Update(ctx context.Context) (string, error) {
	// for middleware
	token, err := c.login(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot login: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/db/update", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Token "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			c.log.Debug("close body failed", "error", e)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		return "Update finished successfully", nil
	case http.StatusAccepted:
		return "Update already running", nil
	case http.StatusUnauthorized:
		return "", errors.New("unauthorized")
	default:
		return "", fmt.Errorf("update failed: status %d", resp.StatusCode)
	}
}

// Endpoint GET /api/db/stats
func (c *Client) Stats(ctx context.Context) (core.UpdateStatsResponse, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/db/stats", nil)
	if err != nil {
		return core.UpdateStatsResponse{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return core.UpdateStatsResponse{}, err
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			c.log.Debug("close body failed", "error", e)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return core.UpdateStatsResponse{}, fmt.Errorf("stats failed: status: %d", resp.StatusCode)
	}

	var statsResponse core.UpdateStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&statsResponse); err != nil {
		return core.UpdateStatsResponse{}, err
	}

	return statsResponse, nil
}

// Endpoint GET /api/db/status
func (c *Client) Status(ctx context.Context) (core.UpdateStatusResponse, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/db/status", nil)
	if err != nil {
		return core.UpdateStatusResponse{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return core.UpdateStatusResponse{}, err
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			c.log.Debug("close body failed", "error", e)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return core.UpdateStatusResponse{}, fmt.Errorf("status failed: status: %d", resp.StatusCode)
	}

	var statusResponse core.UpdateStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResponse); err != nil {
		return core.UpdateStatusResponse{}, err
	}

	return statusResponse, nil
}

// Endpoint DELETE /api/db + authorization
func (c *Client) Drop(ctx context.Context) error {
	token, err := c.login(ctx)
	if err != nil {
		return fmt.Errorf("cannot login: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/api/db", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Token "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			c.log.Debug("close body failed", "error", e)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("drop failed: status: %d", resp.StatusCode)
	}
	return nil
}

// Endpoint GET /api/search - не используется
func (c *Client) Search(ctx context.Context, phrase string, limit int) (core.SearchResponse, error) {
	q := url.QueryEscape(phrase)
	u := fmt.Sprintf("%s/api/search?phrase=%s&limit=%d", c.baseURL, q, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return core.SearchResponse{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return core.SearchResponse{}, err
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			c.log.Debug("close body failed", "error", e)
		}
	}()

	if resp.StatusCode == http.StatusBadRequest {
		return core.SearchResponse{}, fmt.Errorf("bad request: : %s", resp.Status)
	}

	if resp.StatusCode != http.StatusOK {
		return core.SearchResponse{}, fmt.Errorf("search failed: status: %d", resp.StatusCode)
	}

	var searchResponse core.SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return core.SearchResponse{}, err
	}
	return searchResponse, nil
}

// Endpoint GET /api/isearch
func (c *Client) IndexSearch(ctx context.Context, phrase string, limit int) (core.SearchResponse, error) {
	q := url.QueryEscape(phrase)
	u := fmt.Sprintf("%s/api/isearch?phrase=%s&limit=%d", c.baseURL, q, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return core.SearchResponse{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return core.SearchResponse{}, err
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			c.log.Debug("close body failed", "error", e)
		}
	}()

	if resp.StatusCode == http.StatusBadRequest {
		return core.SearchResponse{}, fmt.Errorf("bad request: : %s", resp.Status)
	}

	if resp.StatusCode != http.StatusOK {
		return core.SearchResponse{}, fmt.Errorf("index search failed: status: %d", resp.StatusCode)
	}

	var searchResponse core.SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return core.SearchResponse{}, err
	}
	return searchResponse, nil
}
