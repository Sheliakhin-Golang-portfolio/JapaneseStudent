package models

// ComplexityLevel represents the complexity level of a course
type ComplexityLevel string

const (
	ComplexityLevelAbsoluteBeginner  ComplexityLevel = "Absolute beginner"
	ComplexityLevelBeginner          ComplexityLevel = "Beginner"
	ComplexityLevelIntermediate      ComplexityLevel = "Intermediate"
	ComplexityLevelUpperIntermediate ComplexityLevel = "Upper Intermediate"
	ComplexityLevelAdvanced          ComplexityLevel = "Advanced"
)

// ComplexityLevelAbbreviation maps abbreviations to full complexity levels
var ComplexityLevelAbbreviation = map[string]ComplexityLevel{
	"ab": ComplexityLevelAbsoluteBeginner,
	"b":  ComplexityLevelBeginner,
	"i":  ComplexityLevelIntermediate,
	"ui": ComplexityLevelUpperIntermediate,
	"a":  ComplexityLevelAdvanced,
}

// Course represents a course in the learning system
type Course struct {
	ID              int             `json:"id"`
	Slug            string          `json:"slug"`
	AuthorID        int             `json:"authorId"`
	Title           string          `json:"title"`
	ShortSummary    string          `json:"shortSummary"`
	ComplexityLevel ComplexityLevel `json:"complexityLevel"`
}

// CourseListItem represents a course in list responses
type CourseListItem struct {
	ID              int             `json:"id,omitempty"`
	Slug            string          `json:"slug"`
	Title           string          `json:"title"`
	ComplexityLevel ComplexityLevel `json:"complexityLevel"`
	AuthorID        int             `json:"authorId,omitempty"`
}

// CourseDetailResponse represents a course with additional details for user endpoints
type CourseDetailResponse struct {
	ID               int             `json:"id,omitempty"`
	Slug             string          `json:"slug,omitempty"`
	Title            string          `json:"title"`
	ShortSummary     string          `json:"shortSummary,omitempty"`
	ComplexityLevel  ComplexityLevel `json:"complexityLevel"`
	TotalLessons     int             `json:"totalLessons"`
	CompletedLessons int             `json:"completedLessons"`
}

// CreateCourseRequest represents a request to create a course
type CreateCourseRequest struct {
	AuthorID        int             `json:"authorId"`
	Slug            string          `json:"slug"`
	Title           string          `json:"title"`
	ShortSummary    string          `json:"shortSummary"`
	ComplexityLevel ComplexityLevel `json:"complexityLevel"`
}

// UpdateCourseRequest represents a request to update a course (partial update)
type UpdateCourseRequest struct {
	AuthorID        *int            `json:"authorId,omitempty"`
	Slug            string          `json:"slug,omitempty"`
	Title           string          `json:"title,omitempty"`
	ShortSummary    string          `json:"shortSummary,omitempty"`
	ComplexityLevel ComplexityLevel `json:"complexityLevel,omitempty"`
}

// CourseShortInfo represents a course with only ID and Title (for select options)
type CourseShortInfo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}
