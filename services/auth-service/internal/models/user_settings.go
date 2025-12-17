package models

// Language represents the user's preferred language
type Language string

const (
	LanguageEnglish Language = "en"
	LanguageRussian Language = "ru"
	LanguageGerman  Language = "de"
)

// UserSettings represents user preferences and settings
type UserSettings struct {
	ID                 int      `json:"id"`
	UserID             int      `json:"userId"`
	NewWordCount       int      `json:"newWordCount"`       // Default: 20
	OldWordCount       int      `json:"oldWordCount"`       // Default: 20
	AlphabetLearnCount int      `json:"alphabetLearnCount"` // Default: 10
	Language           Language `json:"language"`           // Default: "en"
}

// UserSettingsResponse represents user settings in API responses (without IDs)
type UserSettingsResponse struct {
	NewWordCount       int      `json:"newWordCount"`
	OldWordCount       int      `json:"oldWordCount"`
	AlphabetLearnCount int      `json:"alphabetLearnCount"`
	Language           Language `json:"language"`
}

// UpdateUserSettingsRequest represents a request to update user settings
type UpdateUserSettingsRequest struct {
	NewWordCount       *int      `json:"newWordCount,omitempty"`
	OldWordCount       *int      `json:"oldWordCount,omitempty"`
	AlphabetLearnCount *int      `json:"alphabetLearnCount,omitempty"`
	Language           *Language `json:"language,omitempty"`
}
