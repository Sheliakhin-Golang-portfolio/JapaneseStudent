package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
)

// mockTutorCourseRepository is a mock implementation of TutorCourseRepository
type mockTutorCourseRepository struct {
	courses         []models.CourseListItem
	course          *models.Course
	courseShortInfo []models.CourseShortInfo
	existsBySlug    bool
	existsByTitle   bool
	err             error
	createErr       error
	updateErr       error
	deleteErr       error
	checkOwnership  bool
}

func (m *mockTutorCourseRepository) GetByID(ctx context.Context, id int) (*models.Course, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.course, nil
}

func (m *mockTutorCourseRepository) GetByAuthorOrFull(ctx context.Context, authorID *int, complexityLevel *models.ComplexityLevel, search string, page, count int) ([]models.CourseListItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.courses, nil
}

func (m *mockTutorCourseRepository) GetShortInfo(ctx context.Context, authorID *int) ([]models.CourseShortInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.courseShortInfo, nil
}

func (m *mockTutorCourseRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.existsBySlug, nil
}

func (m *mockTutorCourseRepository) ExistsByTitle(ctx context.Context, title string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.existsByTitle, nil
}

func (m *mockTutorCourseRepository) Create(ctx context.Context, course *models.Course) error {
	if m.createErr != nil {
		return m.createErr
	}
	course.ID = 1
	return m.err
}

func (m *mockTutorCourseRepository) Update(ctx context.Context, course *models.Course) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.err
}

func (m *mockTutorCourseRepository) Delete(ctx context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return m.err
}

func (m *mockTutorCourseRepository) CheckOwnership(ctx context.Context, id, tutorID int) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.checkOwnership, nil
}

// mockTutorLessonRepository is a minimal mock for testing
type mockTutorLessonRepository struct {
	lessons      []models.Lesson
	lesson       *models.Lesson
	shortInfo    []models.LessonShortInfo
	err          error
	createErr    error
	updateErr    error
	deleteErr    error
	checkOwnership bool
}

func (m *mockTutorLessonRepository) GetByID(ctx context.Context, id int) (*models.Lesson, error) {
	return m.lesson, m.err
}

func (m *mockTutorLessonRepository) GetByCourseID(ctx context.Context, courseID int) ([]models.Lesson, error) {
	return m.lessons, m.err
}

func (m *mockTutorLessonRepository) GetShortInfoByCourseID(ctx context.Context, courseID *int) ([]models.LessonShortInfo, error) {
	return m.shortInfo, m.err
}

func (m *mockTutorLessonRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return false, m.err
}

func (m *mockTutorLessonRepository) ExistsByTitleInCourse(ctx context.Context, courseID int, title string) (bool, error) {
	return false, m.err
}

func (m *mockTutorLessonRepository) ExistsByOrderInCourse(ctx context.Context, courseID int, order int) (bool, error) {
	return false, m.err
}

func (m *mockTutorLessonRepository) IncrementOrderForLessons(ctx context.Context, courseID, order int) error {
	return m.err
}

func (m *mockTutorLessonRepository) Create(ctx context.Context, lesson *models.Lesson) error {
	if m.createErr != nil {
		return m.createErr
	}
	lesson.ID = 1
	return m.err
}

func (m *mockTutorLessonRepository) Update(ctx context.Context, lesson *models.Lesson) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.err
}

func (m *mockTutorLessonRepository) Delete(ctx context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return m.err
}

func (m *mockTutorLessonRepository) CheckOwnership(ctx context.Context, id, tutorID int) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.checkOwnership, nil
}

// mockTutorLessonBlockRepository is a minimal mock for testing
type mockTutorLessonBlockRepository struct {
	blocks []models.LessonBlockResponse
	block  *models.LessonBlock
	err    error
}

func (m *mockTutorLessonBlockRepository) GetByID(ctx context.Context, id int) (*models.LessonBlock, error) {
	return m.block, m.err
}

func (m *mockTutorLessonBlockRepository) GetByLessonID(ctx context.Context, lessonID int) ([]models.LessonBlockResponse, error) {
	return m.blocks, m.err
}

func (m *mockTutorLessonBlockRepository) ExistsByOrderInLesson(ctx context.Context, lessonID int, order int) (bool, error) {
	return false, m.err
}

func (m *mockTutorLessonBlockRepository) IncrementOrderForBlocks(ctx context.Context, lessonID, order int) error {
	return m.err
}

func (m *mockTutorLessonBlockRepository) Create(ctx context.Context, block *models.LessonBlock) error {
	block.ID = 1
	return m.err
}

func (m *mockTutorLessonBlockRepository) Update(ctx context.Context, block *models.LessonBlock) error {
	return m.err
}

