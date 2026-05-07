package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client

	mu          sync.RWMutex
	accessToken string
}

type Message struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) SetAuth(accessToken string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = accessToken
}

func (c *Client) authHeader() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}

type errorResponse struct {
	Error string `json:"error"`
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reqBody *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("api.do: marshal body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("api.do: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if token := c.authHeader(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("api.do: %w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return translateError(resp)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("api.do: decode response: %w", err)
		}
	}
	return nil
}

func translateError(resp *http.Response) error {
	var msg string
	var errResp errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
		msg = errResp.Error
	}

	switch resp.StatusCode {
	case http.StatusBadRequest: // 400
		return fmt.Errorf("%w: %s", ErrBadRequest, msg)
	case http.StatusUnauthorized: // 401
		return fmt.Errorf("%w: %s", ErrInvalidCredentials, msg)
	case http.StatusForbidden: // 403
		return fmt.Errorf("%w: %s", ErrForbidden, msg)
	case http.StatusNotFound: // 404
		return fmt.Errorf("%w: %s", ErrNotFound, msg)
	case http.StatusConflict: // 409
		return fmt.Errorf("%w: %s", ErrConflict, msg)
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: status=%d msg=%q", ErrServer, resp.StatusCode, msg)
	}

	return fmt.Errorf("%w: status=%d msg=%q", ErrUnexpected, resp.StatusCode, msg)
}

func buildHistoryPath(base string, limit int, before time.Time) string {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if !before.IsZero() {
		q.Set("before", before.Format(time.RFC3339))
	}
	if len(q) == 0 {
		return base
	}
	return base + "?" + q.Encode()
}

func (c *Client) WSURL() string {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return c.baseURL + "/ws"
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = "/ws"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func (c *Client) GetDMMessages(ctx context.Context, dmID string, limit int, before time.Time) ([]Message, error) {
	path := buildHistoryPath("/api/v1/direct/"+dmID+"/messages", limit, before)
	var out []Message
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetChannelMessages(ctx context.Context, channelID string, limit int, before time.Time) ([]Message, error) {
	path := buildHistoryPath("/api/v1/channels/"+channelID+"/messages", limit, before)
	var out []Message
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
