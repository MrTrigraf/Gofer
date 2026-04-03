package domain

import "time"

type DirectChat struct {
	ID        string
	UserID1   string
	UserID2   string
	CreatedAt time.Time
}
