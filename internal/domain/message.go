package domain

import "time"

type Message struct {
	ID           string
	UserID       string
	Content      string
	ChannelID    *string
	DirectChatID *string
	ClientMsgID  *string
	CreatedAt    time.Time
}
