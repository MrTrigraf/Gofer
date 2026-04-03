package domain

import "time"

type Channel struct {
	ID        string
	Name      string
	CreatedBy string
	Members   []User
	CreatedAt time.Time
}
