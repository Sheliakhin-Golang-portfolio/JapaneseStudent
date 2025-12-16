package models

type Role int

// UserRole constants
const (
	RoleUser  Role = 1
	RoleAdmin Role = 2
)

// User represents a user in the system
type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`    // Never serialize password hash
	Role         Role   `json:"role"` // 1=User, 2=Admin, default=1
}
