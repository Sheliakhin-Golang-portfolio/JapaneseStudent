package models

// Lesson represents a lesson in a course
type Lesson struct {
	ID           int    `json:"id"`
	Slug         string `json:"slug"`
	CourseID     int    `json:"courseId,omitempty"`
	Title        string `json:"title"`
	ShortSummary string `json:"shortSummary"`
	Order        int    `json:"order"`
}

// LessonListItem represents a lesson in user list responses
type LessonListItem struct {
	ID           int    `json:"id,omitempty"`
	Slug         string `json:"slug,omitempty"`
	CourseID     int    `json:"courseId,omitempty"`
	Title        string `json:"title"`
	ShortSummary string `json:"shortSummary,omitempty"`
	Order        int    `json:"order,omitempty"`
	Completed    bool   `json:"completed"`
}

// CreateLessonRequest represents a request to create a lesson
type CreateLessonRequest struct {
	Slug         string `json:"slug"`
	CourseID     int    `json:"courseId"`
	Title        string `json:"title"`
	ShortSummary string `json:"shortSummary"`
	Order        int    `json:"order"`
}

// UpdateLessonRequest represents a request to update a lesson (partial update)
type UpdateLessonRequest struct {
	Slug         string `json:"slug,omitempty"`
	CourseID     *int   `json:"courseId,omitempty"`
	Title        string `json:"title,omitempty"`
	ShortSummary string `json:"shortSummary,omitempty"`
	Order        *int   `json:"order,omitempty"`
}

// LessonShortInfo represents a lesson with only ID and Title (for select options)
type LessonShortInfo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}
