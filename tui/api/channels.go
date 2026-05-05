package api

import (
	"context"
	"net/http"
	"time"
)

type Channel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func (c *Client) ListChannels(ctx context.Context) ([]Channel, error) {
	var channels []Channel
	err := c.do(ctx, http.MethodGet, "/api/v1/channels", nil, &channels)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

func (c *Client) LeaveChannel(ctx context.Context, channelID string) error {
	return c.do(ctx, http.MethodPost, "/api/v1/channels/"+channelID+"/leave", nil, nil)
}

func (c *Client) DeleteChannel(ctx context.Context, channelID string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/channels/"+channelID, nil, nil)
}

func (c *Client) CreateChannel(ctx context.Context, name string) (Channel, error) {
	body := map[string]string{"name": name}
	var resp Channel
	err := c.do(ctx, http.MethodPost, "/api/v1/channels", body, &resp)
	if err != nil {
		return Channel{}, err
	}
	return resp, nil
}

func (c *Client) JoinChannel(ctx context.Context, channelID string) error {
	return c.do(ctx, http.MethodPost, "/api/v1/channels/"+channelID+"/join", nil, nil)
}
