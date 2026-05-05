package api

import (
	"context"
	"net/http"
	"time"
)

type DirectChat struct {
	ID            string    `json:"id"`
	OtherUserID   string    `json:"other_user_id"`
	OtherUsername string    `json:"other_username"`
	CreatedAt     time.Time `json:"created_at"`
}

func (c *Client) ListDMs(ctx context.Context) ([]DirectChat, error) {
	var dms []DirectChat
	err := c.do(ctx, http.MethodGet, "/api/v1/direct", nil, &dms)
	if err != nil {
		return nil, err
	}
	return dms, nil
}

func (c *Client) DeleteDM(ctx context.Context, chatID string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/direct/"+chatID, nil, nil)
}

func (c *Client) StartDM(ctx context.Context, userID string) (DirectChat, error) {
	var resp DirectChat
	err := c.do(ctx, http.MethodPost, "/api/v1/direct/"+userID, nil, &resp)
	if err != nil {
		return DirectChat{}, err
	}
	return resp, nil
}
