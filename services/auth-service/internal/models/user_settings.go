package models

// Language represents the user's preferred language
type Language string

const (
	LanguageEnglish Language = "en"
	LanguageRussian Language = "ru"
	LanguageGerman  Language = "de"
)

// AlphabetRepeat represents the user's alphabet repeat types
type RepeatType string

const (
	RepeatTypeInQuestion RepeatType = "in question"
	RepeatTypeIgnore     RepeatType = "ignore"
	RepeatTypeRepeat     RepeatType = "repeat"
)

// UserSettings represents user preferences and settings
type UserSettings struct {
	ID                 int        `json:"id"`
	UserID             int        `json:"userId"`
	NewWordCount       int        `json:"newWordCount"`       // Default: 20
	OldWordCount       int        `json:"oldWordCount"`       // Default: 20
	AlphabetLearnCount int        `json:"alphabetLearnCount"` // Default: 10
	Language           Language   `json:"language"`           // Default: "en"
	AlphabetRepeat     RepeatType `json:"alphabetRepeat"`     // Default: "in question"
}

// UserSettingsResponse represents user settings in API responses (without IDs)
type UserSettingsResponse struct {
	NewWordCount       int        `json:"newWordCount"`
	OldWordCount       int        `json:"oldWordCount"`
	AlphabetLearnCount int        `json:"alphabetLearnCount"`
	Language           Language   `json:"language"`
	AlphabetRepeat     RepeatType `json:"alphabetRepeat"`
}

// UpdateUserSettingsRequest represents a request to update user settings
type UpdateUserSettingsRequest struct {
	NewWordCount       *int       `json:"newWordCount,omitempty"`
	OldWordCount       *int       `json:"oldWordCount,omitempty"`
	AlphabetLearnCount *int       `json:"alphabetLearnCount,omitempty"`
	Language           Language   `json:"language,omitempty"`
	AlphabetRepeat     RepeatType `json:"alphabetRepeat,omitempty"`
}
