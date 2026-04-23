package domain

import "time"

type Channel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by"`
	Members   []User    `json:"members,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
