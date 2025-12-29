package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"slices"

	"github.com/japanesestudent/learn-service/internal/models"
)

// TutorCourseRepository defines methods for course data access for tutors
type TutorCourseRepository interface {
	// GetByID retrieves a course by ID
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the course.
	//
	// Returns the course and an error if any.
	GetByID(ctx context.Context, id int) (*models.Course, error)
	// GetByAuthorOrFull retrieves courses by author ID or full list with filtering and pagination
	//
	// "ctx" is the context for the request.
	// "authorID" is the ID of the author or nil for full list.
	// "complexityLevel" is the complexity level of the courses to retrieve.
	// "search" is the search query for the courses.
	// "page" is the page number to retrieve.
	// "count" is the number of items per page.
	//
	// Returns a list of courses and an error if any.
	GetByAuthorOrFull(ctx context.Context, authorID *int, complexityLevel *models.ComplexityLevel, search string, page, count int) ([]models.CourseListItem, error)
	// GetShortInfo retrieves short information about courses by author ID
	//
	// "ctx" is the context for the request.
	// "authorID" is the ID of the author (optional, if nil, the courses are being retrieved by an admin).
	//
	// Returns a list of course short information and an error if any.
	GetShortInfo(ctx context.Context, authorID *int) ([]models.CourseShortInfo, error)
	// ExistsBySlug checks if a course with the given slug exists
	//
	// "ctx" is the context for the request.
	// "slug" is the slug of the course.
	//
	// Returns a boolean and an error if any.
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	// ExistsByTitle checks if a course with the given title exists
	//
	// "ctx" is the context for the request.
	// "title" is the title of the course.
	//
	// Returns a boolean and an error if any.
	ExistsByTitle(ctx context.Context, title string) (bool, error)
	// Create creates a new course
	//
	// "ctx" is the context for the request.
	// "course" is the course to create.
	//
	// Returns an error if any.
	Create(ctx context.Context, course *models.Course) error
	// Update updates a course
	//
	// "ctx" is the context for the request.
	// "course" is the course to update.
	//
	// Returns an error if any.
	Update(ctx context.Context, course *models.Course) error
	// Delete deletes a course
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the course.
	//
	// Returns an error if any.
	Delete(ctx context.Context, id int) error
	// CheckOwnership checks if a course belongs to a tutor
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the course.
	// "tutorID" is the ID of the tutor.
	//
	// Returns a boolean and an error if any.
	CheckOwnership(ctx context.Context, id, tutorID int) (bool, error)
}

// TutorLessonRepository defines methods for lesson data access for tutors
type TutorLessonRepository interface {
	// GetByID retrieves a lesson by ID
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the lesson.
	//
	// Returns the lesson and an error if any.
	GetByID(ctx context.Context, id int) (*models.Lesson, error)
	// GetByCourseID retrieves lessons by course ID
	//
	// "ctx" is the context for the request.
	// "courseID" is the ID of the course.
	//
	// Returns a list of lessons and an error if any.
	GetByCourseID(ctx context.Context, courseID int) ([]models.Lesson, error)
	// GetShortInfoByCourseID retrieves short information about lessons by course ID
	//
	// "courseID" is the ID of the course (optional, if nil, the lessons are being retrieved by an admin).
	//
	// Returns a list of lesson short information and an error if any.
	GetShortInfoByCourseID(ctx context.Context, courseID *int) ([]models.LessonShortInfo, error)
	// ExistsBySlug checks if a lesson with the given slug exists
	//
	// "ctx" is the context for the request.
	// "slug" is the slug of the lesson.
	//
	// Returns a boolean and an error if any.
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	// ExistsByTitleInCourse checks if a lesson with the given title exists in a course
	//
	// "ctx" is the context for the request.
	// "courseID" is the ID of the course.
	// "title" is the title of the lesson.
	//
	// Returns a boolean and an error if any.
	ExistsByTitleInCourse(ctx context.Context, courseID int, title string) (bool, error)
	// ExistsByOrderInCourse checks if a lesson with the given order exists in a course
	//
	// "ctx" is the context for the request.
	// "courseID" is the ID of the course.
	// "order" is the order of the lesson.
	//
	// Returns a boolean and an error if any.
	ExistsByOrderInCourse(ctx context.Context, courseID int, order int) (bool, error)
	// IncrementOrderForLessons increments the order of lessons in a course
	//
	// "ctx" is the context for the request.
	// "courseID" is the ID of the course.
	// "order" is the order of the lesson.
	//
	// Returns an error if any.
	IncrementOrderForLessons(ctx context.Context, courseID, order int) error
	// Create creates a new lesson
	//
	// "ctx" is the context for the request.
	// "lesson" is the lesson to create.
	//
	// Returns an error if any.
	Create(ctx context.Context, lesson *models.Lesson) error
	// Update updates a lesson
	//
	// "ctx" is the context for the request.
	// "lesson" is the lesson to update.
	//
	// Returns an error if any.
	Update(ctx context.Context, lesson *models.Lesson) error
	// Delete deletes a lesson
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the lesson.
	//
	// Returns an error if any.
	Delete(ctx context.Context, id int) error
	// CheckOwnership checks if a lesson belongs to a tutor
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the lesson.
	// "tutorID" is the ID of the tutor.
	//
	// Returns a boolean and an error if any.
	CheckOwnership(ctx context.Context, id, tutorID int) (bool, error)
}

