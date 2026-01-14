package models

type Role int

// UserRole constants
const (
	RoleUser  Role = 1
	RoleTutor Role = 2
	RoleAdmin Role = 3
)

// User represents a user in the system
type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`    // Never serialize password hash
	Role         Role   `json:"role"` // 1=User, 2=Tutor, 3=Admin, default=1
	Avatar       string `json:"avatar,omitempty"`
	Active       bool   `json:"active"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// UserListItem represents a user in the list response
type UserListItem struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	Avatar   string `json:"avatar,omitempty"`
}

// UserWithSettingsResponse represents a user with settings in the response
type UserWithSettingsResponse struct {
	ID       int           `json:"id"`
	Username string        `json:"username"`
	Email    string        `json:"email"`
	Role     Role          `json:"role"`
	Avatar   string        `json:"avatar,omitempty"`
	Active   bool          `json:"active"`
	Settings *UserSettings `json:"settings,omitempty"`
	Message  string        `json:"message,omitempty"`
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}

// UpdateUserWithSettingsRequest represents a request to update user and settings
type UpdateUserWithSettingsRequest struct {
	Username string                     `json:"username,omitempty"`
	Email    string                     `json:"email,omitempty"`
	Active   *bool                      `json:"active,omitempty"`
	Role     *Role                      `json:"role,omitempty"`
	Settings *UpdateUserSettingsRequest `json:"settings,omitempty"`
}

// TutorListItem represents a tutor in the list response (only ID and username)
type TutorListItem struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// ProfileResponse represents a user profile in the response
type ProfileResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar,omitempty"`
}

// UpdatePasswordRequest represents a request to update password
type UpdatePasswordRequest struct {
	Password string `json:"password"`
}
