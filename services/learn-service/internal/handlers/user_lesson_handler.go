package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/learn-service/internal/models"
	authMiddleware "github.com/japanesestudent/libs/auth/middleware"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// UserLessonService is the interface that wraps methods for user lesson operations
type UserLessonService interface {
	// GetCoursesList retrieves a paginated list of courses for a user
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
	GetCoursesList(ctx context.Context, userID int, complexityLevel *models.ComplexityLevel, search string, isMine bool, page, count int) ([]models.CourseDetailResponse, error)
	// GetLessonsInCourse retrieves the details of a course and a list of lessons for a user
	//
	// "ctx" is the context for the request.
	// "courseSlug" is the slug of the course.
	// "userID" is the ID of the user.
	//
	// Returns the course details, a list of lessons, and an error if any.
	GetLessonsInCourse(ctx context.Context, courseSlug string, userID int) (*models.CourseDetailResponse, []models.LessonListItem, error)
	// GetLesson retrieves the details of a lesson for a user
	//
	// "ctx" is the context for the request.
	// "lessonSlug" is the slug of the lesson.
	// "userID" is the ID of the user.
	//
	// Returns the lesson details, a list of lesson blocks, and an error if any.
	GetLesson(ctx context.Context, lessonSlug string, userID int) (*models.LessonListItem, []models.LessonBlockResponse, error)
	// ToggleLessonCompletion toggles the completion status of a lesson for a user
	//
	// "ctx" is the context for the request.
	// "lessonSlug" is the slug of the lesson.
	// "userID" is the ID of the user.
	//
	// Returns an error if any.
	ToggleLessonCompletion(ctx context.Context, lessonSlug string, userID int) error
}

// UserLessonHandler handles HTTP requests for user lesson operations
type UserLessonHandler struct {
	handlers.BaseHandler
	service UserLessonService
}

// NewUserLessonHandler creates a new user lesson handler
func NewUserLessonHandler(svc UserLessonService, logger *zap.Logger) *UserLessonHandler {
	return &UserLessonHandler{
		service:     svc,
		BaseHandler: handlers.BaseHandler{Logger: logger},
	}
}

// RegisterRoutes registers all user lesson handler routes
func (h *UserLessonHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/courses", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", h.GetCoursesList)
		r.Get("/{slug}/lessons", h.GetLessonsInCourse)
	})
	r.Route("/lessons", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/{slug}", h.GetLesson)
		r.Post("/{slug}/complete", h.ToggleLessonCompletion)
	})
}

// GetCoursesList handles GET /courses
// @Summary Get list of courses
// @Description Get a paginated list of courses with optional filtering by complexity level, search, and isMine flag
// @Tags lessons
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param complexityLevel query string false "Complexity level (ab, b, i, ui, a)"
// @Param search query string false "Search by course title"
// @Param isMine query bool false "Filter courses by user's completion history"
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 10)"
// @Success 200 {array} models.CourseDetailResponse "List of courses"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /courses [get]
func (h *UserLessonHandler) GetCoursesList(w http.ResponseWriter, r *http.Request) {
	// Extract userID from context
	userID, ok := authMiddleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Parse query parameters
	complexityLevelStr := r.URL.Query().Get("complexityLevel")
	search := r.URL.Query().Get("search")
	isMineStr := r.URL.Query().Get("isMine")
	pageStr := r.URL.Query().Get("page")
	countStr := r.URL.Query().Get("count")

	// Parse complexity level
	var complexityLevel *models.ComplexityLevel
	if complexityLevelStr != "" {
		// Check if it's an abbreviation
		if level, ok := models.ComplexityLevelAbbreviation[complexityLevelStr]; ok {
			complexityLevel = &level
		} else {
			// Try as full name
			level := models.ComplexityLevel(complexityLevelStr)
			complexityLevel = &level
		}
	}

	// Parse isMine
	isMine := isMineStr == "true"
	if isMineStr == "true" {
		isMine = true
	}

	// Parse pagination
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	count := 10
	if countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 {
			count = c
		}
	}

	courses, err := h.service.GetCoursesList(r.Context(), userID, complexityLevel, search, isMine, page, count)
	if err != nil {
		h.Logger.Error("failed to get courses list", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, courses)
}

// GetLessonsInCourse handles GET /courses/{slug}/lessons
// @Summary Get lessons in a course
// @Description Get course details with list of lessons and completion status
// @Tags lessons
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param slug path string true "Course slug"
// @Success 200 {object} map[string]any{} "Course with lessons"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Course not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /courses/{slug}/lessons [get]
func (h *UserLessonHandler) GetLessonsInCourse(w http.ResponseWriter, r *http.Request) {
	// Extract userID from context
	userID, ok := authMiddleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	courseSlug := chi.URLParam(r, "slug")
	if courseSlug == "" {
		h.RespondError(w, http.StatusBadRequest, "course slug is required")
		return
	}

	course, lessons, err := h.service.GetLessonsInCourse(r.Context(), courseSlug, userID)
	if err != nil {
		h.Logger.Error("failed to get lessons in course", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "course not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	response := map[string]any{
		"course":  course,
		"lessons": lessons,
	}

	h.RespondJSON(w, http.StatusOK, response)
}

// GetLesson handles GET /lessons/{slug}
// @Summary Get lesson details
// @Description Get full lesson details with blocks and completion status
// @Tags lessons
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param slug path string true "Lesson slug"
// @Success 200 {object} map[string]any{} "Lesson details"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Lesson not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /lessons/{slug} [get]
func (h *UserLessonHandler) GetLesson(w http.ResponseWriter, r *http.Request) {
	// Extract userID from context
	userID, ok := authMiddleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	lessonSlug := chi.URLParam(r, "slug")
	if lessonSlug == "" {
		h.RespondError(w, http.StatusBadRequest, "lesson slug is required")
		return
	}

	lesson, blocks, err := h.service.GetLesson(r.Context(), lessonSlug, userID)
	if err != nil {
		h.Logger.Error("failed to get lesson", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "lesson not found" || err.Error() == "failed to get lesson: lesson not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	response := map[string]any{
		"lesson": lesson,
		"blocks": blocks,
	}

	h.RespondJSON(w, http.StatusOK, response)
}

// ToggleLessonCompletion handles POST /lessons/{slug}/complete
// @Summary Toggle lesson completion
// @Description Complete or uncomplete a lesson
// @Tags lessons
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param slug path string true "Lesson slug"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Lesson not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /lessons/{slug}/complete [post]
func (h *UserLessonHandler) ToggleLessonCompletion(w http.ResponseWriter, r *http.Request) {
	// Extract userID from context
	userID, ok := authMiddleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	lessonSlug := chi.URLParam(r, "slug")
	if lessonSlug == "" {
		h.RespondError(w, http.StatusBadRequest, "lesson slug is required")
		return
	}

	err := h.service.ToggleLessonCompletion(r.Context(), lessonSlug, userID)
	if err != nil {
		h.Logger.Error("failed to toggle lesson completion", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "lesson not found" || err.Error() == "failed to get lesson: lesson not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
