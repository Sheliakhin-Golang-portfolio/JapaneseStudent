package services

import (
	"context"
	"errors"
	"testing"

	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
)

// mockCourseRepository is a mock implementation of CourseRepository
type mockCourseRepository struct {
	course       *models.CourseDetailResponse
	courses      []models.CourseDetailResponse
	err          error
	getBySlugErr error
}

func (m *mockCourseRepository) GetBySlug(ctx context.Context, slug string, userID int) (*models.CourseDetailResponse, error) {
	if m.getBySlugErr != nil {
		return nil, m.getBySlugErr
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.course, nil
}

func (m *mockCourseRepository) GetAll(ctx context.Context, userID int, complexityLevel *models.ComplexityLevel, search string, isMine bool, page, count int) ([]models.CourseDetailResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.courses, nil
}

// mockLessonRepository is a mock implementation of LessonRepository
type mockLessonRepository struct {
	lesson       *models.LessonListItem
	lessons      []models.LessonListItem
	err          error
	getBySlugErr error
}

func (m *mockLessonRepository) GetBySlug(ctx context.Context, slug string, userID int) (*models.LessonListItem, error) {
	if m.getBySlugErr != nil {
		return nil, m.getBySlugErr
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.lesson, nil
}

func (m *mockLessonRepository) GetByCourseIDWithCompletion(ctx context.Context, courseID, userID int) ([]models.LessonListItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.lessons, nil
}

// mockLessonBlockRepository is a mock implementation of LessonBlockRepository
type mockLessonBlockRepository struct {
	blocks []models.LessonBlockResponse
	err    error
}

func (m *mockLessonBlockRepository) GetByLessonID(ctx context.Context, lessonID int) ([]models.LessonBlockResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.blocks, nil
}

// mockLessonUserHistoryRepository is a mock implementation of LessonUserHistoryRepository
type mockLessonUserHistoryRepository struct {
	exists       bool
	err          error
	existsErr    error
	createErr    error
	deleteErr    error
	countErr     error
	count        int
	createCalled bool
	deleteCalled bool
}

func (m *mockLessonUserHistoryRepository) Exists(ctx context.Context, userID, courseID, lessonID int) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	return m.exists, nil
}

func (m *mockLessonUserHistoryRepository) Create(ctx context.Context, history *models.LessonUserHistory) error {
	m.createCalled = true
	return m.createErr
}

func (m *mockLessonUserHistoryRepository) Delete(ctx context.Context, userID, courseID, lessonID int) error {
	m.deleteCalled = true
	return m.deleteErr
}

func (m *mockLessonUserHistoryRepository) CountCompletedLessonsByCourse(ctx context.Context, userID, courseID int) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.count, nil
}

func TestNewUserLessonService(t *testing.T) {
	courseRepo := &mockCourseRepository{}
	lessonRepo := &mockLessonRepository{}
	blockRepo := &mockLessonBlockRepository{}
	historyRepo := &mockLessonUserHistoryRepository{}

	svc := NewUserLessonService(courseRepo, lessonRepo, blockRepo, historyRepo)

	assert.NotNil(t, svc)
	assert.Equal(t, courseRepo, svc.courseRepo)
	assert.Equal(t, lessonRepo, svc.lessonRepo)
	assert.Equal(t, blockRepo, svc.blockRepo)
	assert.Equal(t, historyRepo, svc.historyRepo)
}

