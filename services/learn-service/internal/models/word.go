package models

// Word represents a Japanese word in the dictionary
type Word struct {
	ID                        int    `json:"id"`
	Word                      string `json:"word"`          // Kanji word
	PhoneticClues             string `json:"phoneticClues"` // Hiragana reading
	RussianTranslation        string `json:"russianTranslation"`
	EnglishTranslation        string `json:"englishTranslation"`
	GermanTranslation         string `json:"germanTranslation"`
	Example                   string `json:"example"` // Japanese sentence
	ExampleRussianTranslation string `json:"exampleRussianTranslation"`
	ExampleEnglishTranslation string `json:"exampleEnglishTranslation"`
	ExampleGermanTranslation  string `json:"exampleGermanTranslation"`
	EasyPeriod                int    `json:"easyPeriod"`      // Days
	NormalPeriod              int    `json:"normalPeriod"`    // Days
	HardPeriod                int    `json:"hardPeriod"`      // Days
	ExtraHardPeriod           int    `json:"extraHardPeriod"` // Days
}

// WordResponse represents a word in API responses with locale-specific translations
type WordResponse struct {
	ID                 int    `json:"id"`
	Word               string `json:"word"`
	PhoneticClues      string `json:"phoneticClues"`
	Example            string `json:"example"`
	Translation        string `json:"translation"`        // Locale-specific word translation
	ExampleTranslation string `json:"exampleTranslation"` // Locale-specific example translation
	EasyPeriod         int    `json:"easyPeriod"`
	NormalPeriod       int    `json:"normalPeriod"`
	HardPeriod         int    `json:"hardPeriod"`
	ExtraHardPeriod    int    `json:"extraHardPeriod"`
}

// WordResult represents a word learning result submission
type WordResult struct {
	WordID int `json:"wordId"`
	Period int `json:"period"` // Days (1-30)
}

// WordListItem represents a word in the list response
type WordListItem struct {
	ID                 int    `json:"id"`
	Word               string `json:"word"`
	PhoneticClues      string `json:"phoneticClues"`
	EnglishTranslation string `json:"englishTranslation"`
}

// CreateWordRequest represents a request to create a word
type CreateWordRequest struct {
	Word                      string `json:"word"`
	PhoneticClues             string `json:"phoneticClues"`
	RussianTranslation        string `json:"russianTranslation"`
	EnglishTranslation        string `json:"englishTranslation"`
	GermanTranslation         string `json:"germanTranslation"`
	Example                   string `json:"example"`
	ExampleRussianTranslation string `json:"exampleRussianTranslation"`
	ExampleEnglishTranslation string `json:"exampleEnglishTranslation"`
	ExampleGermanTranslation  string `json:"exampleGermanTranslation"`
	EasyPeriod                int    `json:"easyPeriod"`
	NormalPeriod              int    `json:"normalPeriod"`
	HardPeriod                int    `json:"hardPeriod"`
	ExtraHardPeriod           int    `json:"extraHardPeriod"`
}

// UpdateWordRequest represents a request to update a word (partial update)
type UpdateWordRequest struct {
	Word                      string `json:"word,omitempty"`
	PhoneticClues             string `json:"phoneticClues,omitempty"`
	RussianTranslation        string `json:"russianTranslation,omitempty"`
	EnglishTranslation        string `json:"englishTranslation,omitempty"`
	GermanTranslation         string `json:"germanTranslation,omitempty"`
	Example                   string `json:"example,omitempty"`
	ExampleRussianTranslation string `json:"exampleRussianTranslation,omitempty"`
	ExampleEnglishTranslation string `json:"exampleEnglishTranslation,omitempty"`
	ExampleGermanTranslation  string `json:"exampleGermanTranslation,omitempty"`
	EasyPeriod                *int   `json:"easyPeriod,omitempty"`
	NormalPeriod              *int   `json:"normalPeriod,omitempty"`
	HardPeriod                *int   `json:"hardPeriod,omitempty"`
	ExtraHardPeriod           *int   `json:"extraHardPeriod,omitempty"`
}
