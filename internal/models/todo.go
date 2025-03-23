package models

import "time"

type Todo struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Task      string    `json:"task"`
	Done      bool      `json:"done"`
	Note      *string   `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