// TutorLessonBlockRepository defines methods for lesson block data access for tutors
type TutorLessonBlockRepository interface {
	// GetByID retrieves a lesson block by ID
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the lesson block.
	//
	// Returns the lesson block and an error if any.
	GetByID(ctx context.Context, id int) (*models.LessonBlock, error)
	// GetByLessonID retrieves lesson blocks by lesson ID
	//
	// "ctx" is the context for the request.
	// "lessonID" is the ID of the lesson.
	//
	// Returns a list of lesson blocks and an error if any.
	GetByLessonID(ctx context.Context, lessonID int) ([]models.LessonBlockResponse, error)
	// ExistsByOrderInLesson checks if a lesson block with the given order exists in a lesson
	//
	// "ctx" is the context for the request.
	// "lessonID" is the ID of the lesson.
	// "order" is the order of the lesson block.
	//
	// Returns a boolean and an error if any.
	ExistsByOrderInLesson(ctx context.Context, lessonID int, order int) (bool, error)
	// IncrementOrderForBlocks increments the order of lesson blocks in a lesson
	//
	// "ctx" is the context for the request.
	// "lessonID" is the ID of the lesson.
	// "order" is the order of the lesson block.
	//
	// Returns an error if any.
	IncrementOrderForBlocks(ctx context.Context, lessonID, order int) error
	// Create creates a new lesson block
	//
	// "ctx" is the context for the request.
	// "block" is the lesson block to create.
	//
	// Returns an error if any.
	Create(ctx context.Context, block *models.LessonBlock) error
	// Update updates a lesson block
	//
	// "ctx" is the context for the request.
	// "block" is the lesson block to update.
	//
	// Returns an error if any.
	Update(ctx context.Context, block *models.LessonBlock) error
	// Delete deletes a lesson block
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the lesson block.
	//
	// Returns an error if any.
	Delete(ctx context.Context, id int) error
	// CheckOwnership checks if a lesson block belongs to a tutor
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the lesson block.
	// "tutorID" is the ID of the tutor.
	//
	// Returns a boolean and an error if any.
	CheckOwnership(ctx context.Context, id, tutorID int) (bool, error)
}

// TutorMediaRepository defines methods for tutor media data access
type TutorMediaRepository interface {
	// GetByID retrieves a tutor media by ID
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the tutor media.
	//
	// Returns the tutor media and an error if any.
	GetByID(ctx context.Context, id int) (*models.TutorMedia, error)
	// GetByTutorID retrieves tutor media by tutor ID with filtering and pagination
	//
	// "ctx" is the context for the request.
	// "tutorID" is the ID of the tutor (optional filtering).
	// "mediaType" is the type of the media.
	// "page" is the page number to retrieve.
	// "count" is the number of items per page.
	//
	// Returns a list of tutor media and an error if any.
	GetByTutorID(ctx context.Context, tutorID *int, mediaType *models.MediaType, page, count int) ([]models.TutorMediaResponse, error)
	// ExistsBySlug checks if a tutor media with the given slug exists
	//
	// "ctx" is the context for the request.
	// "slug" is the slug of the tutor media.
	//
	// Returns a boolean and an error if any.
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	// Create creates a new tutor media
	//
	// "ctx" is the context for the request.
	// "media" is the tutor media to create.
	//
	// Returns an error if any.
	Create(ctx context.Context, media *models.TutorMedia) error
	// Delete deletes a tutor media
	//
	// "ctx" is the context for the request.
	// "id" is the ID of the tutor media.
	//
	// Returns an error if any.
	Delete(ctx context.Context, id int) error
}

