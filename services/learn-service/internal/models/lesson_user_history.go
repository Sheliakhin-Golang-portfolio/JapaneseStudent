package models

// LessonUserHistory represents a user's completion history for a lesson
type LessonUserHistory struct {
	ID       int `json:"id"`
	UserID   int `json:"userId"`
	CourseID int `json:"courseId"`
	LessonID int `json:"lessonId"`
}

