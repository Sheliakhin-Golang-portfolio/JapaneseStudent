package models

import "time"

// DictionaryHistory represents a user's learning history for a word
type DictionaryHistory struct {
	ID             int       `json:"id"`
	WordID         int       `json:"wordId"`
	UserID         int       `json:"userId"`
	NextAppearance time.Time `json:"nextAppearance"`
}