type tutorLessonService struct {
	courseRepo   TutorCourseRepository
	lessonRepo   TutorLessonRepository
	blockRepo    TutorLessonBlockRepository
	mediaRepo    TutorMediaRepository
	mediaBaseURL string
	apiKey       string
}

// NewTutorLessonService creates a new tutor lesson service
func NewTutorLessonService(
	courseRepo TutorCourseRepository,
	lessonRepo TutorLessonRepository,
	blockRepo TutorLessonBlockRepository,
	mediaRepo TutorMediaRepository,
	mediaBaseURL, apiKey string,
) *tutorLessonService {
	return &tutorLessonService{
		courseRepo:   courseRepo,
		lessonRepo:   lessonRepo,
		blockRepo:    blockRepo,
		mediaRepo:    mediaRepo,
		mediaBaseURL: mediaBaseURL,
		apiKey:       apiKey,
	}
}

// GetCourses retrieves courses authored by tutor or all courses with filtering and pagination
func (s *tutorLessonService) GetCourses(ctx context.Context, tutorID *int, complexityLevel *models.ComplexityLevel, search string, page, count int) ([]models.CourseListItem, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 10
	}

	return s.courseRepo.GetByAuthorOrFull(ctx, tutorID, complexityLevel, search, page, count)
}

// CreateCourse creates a new course
func (s *tutorLessonService) CreateCourse(ctx context.Context, req *models.CreateCourseRequest) (int, error) {
	// Validate course creation request
	if err := s.validateCreateCourse(ctx, req); err != nil {
		return 0, err
	}

	course := &models.Course{
		Slug:            req.Slug,
		AuthorID:        req.AuthorID,
		Title:           req.Title,
		ShortSummary:    req.ShortSummary,
		ComplexityLevel: req.ComplexityLevel,
	}

	err := s.courseRepo.Create(ctx, course)
	if err != nil {
		return 0, err
	}

	return course.ID, nil
}

