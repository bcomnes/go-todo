package models

import "time"

// Todo is the public representation of one user-owned task. UserID is retained
// for internal ownership checks but is never exposed in JSON responses.
type Todo struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"`
	Task      string    `json:"task"`
	Done      bool      `json:"done"`
	Note      *string   `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
