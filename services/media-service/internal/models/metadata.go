package models

// Metadata represents file metadata in the database
type Metadata struct {
	ID          string    `json:"id" db:"id"`
	ContentType string    `json:"contentType" db:"content_type"`
	Size        int64     `json:"size" db:"size"`
	URL         string    `json:"url" db:"url"`
	Type        MediaType `json:"type" db:"type"`
}

// MediaType represents valid media types
type MediaType string

const (
	MediaTypeCharacter   MediaType = "character"
	MediaTypeWord        MediaType = "word"
	MediaTypeWordExample MediaType = "word_example"
	MediaTypeLessonAudio MediaType = "lesson_audio"
	MediaTypeLessonVideo MediaType = "lesson_video"
	MediaTypeLessonDoc   MediaType = "lesson_doc"
)
