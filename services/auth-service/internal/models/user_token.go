package models

// UserToken represents a refresh token for a user
type UserToken struct {
	ID     int    `json:"id"`
	UserID int    `json:"userId"`
	Token  string `json:"token"`
}
