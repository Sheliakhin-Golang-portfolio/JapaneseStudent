package services

import (
	"context"
	"fmt"

	"github.com/japanesestudent/learn-service/internal/models"
)

// CourseRepository defines methods for course data access
type CourseRepository interface {
	// GetBySlug retrieves a course by slug
	//
	// "ctx" is the context for the request.
	// "slug" is the slug of the course.
	// "userID" is the ID of the user.
	//
	// Returns the course and an error if any.
	GetBySlug(ctx context.Context, slug string, userID int) (*models.CourseDetailResponse, error)
	// GetAll retrieves a list of courses with filtering and pagination
	//
	// "ctx" is the context for the request.
	// "userID" is the ID of the user.
	// "complexityLevel" is the complexity level of the courses to retrieve.
	// "search" is the search query for the courses.
	// "isMine" is a flag to filter courses by user's completion history.
	// "page" is the page number to retrieve.
	// "count" is the number of items per page.
	//
	// Returns a list of courses and an error if any.
	GetAll(ctx context.Context, userID int, complexityLevel *models.ComplexityLevel, search string, isMine bool, page, count int) ([]models.CourseDetailResponse, error)
}

// LessonRepository defines methods for lesson data access
type LessonRepository interface {
	// GetBySlug retrieves a lesson by slug
	//
	// "ctx" is the context for the request.
	// "slug" is the slug of the lesson.
	// "userID" is the ID of the user.
	//
	// Returns the lesson and an error if any.
	GetBySlug(ctx context.Context, slug string, userID int) (*models.LessonListItem, error)
	// GetByCourseIDWithCompletion retrieves a list of lessons with completion status for a course
	//
	// "ctx" is the context for the request.
	// "courseID" is the ID of the course.
	// "userID" is the ID of the user.
	//
	// Returns a list of lessons and an error if any.
	GetByCourseIDWithCompletion(ctx context.Context, courseID, userID int) ([]models.LessonListItem, error)
}

// LessonBlockRepository defines methods for lesson block data access
type LessonBlockRepository interface {
	// GetByLessonID retrieves a list of lesson blocks by lesson ID
	//
	// "ctx" is the context for the request.
	// "lessonID" is the ID of the lesson.
	//
	// Returns a list of lesson blocks and an error if any.
	GetByLessonID(ctx context.Context, lessonID int) ([]models.LessonBlockResponse, error)
}

// LessonUserHistoryRepository defines methods for lesson user history data access
type LessonUserHistoryRepository interface {
	// Exists checks if a lesson user history record exists
	//
	// "ctx" is the context for the request.
	// "userID" is the ID of the user.
	// "courseID" is the ID of the course.
	// "lessonID" is the ID of the lesson.
	//
	// Returns a boolean and an error if any.
	Exists(ctx context.Context, userID, courseID, lessonID int) (bool, error)
	// Create creates a new lesson user history record
	//
	// "ctx" is the context for the request.
	// "history" is the lesson user history record to create.
	//
	// Returns an error if any.
	Create(ctx context.Context, history *models.LessonUserHistory) error
	// Delete deletes a lesson user history record
	//
	// "ctx" is the context for the request.
	// "userID" is the ID of the user.
	// "courseID" is the ID of the course.
	// "lessonID" is the ID of the lesson.
	//
	// Returns an error if any.
	Delete(ctx context.Context, userID, courseID, lessonID int) error
	// CountCompletedLessonsByCourse counts the number of completed lessons in a course
	//
	// "ctx" is the context for the request.
	// "userID" is the ID of the user.
	// "courseID" is the ID of the course.
	//
	// Returns the number of completed lessons and an error if any.
	CountCompletedLessonsByCourse(ctx context.Context, userID, courseID int) (int, error)
}

type userLessonService struct {
	courseRepo  CourseRepository
	lessonRepo  LessonRepository
	blockRepo   LessonBlockRepository
	historyRepo LessonUserHistoryRepository
}

// NewUserLessonService creates a new user lesson service
func NewUserLessonService(
	courseRepo CourseRepository,
	lessonRepo LessonRepository,
	blockRepo LessonBlockRepository,
	historyRepo LessonUserHistoryRepository,
) *userLessonService {
	return &userLessonService{
		courseRepo:  courseRepo,
		lessonRepo:  lessonRepo,
		blockRepo:   blockRepo,
		historyRepo: historyRepo,
	}
}

// GetCoursesList retrieves a list of courses with filtering and pagination
func (s *userLessonService) GetCoursesList(ctx context.Context, userID int, complexityLevel *models.ComplexityLevel, search string, isMine bool, page, count int) ([]models.CourseDetailResponse, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 10
	}

	return s.courseRepo.GetAll(ctx, userID, complexityLevel, search, isMine, page, count)
}

// GetLessonsInCourse retrieves course details with lesson list and completion status
func (s *userLessonService) GetLessonsInCourse(ctx context.Context, courseSlug string, userID int) (*models.CourseDetailResponse, []models.LessonListItem, error) {
	// Get course by slug
	course, err := s.courseRepo.GetBySlug(ctx, courseSlug, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get course: %w", err)
	}

	// Get lessons with completion status
	lessons, err := s.lessonRepo.GetByCourseIDWithCompletion(ctx, course.ID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get lessons: %w", err)
	}

	course.ID = 0
	return course, lessons, nil
}

// GetLesson retrieves a full lesson with blocks and completion status
func (s *userLessonService) GetLesson(ctx context.Context, lessonSlug string, userID int) (*models.LessonListItem, []models.LessonBlockResponse, error) {
	// Get lesson by slug
	lesson, err := s.lessonRepo.GetBySlug(ctx, lessonSlug, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get lesson: %w", err)
	}

	// Get lesson blocks
	blocks, err := s.blockRepo.GetByLessonID(ctx, lesson.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get lesson blocks: %w", err)
	}

	lesson.CourseID = 0 // Clear course ID to avoid leaking course information
	lesson.ID = 0       // Clear lesson ID to avoid leaking lesson information
	return lesson, blocks, nil
}

// ToggleLessonCompletion toggles lesson completion status
func (s *userLessonService) ToggleLessonCompletion(ctx context.Context, lessonSlug string, userID int) error {
	// Get lesson by slug
	lesson, err := s.lessonRepo.GetBySlug(ctx, lessonSlug, userID)
	if err != nil {
		return fmt.Errorf("failed to get lesson: %w", err)
	}

	// Check if history record exists
	exists, err := s.historyRepo.Exists(ctx, userID, lesson.CourseID, lesson.ID)
	if err != nil {
		return fmt.Errorf("failed to check history existence: %w", err)
	}

	if exists {
		// Delete history record (uncomplete)
		err = s.historyRepo.Delete(ctx, userID, lesson.CourseID, lesson.ID)
		if err != nil {
			return fmt.Errorf("failed to delete history record: %w", err)
		}
	} else {
		// Create history record (complete)
		history := &models.LessonUserHistory{
			UserID:   userID,
			CourseID: lesson.CourseID,
			LessonID: lesson.ID,
		}
		err = s.historyRepo.Create(ctx, history)
		if err != nil {
			return fmt.Errorf("failed to create history record: %w", err)
		}
	}

	return nil
}