func (s *tutorLessonService) validateCreateCourse(ctx context.Context, req *models.CreateCourseRequest) error {
	// Validate all fields are provided
	if req.Slug == "" || req.Title == "" || req.ShortSummary == "" || req.ComplexityLevel == "" {
		return fmt.Errorf("all fields are required")
	}

	// Prepare for concurrent check
	errorChan := make(chan error, 3)

	// Validate complexity level
	go func() {
		if !s.isValidComplexityLevel(req.ComplexityLevel) {
			errorChan <- fmt.Errorf("invalid complexity level")
			return
		}
		errorChan <- nil
	}()
	// Check slug uniqueness
	go func() {
		exists, err := s.courseRepo.ExistsBySlug(ctx, req.Slug)
		if err != nil {
			errorChan <- err
			return
		}
		if exists {
			errorChan <- fmt.Errorf("course with slug '%s' already exists", req.Slug)
			return
		}
		errorChan <- nil
	}()
	// Check title uniqueness
	go func() {
		exists, err := s.courseRepo.ExistsByTitle(ctx, req.Title)
		if err != nil {
			errorChan <- err
			return
		}
		if exists {
			errorChan <- fmt.Errorf("course with title '%s' already exists", req.Title)
			return
		}
		errorChan <- nil
	}()

	for range 3 {
		err := <-errorChan
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateCourse updates a course (partial update)
func (s *tutorLessonService) UpdateCourse(ctx context.Context, courseID int, tutorID *int, req *models.UpdateCourseRequest) error {
	// Get course to check ownership
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return fmt.Errorf("course not found")
	}

	// Check ownership (if tutorID is not nil, it means that the course is being updated by a tutor)
	if tutorID != nil && course.AuthorID != *tutorID {
		return fmt.Errorf("you do not have rights to manage this course")
	}

	// Validate the update course request
	if err := s.validateUpdateCourse(ctx, course, req, tutorID); err != nil {
		return err
	}

	updateCourse := &models.Course{
		ID: courseID,
	}
	if req.Slug != course.Slug {
		updateCourse.Slug = req.Slug
	}
	if req.Title != course.Title {
		updateCourse.Title = req.Title
	}
	if req.ShortSummary != course.ShortSummary {
		updateCourse.ShortSummary = req.ShortSummary
	}
	if req.ComplexityLevel != course.ComplexityLevel {
		updateCourse.ComplexityLevel = req.ComplexityLevel
	}
	// tutorID is nil, it means that the course is being updated by an admin
	if req.AuthorID != nil && tutorID == nil {
		updateCourse.AuthorID = *req.AuthorID
	}

	return s.courseRepo.Update(ctx, updateCourse)
}

// validateUpdateCourse validates the update course request
func (s *tutorLessonService) validateUpdateCourse(ctx context.Context, course *models.Course, req *models.UpdateCourseRequest, tutorID *int) error {
	// Normalize authorID by assigning it to nil if tutorID is not nil
	if tutorID != nil {
		req.AuthorID = nil
	}

	// Validate if any field is provided
	if req.Slug == "" && req.Title == "" && req.ShortSummary == "" && req.ComplexityLevel == "" && req.AuthorID == nil {
		return fmt.Errorf("at least one field must be provided")
	}

	// Prepare for concurrent check
	errorChan := make(chan error, 3)

	// Check slug uniqueness if provided
	go func() {
		if req.Slug != "" && req.Slug != course.Slug {
			exists, err := s.courseRepo.ExistsBySlug(ctx, req.Slug)
			if err != nil {
				errorChan <- err
				return
			}
			if exists {
				errorChan <- fmt.Errorf("course with slug '%s' already exists", req.Slug)
				return
			}
		}
		errorChan <- nil
	}()

	// Check title uniqueness if provided
	go func() {
		if req.Title != "" && req.Title != course.Title {
			exists, err := s.courseRepo.ExistsByTitle(ctx, req.Title)
			if err != nil {
				errorChan <- err
				return
			}
			if exists {
				errorChan <- fmt.Errorf("course with title '%s' already exists", req.Title)
				return
			}
		}
		errorChan <- nil
	}()

	// Check complexity level if provided
	go func() {
		if req.ComplexityLevel != "" && req.ComplexityLevel != course.ComplexityLevel {
			if !s.isValidComplexityLevel(req.ComplexityLevel) {
				errorChan <- fmt.Errorf("invalid complexity level")
				return
			}
		}
		errorChan <- nil
	}()

	for range 3 {
		err := <-errorChan
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteCourse deletes a course
func (s *tutorLessonService) DeleteCourse(ctx context.Context, courseID int, tutorID *int) error {
	// Get course to check ownership
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return fmt.Errorf("course not found")
	}

	// Check ownership (if tutorID is not nil, it means that the course is being deleted by a tutor)
	if tutorID != nil && course.AuthorID != *tutorID {
		return fmt.Errorf("you do not have rights to manage this course")
	}

	return s.courseRepo.Delete(ctx, courseID)
}

// GetCoursesShortInfo retrieves courses with only ID and Title
func (s *tutorLessonService) GetCoursesShortInfo(ctx context.Context, tutorID *int) ([]models.CourseShortInfo, error) {
	return s.courseRepo.GetShortInfo(ctx, tutorID)
}

// GetLessonsForCourse retrieves course with lessons list
//
// If tutorID is not nil, it will check if the course belongs to the tutor.
func (s *tutorLessonService) GetLessonsForCourse(ctx context.Context, courseID int, tutorID *int) (*models.Course, []models.Lesson, error) {
	// Get course to check ownership
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return nil, nil, fmt.Errorf("course not found")
	}

	// Check ownership (if tutorID is not nil, it means that the course is being retrieved by a tutor)
	if tutorID != nil && course.AuthorID != *tutorID {
		return nil, nil, fmt.Errorf("you do not have rights to manage this course")
	}

	// Get lessons
	lessons, err := s.lessonRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		return nil, nil, err
	}

	// If tutorID is not nil, it means that the course is being retrieved by a tutor, so we already know the tutor
	if tutorID != nil {
		course.AuthorID = 0
	}

	return course, lessons, nil
}

