package models

// MediaType represents the type of tutor media
type MediaType string

const (
	MediaTypeVideo MediaType = "video"
	MediaTypeDoc   MediaType = "doc"
	MediaTypeAudio MediaType = "audio"
)

// TutorMedia represents media content uploaded by a tutor
type TutorMedia struct {
	ID        int       `json:"id"`
	TutorID   int       `json:"tutorId"`
	Slug      string    `json:"slug"`
	MediaType MediaType `json:"mediaType"`
	URL       string    `json:"url"`
}

// TutorMediaResponse represents tutor media in API responses (without TutorID)
type TutorMediaResponse struct {
	ID        int       `json:"id"`
	Slug      string    `json:"slug"`
	MediaType MediaType `json:"mediaType"`
	URL       string    `json:"url"`
}

// CreateTutorMediaRequest represents a request to create tutor media
type CreateTutorMediaRequest struct {
	TutorID   int       `json:"tutorId"`
	Slug      string    `json:"slug"`
	MediaType MediaType `json:"mediaType"`
}

// TutorInfo represents tutor information (Id and Username)
type TutorInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}