func (m *mockTutorLessonBlockRepository) Delete(ctx context.Context, id int) error {
	return m.err
}

func (m *mockTutorLessonBlockRepository) CheckOwnership(ctx context.Context, id, tutorID int) (bool, error) {
	return true, m.err
}

// mockTutorMediaRepository is a minimal mock for testing
type mockTutorMediaRepository struct {
	media []models.TutorMediaResponse
	err   error
}

func (m *mockTutorMediaRepository) GetByID(ctx context.Context, id int) (*models.TutorMedia, error) {
	return nil, m.err
}

func (m *mockTutorMediaRepository) GetByTutorID(ctx context.Context, tutorID *int, mediaType *models.MediaType, page, count int) ([]models.TutorMediaResponse, error) {
	return m.media, m.err
}

func (m *mockTutorMediaRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return false, m.err
}

func (m *mockTutorMediaRepository) Create(ctx context.Context, media *models.TutorMedia) error {
	media.ID = 1
	return m.err
}

func (m *mockTutorMediaRepository) Delete(ctx context.Context, id int) error {
	return m.err
}

func TestNewTutorLessonService(t *testing.T) {
	courseRepo := &mockTutorCourseRepository{}
	lessonRepo := &mockTutorLessonRepository{}
	blockRepo := &mockTutorLessonBlockRepository{}
	mediaRepo := &mockTutorMediaRepository{}

	svc := NewTutorLessonService(courseRepo, lessonRepo, blockRepo, mediaRepo, "", "")

	assert.NotNil(t, svc)
	assert.Equal(t, courseRepo, svc.courseRepo)
	assert.Equal(t, lessonRepo, svc.lessonRepo)
	assert.Equal(t, blockRepo, svc.blockRepo)
	assert.Equal(t, mediaRepo, svc.mediaRepo)
}

func TestTutorLessonService_GetCourses(t *testing.T) {
	tests := []struct {
		name           string
		tutorID        *int
		complexityLevel *models.ComplexityLevel
		search         string
		page           int
		count          int
		mockRepo       *mockTutorCourseRepository
		expectedError  bool
		expectedCount  int
	}{
		{
			name:   "success with defaults",
			tutorID: nil,
			page:    0,
			count:   0,
			mockRepo: &mockTutorCourseRepository{
				courses: []models.CourseListItem{
					{ID: 1, Title: "Course 1"},
					{ID: 2, Title: "Course 2"},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "success with pagination",
			tutorID: nil,
			page:    2,
			count:   10,
			mockRepo: &mockTutorCourseRepository{
				courses: []models.CourseListItem{
					{ID: 1, Title: "Course 1"},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "repository error",
			tutorID: nil,
			page:    1,
			count:   10,
			mockRepo: &mockTutorCourseRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:   "empty result",
			tutorID: nil,
			page:    1,
			count:   10,
			mockRepo: &mockTutorCourseRepository{
				courses: []models.CourseListItem{},
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewTutorLessonService(tt.mockRepo, &mockTutorLessonRepository{}, &mockTutorLessonBlockRepository{}, &mockTutorMediaRepository{}, "", "")
			ctx := context.Background()

			result, err := svc.GetCourses(ctx, tt.tutorID, tt.complexityLevel, tt.search, tt.page, tt.count)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}
		})
	}
}

func TestTutorLessonService_GetCoursesShortInfo(t *testing.T) {
	tests := []struct {
		name           string
		tutorID        *int
		mockRepo       *mockTutorCourseRepository
		expectedError  bool
		expectedCount  int
	}{
		{
			name:    "success",
			tutorID: intPtr(1),
			mockRepo: &mockTutorCourseRepository{
				courseShortInfo: []models.CourseShortInfo{
					{ID: 1, Title: "Course 1"},
					{ID: 2, Title: "Course 2"},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:    "success with nil tutorID",
			tutorID: nil,
			mockRepo: &mockTutorCourseRepository{
				courseShortInfo: []models.CourseShortInfo{
					{ID: 1, Title: "Course 1"},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:    "repository error",
			tutorID: intPtr(1),
			mockRepo: &mockTutorCourseRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:    "empty result",
			tutorID: intPtr(1),
			mockRepo: &mockTutorCourseRepository{
				courseShortInfo: []models.CourseShortInfo{},
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewTutorLessonService(tt.mockRepo, &mockTutorLessonRepository{}, &mockTutorLessonBlockRepository{}, &mockTutorMediaRepository{}, "", "")
			ctx := context.Background()

			result, err := svc.GetCoursesShortInfo(ctx, tt.tutorID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
