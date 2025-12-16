package models

// CharacterLearnHistory represents a user's learning history for a character
//
// Result fields have their values in float because in the future they will have float values.
type CharacterLearnHistory struct {
	ID                      int     `json:"id"`
	UserID                  int     `json:"userId"`
	CharacterID             int     `json:"characterId"`
	HiraganaReadingResult   float32 `json:"hiraganaReadingResult"`   // 0 or 1
	HiraganaWritingResult   float32 `json:"hiraganaWritingResult"`   // 0 or 1
	HiraganaListeningResult float32 `json:"hiraganaListeningResult"` // 0 or 1
	KatakanaReadingResult   float32 `json:"katakanaReadingResult"`   // 0 or 1
	KatakanaWritingResult   float32 `json:"katakanaWritingResult"`   // 0 or 1
	KatakanaListeningResult float32 `json:"katakanaListeningResult"` // 0 or 1
}

// UserLearnHistory represents a user's learning history
type UserLearnHistory struct {
	CharacterHiragana       string  `json:"characterHiragana"`
	CharacterKatakana       string  `json:"characterKatakana"`
	HiraganaReadingResult   float32 `json:"hiraganaReadingResult"`
	HiraganaWritingResult   float32 `json:"hiraganaWritingResult"`
	HiraganaListeningResult float32 `json:"hiraganaListeningResult"`
	KatakanaReadingResult   float32 `json:"katakanaReadingResult"`
	KatakanaWritingResult   float32 `json:"katakanaWritingResult"`
	KatakanaListeningResult float32 `json:"katakanaListeningResult"`
}

// TestResultItem represents a single test result
type TestResultItem struct {
	CharacterID int  `json:"characterId"`
	Passed      bool `json:"passed"`
}