// CreateLesson creates a new lesson
func (s *tutorLessonService) CreateLesson(ctx context.Context, tutorID *int, req *models.CreateLessonRequest) (int, error) {
	if err := s.validateCreateLesson(ctx, tutorID, req); err != nil {
		return 0, err
	}

	// Handle order conflicts
	exists, err := s.lessonRepo.ExistsByOrderInCourse(ctx, req.CourseID, req.Order)
	if err != nil {
		return 0, err
	}
	if exists {
		// Increment order for all lessons with order >= new order
		// We do it to insert new lesson without breaking the order
		err = s.lessonRepo.IncrementOrderForLessons(ctx, req.CourseID, req.Order)
		if err != nil {
			return 0, err
		}
	}

	lesson := &models.Lesson{
		Slug:         req.Slug,
		CourseID:     req.CourseID,
		Title:        req.Title,
		ShortSummary: req.ShortSummary,
		Order:        req.Order,
	}

	err = s.lessonRepo.Create(ctx, lesson)
	if err != nil {
		return 0, fmt.Errorf("failed to create lesson: %w", err)
	}

	return lesson.ID, nil
}

func (s *tutorLessonService) validateCreateLesson(ctx context.Context, tutorID *int, req *models.CreateLessonRequest) error {
	// Validate all fields are provided
	if req.Slug == "" || req.CourseID == 0 || req.Title == "" || req.ShortSummary == "" || req.Order <= 0 {
		return fmt.Errorf("all fields are required and order must be greater than 0")
	}

	// Prepare for concurrent check
	errorChan := make(chan error, 3)

	// Check if course exists and belongs to tutor
	go func() {
		if tutorID != nil {
			exists, err := s.courseRepo.CheckOwnership(ctx, req.CourseID, *tutorID)
			if err != nil {
				errorChan <- err
				return
			}
			if !exists {
				errorChan <- fmt.Errorf("course does not belong to you")
				return
			}
		}
		errorChan <- nil
	}()

	// Check slug uniqueness
	go func() {
		exists, err := s.lessonRepo.ExistsBySlug(ctx, req.Slug)
		if err != nil {
			errorChan <- err
			return
		}
		if exists {
			errorChan <- fmt.Errorf("lesson with slug '%s' already exists", req.Slug)
			return
		}
		errorChan <- nil
	}()

	// Check title uniqueness within course
	go func() {
		exists, err := s.lessonRepo.ExistsByTitleInCourse(ctx, req.CourseID, req.Title)
		if err != nil {
			errorChan <- err
			return
		}
		if exists {
			errorChan <- fmt.Errorf("lesson with title '%s' already exists in this course", req.Title)
			return
		}
		errorChan <- nil
	}()

	for range 3 {
		err := <-errorChan
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateLesson updates a lesson (partial update)
func (s *tutorLessonService) UpdateLesson(ctx context.Context, lessonID int, tutorID *int, req *models.UpdateLessonRequest) error {
	// Get lesson for fields comparison and check ownership
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return err
	}

	// Check ownership (if tutorID is not nil, it means that the lesson is being updated by a tutor)
	if tutorID != nil {
		exists, err := s.courseRepo.CheckOwnership(ctx, lesson.CourseID, *tutorID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("you do not have rights to manage this lesson")
		}
	}

	// Check if new course exists and belongs to tutor (if tutorID is not nil, it means that the lesson is being updated by a tutor)
	if req.CourseID != nil && tutorID != nil {
		exists, err := s.courseRepo.CheckOwnership(ctx, *req.CourseID, *tutorID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("you do not have rights to manage this course")
		}
	}

	// Validate lesson update request
	courseIDToCheck := lesson.CourseID
	if req.CourseID != nil && *req.CourseID != lesson.CourseID {
		courseIDToCheck = *req.CourseID
	}
	if err = s.validateUpdateLesson(ctx, lesson, req, courseIDToCheck); err != nil {
		return err
	}

	// Handle order conflicts if order is provided
	if req.Order != nil && *req.Order > 0 {
		exists, err := s.lessonRepo.ExistsByOrderInCourse(ctx, courseIDToCheck, *req.Order)
		if err != nil {
			return err
		}
		if exists {
			// Increment order for all lessons with order >= new order
			err = s.lessonRepo.IncrementOrderForLessons(ctx, courseIDToCheck, *req.Order)
			if err != nil {
				return err
			}
		}
	}

	updateLesson := &models.Lesson{
		ID:           lessonID,
		Slug:         req.Slug,
		Title:        req.Title,
		ShortSummary: req.ShortSummary,
	}
	if courseIDToCheck != lesson.CourseID {
		updateLesson.CourseID = courseIDToCheck
	}
	if req.Order != nil {
		updateLesson.Order = *req.Order
	}

	return s.lessonRepo.Update(ctx, updateLesson)
}

// validateUpdateLesson validates the update lesson request
func (s *tutorLessonService) validateUpdateLesson(ctx context.Context, lesson *models.Lesson, req *models.UpdateLessonRequest, courseID int) error {
	// Validate if any field is provided
	if req.Slug == "" && req.Title == "" && req.ShortSummary == "" && req.Order == nil && req.CourseID == nil {
		return fmt.Errorf("at least one field must be provided")
	}

	// Prepare for concurrent check
	errorChan := make(chan error, 3)

	// Check slug uniqueness if provided
	go func() {
		if req.Slug != "" && req.Slug != lesson.Slug {
			exists, err := s.lessonRepo.ExistsBySlug(ctx, req.Slug)
			if err != nil {
				errorChan <- err
				return
			}
			if exists {
				errorChan <- fmt.Errorf("lesson with slug '%s' already exists", req.Slug)
				return
			}
		}
		errorChan <- nil
	}()

	// Check title uniqueness if provided
	go func() {
		if req.Title != "" && req.Title != lesson.Title {
			exists, err := s.lessonRepo.ExistsByTitleInCourse(ctx, courseID, req.Title)
			if err != nil {
				errorChan <- err
				return
			}
			if exists {
				errorChan <- fmt.Errorf("lesson with title '%s' already exists in this course", req.Title)
				return
			}
		}
		errorChan <- nil
	}()

	// Check order validity if order provided
	go func() {
		if req.Order != nil && *req.Order <= 0 {
			errorChan <- fmt.Errorf("order must be greater than 0")
			return
		}
		errorChan <- nil
	}()

	for range 3 {
		err := <-errorChan
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteLesson deletes a lesson
func (s *tutorLessonService) DeleteLesson(ctx context.Context, lessonID int, tutorID *int) error {
	// Check ownership (if tutorID is not nil, it means that the lesson is being deleted by a tutor)
	if tutorID != nil {
		exists, err := s.lessonRepo.CheckOwnership(ctx, lessonID, *tutorID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("you do not have rights to manage this lesson")
		}
	}

	return s.lessonRepo.Delete(ctx, lessonID)
}

// GetFullLessonInfo retrieves lesson with all blocks
//
// If tutorID is not nil, it will check if the lesson belongs to the tutor.
func (s *tutorLessonService) GetFullLessonInfo(ctx context.Context, lessonID int, tutorID *int) (*models.Lesson, []models.LessonBlockResponse, error) {
	// Get lesson to check ownership
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, nil, fmt.Errorf("lesson not found")
	}

	// Check ownership (if tutorID is not nil, it means that the lesson is being retrieved by a tutor)
	if tutorID != nil {
		exists, err := s.courseRepo.CheckOwnership(ctx, lesson.CourseID, *tutorID)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			return nil, nil, fmt.Errorf("you do not have rights to manage this lesson")
		}
	}

	// Get blocks
	blocks, err := s.blockRepo.GetByLessonID(ctx, lessonID)
	if err != nil {
		return nil, nil, err
	}

	// If tutorID is not nil, it means that the lesson is being retrieved by a tutor, so we already know the course
	if tutorID != nil {
		lesson.CourseID = 0
	}

	return lesson, blocks, nil
}

// GetLessonsShortInfo retrieves lessons with only ID and Title for a course
//
// If tutorID is not nil, it will check if the course belongs to the tutor.
func (s *tutorLessonService) GetLessonsShortInfo(ctx context.Context, courseID, tutorID *int) ([]models.LessonShortInfo, error) {
	// Check ownership (if tutorID is not nil, it means that the lessons are being retrieved by a tutor)
	if tutorID != nil {
		exists, err := s.courseRepo.CheckOwnership(ctx, *courseID, *tutorID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("you do not have rights to manage this course")
		}
	}

	return s.lessonRepo.GetShortInfoByCourseID(ctx, courseID)
}

// CreateLessonBlock creates a new lesson block
//
// If tutorID is not nil, it will check if the lesson block belongs to the tutor.
func (s *tutorLessonService) CreateLessonBlock(ctx context.Context, tutorID *int, req *models.CreateLessonBlockRequest) (int, error) {
	// Validate all fields are provided
	if req.LessonID == 0 || req.BlockType == "" || req.BlockOrder <= 0 || req.BlockData == nil {
		return 0, fmt.Errorf("all fields are required and block order must be greater than 0")
	}

	// Validate block type
	if !s.isValidBlockType(req.BlockType) {
		return 0, fmt.Errorf("invalid block type")
	}

	// Check ownership (if tutorID is not nil, it means that the lesson block is being created by a tutor)
	if tutorID != nil {
		exists, err := s.lessonRepo.CheckOwnership(ctx, req.LessonID, *tutorID)
		if err != nil {
			return 0, err
		}
		if !exists {
			return 0, fmt.Errorf("you do not have rights to manage this lesson")
		}
	}

	// Handle order conflicts
	exists, err := s.blockRepo.ExistsByOrderInLesson(ctx, req.LessonID, req.BlockOrder)
	if err != nil {
		return 0, err
	}
	if exists {
		// Increment order for all blocks with order >= new order
		err = s.blockRepo.IncrementOrderForBlocks(ctx, req.LessonID, req.BlockOrder)
		if err != nil {
			return 0, err
		}
	}

	block := &models.LessonBlock{
		LessonID:   req.LessonID,
		BlockType:  req.BlockType,
		BlockOrder: req.BlockOrder,
		BlockData:  req.BlockData,
	}

	err = s.blockRepo.Create(ctx, block)
	if err != nil {
		return 0, fmt.Errorf("failed to create lesson block: %w", err)
	}

	return block.ID, nil
}

// UpdateLessonBlock updates a lesson block (partial update)
//
// If tutorID is not nil, it will check if the lesson block belongs to the tutor.
func (s *tutorLessonService) UpdateLessonBlock(ctx context.Context, blockID int, tutorID *int, req *models.UpdateLessonBlockRequest) error {
	// Validate if any field is provided
	if req.LessonID == nil && req.BlockType == "" && req.BlockOrder == nil && req.BlockData == nil {
		return fmt.Errorf("at least one field must be provided")
	}

	// Get block to check ownership
	block, err := s.blockRepo.GetByID(ctx, blockID)
	if err != nil {
		return fmt.Errorf("lesson block not found")
	}

	// Check ownership (if tutorID is not nil, it means that the lesson block is being updated by a tutor)
	if tutorID != nil {
		exists, err := s.lessonRepo.CheckOwnership(ctx, block.LessonID, *tutorID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("you do not have rights to manage this block")
		}
	}

	// Check if new lesson exists and belongs to tutor (if tutorID is not nil, it means that the lesson block is being updated by a tutor)
	if req.LessonID != nil && tutorID != nil {
		exists, err := s.lessonRepo.CheckOwnership(ctx, *req.LessonID, *tutorID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("you do not have rights to manage this lesson")
		}
	}

	// Determine which lesson to use for validation (might change if LessonID is updated)
	lessonIDToCheck := block.LessonID
	if req.LessonID != nil && *req.LessonID != block.LessonID {
		lessonIDToCheck = *req.LessonID
	}

	// Validate block type if provided
	if req.BlockType != "" && !s.isValidBlockType(req.BlockType) {
		return fmt.Errorf("invalid block type")
	}

	// Handle order conflicts if order is provided
	if req.BlockOrder != nil && *req.BlockOrder > 0 && *req.BlockOrder != block.BlockOrder {
		exists, err := s.blockRepo.ExistsByOrderInLesson(ctx, lessonIDToCheck, *req.BlockOrder)
		if err != nil {
			return err
		}
		if exists {
			// Increment order for all blocks with order >= new order
			err = s.blockRepo.IncrementOrderForBlocks(ctx, lessonIDToCheck, *req.BlockOrder)
			if err != nil {
				return err
			}
		}
	} else if req.BlockOrder != nil && *req.BlockOrder <= 0 {
		return fmt.Errorf("block order must be greater than 0")
	}

	updateBlock := &models.LessonBlock{
		ID:        blockID,
		BlockType: block.BlockType,
		BlockData: block.BlockData,
	}
	if req.LessonID != nil {
		updateBlock.LessonID = *req.LessonID
	}
	if req.BlockOrder != nil {
		updateBlock.BlockOrder = *req.BlockOrder
	}
	if req.BlockData != nil {
		updateBlock.BlockData = *req.BlockData
	}

	return s.blockRepo.Update(ctx, updateBlock)
}

// DeleteBlock deletes a lesson block
//
// If tutorID is not nil, it will check if the lesson block belongs to the tutor.
func (s *tutorLessonService) DeleteBlock(ctx context.Context, blockID int, tutorID *int) error {
	// Check ownership (if tutorID is not nil, it means that the lesson block is being deleted by a tutor)
	if tutorID != nil {
		exists, err := s.blockRepo.CheckOwnership(ctx, blockID, *tutorID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("you do not have rights to manage this block")
		}
	}

	return s.blockRepo.Delete(ctx, blockID)
}

// GetTutorMedia retrieves tutor or all media with filtering and pagination
func (s *tutorLessonService) GetTutorMedia(ctx context.Context, tutorID *int, mediaType *models.MediaType, page, count int) ([]models.TutorMediaResponse, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 10
	}

	return s.mediaRepo.GetByTutorID(ctx, tutorID, mediaType, page, count)
}

