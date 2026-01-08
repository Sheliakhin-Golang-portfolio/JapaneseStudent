package models

import "time"

// UserToken represents a refresh token for a user
type UserToken struct {
	ID        int       `json:"id"`
	UserID    int       `json:"userId"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
}
