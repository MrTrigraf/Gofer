package dto

import "time"

type DirectChatResponse struct {
	ID            string    `json:"id"`
	OtherUserID   string    `json:"other_user_id"`
	OtherUsername string    `json:"other_username"`
	CreatedAt     time.Time `json:"created_at"`
}