// CreateTutorMedia creates a new tutor media record
func (s *tutorLessonService) CreateTutorMedia(ctx context.Context, req *models.CreateTutorMediaRequest, file multipart.File, filename string) (int, error) {
	// Validate all fields are present
	if req.Slug == "" || req.MediaType == "" {
		return 0, fmt.Errorf("all fields are required")
	}

	// Validate media type
	if !s.isValidMediaType(req.MediaType) {
		return 0, fmt.Errorf("invalid media type")
	}

	// Check slug uniqueness
	exists, err := s.mediaRepo.ExistsBySlug(ctx, req.Slug)
	if err != nil {
		return 0, fmt.Errorf("failed to check slug uniqueness: %w", err)
	}
	if exists {
		return 0, fmt.Errorf("tutor media with slug '%s' already exists", req.Slug)
	}

	// Upload file to media service
	mediaTypeForService := fmt.Sprintf("lesson_%s", req.MediaType)
	url, err := uploadFileToMediaService(ctx, s.mediaBaseURL, s.apiKey, mediaTypeForService, file, filename)
	if err != nil {
		return 0, fmt.Errorf("failed to upload file: %w", err)
	}

	media := &models.TutorMedia{
		TutorID:   req.TutorID,
		Slug:      req.Slug,
		MediaType: req.MediaType,
		URL:       url,
	}

	err = s.mediaRepo.Create(ctx, media)
	if err != nil {
		return 0, fmt.Errorf("failed to create tutor media: %w", err)
	}

	return media.ID, nil
}