func TestUserLessonService_GetCoursesList(t *testing.T) {
	tests := []struct {
		name            string
		userID          int
		complexityLevel *models.ComplexityLevel
		search          string
		isMine          bool
		page            int
		count           int
		courseRepo      *mockCourseRepository
		expectedError   bool
		expectedCount   int
	}{
		{
			name:   "success with defaults",
			userID: 1,
			page:   1,
			count:  10,
			courseRepo: &mockCourseRepository{
				courses: []models.CourseDetailResponse{
					{Title: "Course 1", ComplexityLevel: models.ComplexityLevelBeginner},
					{Title: "Course 2", ComplexityLevel: models.ComplexityLevelIntermediate},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "success with pagination",
			userID: 1,
			page:   2,
			count:  5,
			courseRepo: &mockCourseRepository{
				courses: []models.CourseDetailResponse{
					{Title: "Course 6", ComplexityLevel: models.ComplexityLevelBeginner},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with complexity filter",
			userID: 1,
			complexityLevel: func() *models.ComplexityLevel { level := models.ComplexityLevelBeginner; return &level }(),
			page:   1,
			count:  10,
			courseRepo: &mockCourseRepository{
				courses: []models.CourseDetailResponse{
					{Title: "Course 1", ComplexityLevel: models.ComplexityLevelBeginner},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with search",
			userID: 1,
			search: "test",
			page:   1,
			count:  10,
			courseRepo: &mockCourseRepository{
				courses: []models.CourseDetailResponse{
					{Title: "Test Course", ComplexityLevel: models.ComplexityLevelBeginner},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with isMine filter",
			userID: 1,
			isMine: true,
			page:   1,
			count:  10,
			courseRepo: &mockCourseRepository{
				courses: []models.CourseDetailResponse{
					{Title: "My Course", ComplexityLevel: models.ComplexityLevelBeginner},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "page less than 1 defaults to 1",
			userID: 1,
			page:   0,
			count:  10,
			courseRepo: &mockCourseRepository{
				courses: []models.CourseDetailResponse{
					{Title: "Course 1", ComplexityLevel: models.ComplexityLevelBeginner},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "count less than 1 defaults to 10",
			userID: 1,
			page:   1,
			count:  0,
			courseRepo: &mockCourseRepository{
				courses: []models.CourseDetailResponse{
					{Title: "Course 1", ComplexityLevel: models.ComplexityLevelBeginner},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "repository error",
			userID: 1,
			page:   1,
			count:  10,
			courseRepo: &mockCourseRepository{
				err: errors.New("repository error"),
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserLessonService(
				tt.courseRepo,
				&mockLessonRepository{},
				&mockLessonBlockRepository{},
				&mockLessonUserHistoryRepository{},
			)

			result, err := svc.GetCoursesList(
				context.Background(),
				tt.userID,
				tt.complexityLevel,
				tt.search,
				tt.isMine,
				tt.page,
				tt.count,
			)

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

func TestUserLessonService_GetLessonsInCourse(t *testing.T) {
	tests := []struct {
		name           string
		courseSlug     string
		userID         int
		courseRepo     *mockCourseRepository
		lessonRepo     *mockLessonRepository
		expectedError  bool
		errorContains  string
		expectedCourse bool
		expectedCount  int
	}{
		{
			name:       "success",
			courseSlug: "test-course",
			userID:     1,
			courseRepo: &mockCourseRepository{
				course: &models.CourseDetailResponse{
					ID:              1,
					Title:           "Test Course",
					ComplexityLevel: models.ComplexityLevelBeginner,
					TotalLessons:    5,
					CompletedLessons: 2,
				},
			},
			lessonRepo: &mockLessonRepository{
				lessons: []models.LessonListItem{
					{Title: "Lesson 1", Completed: true},
					{Title: "Lesson 2", Completed: false},
				},
			},
			expectedError:  false,
			expectedCourse: true,
			expectedCount:  2,
		},
		{
			name:       "course not found",
			courseSlug: "nonexistent",
			userID:     1,
			courseRepo: &mockCourseRepository{
				getBySlugErr: errors.New("course not found"),
			},
			lessonRepo:     &mockLessonRepository{},
			expectedError:  true,
			errorContains:  "failed to get course",
			expectedCourse: false,
			expectedCount:  0,
		},
		{
			name:       "failed to get lessons",
			courseSlug: "test-course",
			userID:     1,
			courseRepo: &mockCourseRepository{
				course: &models.CourseDetailResponse{
					ID:     1,
					Title:  "Test Course",
					ComplexityLevel: models.ComplexityLevelBeginner,
				},
			},
			lessonRepo: &mockLessonRepository{
				err: errors.New("failed to get lessons"),
			},
			expectedError:  true,
			errorContains:  "failed to get lessons",
			expectedCourse: false,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserLessonService(
				tt.courseRepo,
				tt.lessonRepo,
				&mockLessonBlockRepository{},
				&mockLessonUserHistoryRepository{},
			)

			course, lessons, err := svc.GetLessonsInCourse(context.Background(), tt.courseSlug, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, course)
				assert.Nil(t, lessons)
			} else {
				assert.NoError(t, err)
				if tt.expectedCourse {
					assert.NotNil(t, course)
					assert.Equal(t, 0, course.ID, "course ID should be cleared")
				}
				assert.NotNil(t, lessons)
				assert.Len(t, lessons, tt.expectedCount)
			}
		})
	}
}

func TestUserLessonService_GetLesson(t *testing.T) {
	tests := []struct {
		name           string
		lessonSlug     string
		userID         int
		lessonRepo     *mockLessonRepository
		blockRepo      *mockLessonBlockRepository
		expectedError  bool
		errorContains  string
		expectedLesson bool
		expectedBlocks int
	}{
		{
			name:       "success",
			lessonSlug: "test-lesson",
			userID:     1,
			lessonRepo: &mockLessonRepository{
				lesson: &models.LessonListItem{
					ID:        1,
					CourseID:  1,
					Title:     "Test Lesson",
					Completed: false,
				},
			},
			blockRepo: &mockLessonBlockRepository{
				blocks: []models.LessonBlockResponse{
					{ID: 1, BlockType: "text"},
					{ID: 2, BlockType: "image"},
				},
			},
			expectedError:  false,
			expectedLesson: true,
			expectedBlocks: 2,
		},
		{
			name:       "lesson not found",
			lessonSlug: "nonexistent",
			userID:     1,
			lessonRepo: &mockLessonRepository{
				getBySlugErr: errors.New("lesson not found"),
			},
			blockRepo:      &mockLessonBlockRepository{},
			expectedError:  true,
			errorContains:  "failed to get lesson",
			expectedLesson: false,
			expectedBlocks: 0,
		},
		{
			name:       "failed to get lesson blocks",
			lessonSlug: "test-lesson",
			userID:     1,
			lessonRepo: &mockLessonRepository{
				lesson: &models.LessonListItem{
					ID:       1,
					CourseID: 1,
					Title:    "Test Lesson",
				},
			},
			blockRepo: &mockLessonBlockRepository{
				err: errors.New("failed to get blocks"),
			},
			expectedError:  true,
			errorContains:  "failed to get lesson blocks",
			expectedLesson: false,
			expectedBlocks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserLessonService(
				&mockCourseRepository{},
				tt.lessonRepo,
				tt.blockRepo,
				&mockLessonUserHistoryRepository{},
			)

			lesson, blocks, err := svc.GetLesson(context.Background(), tt.lessonSlug, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, lesson)
				assert.Nil(t, blocks)
			} else {
				assert.NoError(t, err)
				if tt.expectedLesson {
					assert.NotNil(t, lesson)
					assert.Equal(t, 0, lesson.ID, "lesson ID should be cleared")
					assert.Equal(t, 0, lesson.CourseID, "course ID should be cleared")
				}
				assert.NotNil(t, blocks)
				assert.Len(t, blocks, tt.expectedBlocks)
			}
		})
	}
}

func TestUserLessonService_ToggleLessonCompletion(t *testing.T) {
	tests := []struct {
		name          string
		lessonSlug    string
		userID        int
		lessonRepo    *mockLessonRepository
		historyRepo   *mockLessonUserHistoryRepository
		expectedError bool
		errorContains string
		shouldCreate  bool
		shouldDelete  bool
	}{
		{
			name:       "success - create history (complete)",
			lessonSlug: "test-lesson",
			userID:     1,
			lessonRepo: &mockLessonRepository{
				lesson: &models.LessonListItem{
					ID:       1,
					CourseID: 1,
					Title:    "Test Lesson",
				},
			},
			historyRepo: &mockLessonUserHistoryRepository{
				exists: false,
			},
			expectedError: false,
			shouldCreate:  true,
			shouldDelete:  false,
		},
		{
			name:       "success - delete history (uncomplete)",
			lessonSlug: "test-lesson",
			userID:     1,
			lessonRepo: &mockLessonRepository{
				lesson: &models.LessonListItem{
					ID:       1,
					CourseID: 1,
					Title:    "Test Lesson",
				},
			},
			historyRepo: &mockLessonUserHistoryRepository{
				exists: true,
			},
			expectedError: false,
			shouldCreate:  false,
			shouldDelete:  true,
		},
		{
			name:       "lesson not found",
			lessonSlug: "nonexistent",
			userID:     1,
			lessonRepo: &mockLessonRepository{
				getBySlugErr: errors.New("lesson not found"),
			},
			historyRepo:   &mockLessonUserHistoryRepository{},
			expectedError: true,
			errorContains: "failed to get lesson",
			shouldCreate:  false,
			shouldDelete:  false,
		},
		{
			name:       "failed to check history existence",
			lessonSlug: "test-lesson",
			userID:     1,
			lessonRepo: &mockLessonRepository{
				lesson: &models.LessonListItem{
					ID:       1,
					CourseID: 1,
					Title:    "Test Lesson",
				},
			},
			historyRepo: &mockLessonUserHistoryRepository{
				existsErr: errors.New("failed to check existence"),
			},
			expectedError: true,
			errorContains: "failed to check history existence",
			shouldCreate:  false,
			shouldDelete:  false,
		},
		{
			name:       "failed to create history",
			lessonSlug: "test-lesson",
			userID:     1,
			lessonRepo: &mockLessonRepository{
				lesson: &models.LessonListItem{
					ID:       1,
					CourseID: 1,
					Title:    "Test Lesson",
				},
			},
			historyRepo: &mockLessonUserHistoryRepository{
				exists:    false,
				createErr: errors.New("failed to create"),
			},
			expectedError: true,
			errorContains: "failed to create history record",
			shouldCreate:  true,
			shouldDelete:  false,
		},
		{
			name:       "failed to delete history",
			lessonSlug: "test-lesson",
			userID:     1,
			lessonRepo: &mockLessonRepository{
				lesson: &models.LessonListItem{
					ID:       1,
					CourseID: 1,
					Title:    "Test Lesson",
				},
			},
			historyRepo: &mockLessonUserHistoryRepository{
				exists:    true,
				deleteErr: errors.New("failed to delete"),
			},
			expectedError: true,
			errorContains: "failed to delete history record",
			shouldCreate:  false,
			shouldDelete:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			historyRepo := &mockLessonUserHistoryRepository{
				exists:    tt.historyRepo.exists,
				existsErr: tt.historyRepo.existsErr,
				createErr: tt.historyRepo.createErr,
				deleteErr: tt.historyRepo.deleteErr,
			}

			svc := NewUserLessonService(
				&mockCourseRepository{},
				tt.lessonRepo,
				&mockLessonBlockRepository{},
				historyRepo,
			)

			err := svc.ToggleLessonCompletion(context.Background(), tt.lessonSlug, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			if tt.shouldCreate {
				assert.True(t, historyRepo.createCalled, "Create should have been called")
			} else if tt.shouldDelete {
				assert.True(t, historyRepo.deleteCalled, "Delete should have been called")
			}
		})
	}
}
