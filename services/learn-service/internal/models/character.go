package models

// Character represents a hiragana/katakana character
type Character struct {
	ID             int    `json:"id"`
	Consonant      string `json:"consonant"` // defines a membership to a consonant group
	Vowel          string `json:"vowel"`     // defines a membership to a vowel group
	EnglishReading string `json:"englishReading"`
	RussianReading string `json:"russianReading"`
	Katakana       string `json:"katakana"`
	Hiragana       string `json:"hiragana"`
}

// AlphabetType represents the type of alphabet (hiragana or katakana)
// Used for filtering characters by alphabet type
type AlphabetType string

const (
	AlphabetTypeHiragana AlphabetType = "hr"
	AlphabetTypeKatakana AlphabetType = "kt"
)

// Locale represents the locale for reading (english or russian)
// Used for filtering characters by locale
type Locale string

const (
	LocaleEnglish Locale = "en"
	LocaleRussian Locale = "ru"
	LocaleGerman  Locale = "de"
)

// CharacterResponse represents a character in most of the API responses
type CharacterResponse struct {
	ID        int    `json:"id"`
	Consonant string `json:"consonant,omitempty"` // Not always present, depends on the context
	Vowel     string `json:"vowel,omitempty"`     // Not always present, depends on the context
	Character string `json:"character"`           // Hiragana or Katakana
	Reading   string `json:"reading"`             // English or Russian reading
}

// ReadingTestItem represents an item in a reading test
type ReadingTestItem struct {
	ID           int      `json:"id"`
	WrongOptions []string `json:"wrongOptions"` // Two wrong character options
	Reading      string   `json:"reading"`      // English or Russian reading
	CorrectChar  string   `json:"correctChar"`  // Correct character
}

// WritingTestItem represents an item in a writing test
type WritingTestItem struct {
	ID             int    `json:"id"`
	CorrectReading string `json:"correctReading"` // English or Russian reading correct reading
	Character      string `json:"Character"`      // Correct character whose reading is to be guessed
}

// CharacterListItem represents a character in the list of characters for admin endpoints
type CharacterListItem struct {
	ID        int    `json:"id"`
	Consonant string `json:"consonant"`
	Vowel     string `json:"vowel"`
	Katakana  string `json:"katakana"`
	Hiragana  string `json:"hiragana"`
}

// CreateCharacterRequest represents a request to create a character
type CreateCharacterRequest struct {
	Consonant      string `json:"consonant"`
	Vowel          string `json:"vowel"`
	EnglishReading string `json:"englishReading"`
	RussianReading string `json:"russianReading"`
	Katakana       string `json:"katakana"`
	Hiragana       string `json:"hiragana"`
}

// UpdateCharacterRequest represents a request to update a character (partial update)
type UpdateCharacterRequest struct {
	Consonant      string `json:"consonant,omitempty"`
	Vowel          string `json:"vowel,omitempty"`
	EnglishReading string `json:"englishReading,omitempty"`
	RussianReading string `json:"russianReading,omitempty"`
	Katakana       string `json:"katakana,omitempty"`
	Hiragana       string `json:"hiragana,omitempty"`
}
