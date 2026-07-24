// Package models contains small application records shared across HTTP and web
// presentation packages.
package models

import "time"

// User is the public account representation. It deliberately excludes the
// password hash and authentication-token records stored by PostgreSQL.
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