// DeleteTutorMedia deletes tutor media
func (s *tutorLessonService) DeleteTutorMedia(ctx context.Context, mediaID int, tutorID *int) error {
	// Get media to check ownership
	media, err := s.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return fmt.Errorf("tutor media not found")
	}

	// Check ownership
	if tutorID != nil && media.TutorID != *tutorID {
		return fmt.Errorf("you do not have permission for this media")
	}

	// Delete file from media service
	fileID := extractFileIDFromURL(media.URL)
	if fileID != "" {
		mediaTypeForService := fmt.Sprintf("lesson_%s", media.MediaType)
		err = deleteFileFromMediaService(ctx, s.mediaBaseURL, s.apiKey, mediaTypeForService, fileID)
		if err != nil {
			return fmt.Errorf("failed to delete file from media service: %w", err)
		}
	}

	// Delete record
	return s.mediaRepo.Delete(ctx, mediaID)
}

// Helper functions

func (s *tutorLessonService) isValidComplexityLevel(level models.ComplexityLevel) bool {
	validLevels := []models.ComplexityLevel{
		models.ComplexityLevelAbsoluteBeginner,
		models.ComplexityLevelBeginner,
		models.ComplexityLevelIntermediate,
		models.ComplexityLevelUpperIntermediate,
		models.ComplexityLevelAdvanced,
	}
	return slices.Contains(validLevels, level)
}

func (s *tutorLessonService) isValidBlockType(blockType models.BlockType) bool {
	validTypes := []models.BlockType{
		models.BlockTypeVideo,
		models.BlockTypeAudio,
		models.BlockTypeText,
		models.BlockTypeDocument,
		models.BlockTypeList,
	}
	return slices.Contains(validTypes, blockType)
}

func (s *tutorLessonService) isValidMediaType(mediaType models.MediaType) bool {
	validTypes := []models.MediaType{
		models.MediaTypeVideo,
		models.MediaTypeDoc,
		models.MediaTypeAudio,
	}
	return slices.Contains(validTypes, mediaType)
}
