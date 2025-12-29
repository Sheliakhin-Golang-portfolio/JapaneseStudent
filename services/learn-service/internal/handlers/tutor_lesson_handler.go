package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/learn-service/internal/models"
	authMiddleware "github.com/japanesestudent/libs/auth/middleware"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// TutorLessonService is the interface that wraps methods for tutor lesson operations
type TutorLessonService interface {
	// GetCourses retrieves a list of courses for a tutor
	//
	// "ctx" is the context for the request.
	// "tutorID" is the ID of the tutor.
	// "complexityLevel" is the complexity level of the courses to retrieve.
	// "search" is the search query for the courses.
	// "page" is the page number to retrieve.
	// "count" is the number of items per page.
	//
	// Returns a list of courses and an error if any.
	GetCourses(ctx context.Context, tutorID *int, complexityLevel *models.ComplexityLevel, search string, page, count int) ([]models.CourseListItem, error)
	// CreateCourse creates a new course
	//
	// "ctx" is the context for the request.
	// "req" is the request to create a course.
	//
	// Returns the ID of the created course and an error if any.
	CreateCourse(ctx context.Context, req *models.CreateCourseRequest) (int, error)
	// UpdateCourse updates a course
	//
	// "ctx" is the context for the request.
	// "courseID" is the ID of the course.
	// "tutorID" is the ID of the tutor (optional, if nil, the course is being updated by an admin).
	// "req" is the request to update a course.
	//
	// Returns an error if any.
	UpdateCourse(ctx context.Context, courseID int, tutorID *int, req *models.UpdateCourseRequest) error
	// DeleteCourse deletes a course
	//
	// "ctx" is the context for the request.
	// "courseID" is the ID of the course.
	// "tutorID" is the ID of the tutor (optional, if nil, the course is being deleted by an admin).
	//
	// Returns an error if any.
	//
	DeleteCourse(ctx context.Context, courseID int, tutorID *int) error
	// GetCoursesShortInfo retrieves a list of short course information for a tutor
	//
	// "ctx" is the context for the request.
	// "tutorID" is the ID of the tutor (optional, if nil, the courses are being retrieved by an admin).
	//
	// Returns a list of short course information and an error if any.
	GetCoursesShortInfo(ctx context.Context, tutorID *int) ([]models.CourseShortInfo, error)
	// GetLessonsForCourse retrieves a list of lessons for a course
	//
	// "ctx" is the context for the request.
	// "courseID" is the ID of the course.
	// "tutorID" is the ID of the tutor (optional, if nil, the lessons are being retrieved by an admin).
	//
	// Returns the course and a list of lessons and an error if any.
	GetLessonsForCourse(ctx context.Context, courseID int, tutorID *int) (*models.Course, []models.Lesson, error)
	// CreateLesson creates a new lesson
	//
	// "ctx" is the context for the request.
	// "tutorID" is the ID of the tutor (optional, if nil, the lesson is being created by an admin).
	// "req" is the request to create a lesson.
	//
	// Returns the ID of the created lesson and an error if any.
	CreateLesson(ctx context.Context, tutorID *int, req *models.CreateLessonRequest) (int, error)
	// UpdateLesson updates a lesson
	//
	// "ctx" is the context for the request.
	// "lessonID" is the ID of the lesson.
	// "tutorID" is the ID of the tutor (optional, if nil, the lesson is being updated by an admin).
	// "req" is the request to update a lesson.
	//
	// Returns an error if any.
	UpdateLesson(ctx context.Context, lessonID int, tutorID *int, req *models.UpdateLessonRequest) error
	// DeleteLesson deletes a lesson
	//
	// "ctx" is the context for the request.
	// "lessonID" is the ID of the lesson.
	// "tutorID" is the ID of the tutor (optional, if nil, the lesson is being deleted by an admin).
	//
	// Returns an error if any.
	DeleteLesson(ctx context.Context, lessonID int, tutorID *int) error
	// GetFullLessonInfo retrieves a full lesson information
	//
	// "ctx" is the context for the request.
	// "lessonID" is the ID of the lesson.
	// "tutorID" is the ID of the tutor (optional, if nil, the lesson is being retrieved by an admin).
	//
	// Returns the lesson and a list of lesson blocks and an error if any.
	GetFullLessonInfo(ctx context.Context, lessonID int, tutorID *int) (*models.Lesson, []models.LessonBlockResponse, error)
	// GetLessonsShortInfo retrieves a list of short lesson information for a course
	//
	// "ctx" is the context for the request.
	// "courseID" is the ID of the course (optional, if nil, the lessons are being retrieved by an admin).
	// "tutorID" is the ID of the tutor (optional, if nil, the lessons are being retrieved by an admin).
	//
	// Returns a list of short lesson information and an error if any.
	GetLessonsShortInfo(ctx context.Context, courseID, tutorID *int) ([]models.LessonShortInfo, error)
	// CreateLessonBlock creates a new lesson block
	//
	// "ctx" is the context for the request.
	// "tutorID" is the ID of the tutor (optional, if nil, the lesson block is being created by an admin).
	// "req" is the request to create a lesson block.
	//
	// Returns the ID of the created lesson block and an error if any.
	CreateLessonBlock(ctx context.Context, tutorID *int, req *models.CreateLessonBlockRequest) (int, error)
	// UpdateLessonBlock updates a lesson block
	//
	// "ctx" is the context for the request.
	// "blockID" is the ID of the lesson block.
	// "tutorID" is the ID of the tutor (optional, if nil, the lesson block is being updated by an admin).
	// "req" is the request to update a lesson block.
	//
	// Returns an error if any.
	UpdateLessonBlock(ctx context.Context, blockID int, tutorID *int, req *models.UpdateLessonBlockRequest) error
	// DeleteBlock deletes a lesson block
	//
	// "ctx" is the context for the request.
	// "blockID" is the ID of the lesson block.
	// "tutorID" is the ID of the tutor (optional, if nil, the lesson block is being deleted by an admin).
	//
	// Returns an error if any.
	DeleteBlock(ctx context.Context, blockID int, tutorID *int) error
	// GetTutorMedia retrieves a list of tutor media
	//
	// "ctx" is the context for the request.
	// "tutorID" is the ID of the tutor (optional, if nil, the tutor media is being retrieved by an admin).
	// "mediaType" is the type of media to retrieve.
	// "page" is the page number to retrieve.
	// "count" is the number of items per page.
	//
	// Returns a list of tutor media and an error if any.
	GetTutorMedia(ctx context.Context, tutorID *int, mediaType *models.MediaType, page, count int) ([]models.TutorMediaResponse, error)
	// CreateTutorMedia creates a new tutor media
	//
	// "ctx" is the context for the request.
	// "req" is the request to create a tutor media.
	// "file" is the file to upload.
	// "filename" is the name of the file.
	//
	// Returns the ID of the created tutor media and an error if any.
	CreateTutorMedia(ctx context.Context, req *models.CreateTutorMediaRequest, file multipart.File, filename string) (int, error)
	// DeleteTutorMedia deletes a tutor media
	//
	// "ctx" is the context for the request.
	// "mediaID" is the ID of the tutor media.
	// "tutorID" is the ID of the tutor (optional, if nil, the tutor media is being deleted by an admin).
	//
	// Returns an error if any.
	DeleteTutorMedia(ctx context.Context, mediaID int, tutorID *int) error
}

// TutorLessonHandler handles HTTP requests for tutor lesson operations
type TutorLessonHandler struct {
	handlers.BaseHandler
	service TutorLessonService
}

// NewTutorLessonHandler creates a new tutor lesson handler
func NewTutorLessonHandler(svc TutorLessonService, logger *zap.Logger) *TutorLessonHandler {
	return &TutorLessonHandler{
		service:     svc,
		BaseHandler: handlers.BaseHandler{Logger: logger},
	}
}

// RegisterRoutes registers all tutor lesson handler routes
func (h *TutorLessonHandler) RegisterRoutes(r chi.Router) {
	r.Route("/tutor", func(r chi.Router) {
		r.Route("/courses", func(r chi.Router) {
			r.Get("/", h.GetCourses)
			r.Post("/", h.CreateCourse)
			r.Get("/short", h.GetCoursesShortInfo)
			r.Get("/{id}/lessons", h.GetLessonsForCourse)
			r.Patch("/{id}", h.UpdateCourse)
			r.Delete("/{id}", h.DeleteCourse)
		})
		r.Route("/lessons", func(r chi.Router) {
			r.Post("/", h.CreateLesson)
			r.Get("/short", h.GetLessonsShortInfo)
			r.Get("/{id}", h.GetFullLessonInfo)
			r.Patch("/{id}", h.UpdateLesson)
			r.Delete("/{id}", h.DeleteLesson)
		})
		r.Route("/blocks", func(r chi.Router) {
			r.Post("/", h.CreateLessonBlock)
			r.Patch("/{id}", h.UpdateLessonBlock)
			r.Delete("/{id}", h.DeleteBlock)
		})
		r.Route("/media", func(r chi.Router) {
			r.Get("/", h.GetTutorMedia)
			r.Post("/", h.CreateTutorMedia)
			r.Delete("/{id}", h.DeleteTutorMedia)
		})
	})
}

// getTutorID extracts tutor ID from context
func (h *TutorLessonHandler) getTutorID(r *http.Request) (int, error) {
	userID, ok := authMiddleware.GetUserID(r.Context())
	if !ok {
		return 0, fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}

// GetCourses handles GET /tutor/courses
// @Summary Get list of courses
// @Description Get paginated list of courses for the authenticated tutor with optional complexity level and search filters
// @Tags tutor
// @Accept json
// @Produce json
// @Param complexityLevel query string false "Complexity level (ab, b, i, ui, a) or full name"
// @Param search query string false "Search query"
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 10)"
// @Success 200 {array} models.CourseListItem "List of courses"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/courses [get]
func (h *TutorLessonHandler) GetCourses(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	complexityLevelStr := r.URL.Query().Get("complexityLevel")
	search := r.URL.Query().Get("search")
	pageStr := r.URL.Query().Get("page")
	countStr := r.URL.Query().Get("count")

	var complexityLevel *models.ComplexityLevel
	if complexityLevelStr != "" {
		if level, ok := models.ComplexityLevelAbbreviation[complexityLevelStr]; ok {
			complexityLevel = &level
		} else {
			level := models.ComplexityLevel(complexityLevelStr)
			complexityLevel = &level
		}
	}

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

	courses, err := h.service.GetCourses(r.Context(), &tutorID, complexityLevel, search, page, count)
	if err != nil {
		h.Logger.Error("failed to get my courses", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, courses)
}

// CreateCourse handles POST /tutor/courses
// @Summary Create a course
// @Description Create a new course for the authenticated tutor
// @Tags tutor
// @Accept json
// @Produce json
// @Param request body models.CreateCourseRequest true "Course creation request"
// @Success 201 {object} map[string]any "Course created successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/courses [post]
func (h *TutorLessonHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req models.CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.AuthorID = tutorID
	courseID, err := h.service.CreateCourse(r.Context(), &req)
	if err != nil {
		h.Logger.Error("failed to create course", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]any{
		"id":      courseID,
		"message": "course created successfully",
	})
}

// UpdateCourse handles PATCH /tutor/courses/{id}
// @Summary Update a course
// @Description Update a course owned by the authenticated tutor (partial update)
// @Tags tutor
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Param request body models.UpdateCourseRequest true "Course update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not course owner or not found"
// @Router /tutor/courses/{id} [patch]
func (h *TutorLessonHandler) UpdateCourse(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	courseIDStr := chi.URLParam(r, "id")
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid course ID")
		return
	}

	var req models.UpdateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.service.UpdateCourse(r.Context(), courseID, &tutorID, &req)
	if err != nil {
		h.Logger.Error("failed to update course", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "rights") {
			errStatus = http.StatusForbidden
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteCourse handles DELETE /tutor/courses/{id}
// @Summary Delete a course
// @Description Delete a course owned by the authenticated tutor
// @Tags tutor
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid course ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not course owner or not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/courses/{id} [delete]
func (h *TutorLessonHandler) DeleteCourse(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	courseIDStr := chi.URLParam(r, "id")
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid course ID")
		return
	}

	err = h.service.DeleteCourse(r.Context(), courseID, &tutorID)
	if err != nil {
		h.Logger.Error("failed to delete course", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "rights") {
			errStatus = http.StatusForbidden
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCoursesShortInfo handles GET /tutor/courses/short
// @Summary Get courses short info
// @Description Get list of courses with only ID and title for the authenticated tutor (for select options)
// @Tags tutor
// @Accept json
// @Produce json
// @Success 200 {array} models.CourseShortInfo "List of courses short info"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/courses/short [get]
func (h *TutorLessonHandler) GetCoursesShortInfo(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	courses, err := h.service.GetCoursesShortInfo(r.Context(), &tutorID)
	if err != nil {
		h.Logger.Error("failed to get courses short info", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, courses)
}

// GetLessonsForCourse handles GET /tutor/courses/{id}/lessons
// @Summary Get lessons for a course
// @Description Get course details and list of lessons for a course owned by the authenticated tutor
// @Tags tutor
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Success 200 {object} map[string]any "Course with lessons"
// @Failure 400 {object} map[string]string "Invalid course ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not course owner or not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/courses/{id}/lessons [get]
func (h *TutorLessonHandler) GetLessonsForCourse(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	courseIDStr := chi.URLParam(r, "id")
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid course ID")
		return
	}

	course, lessons, err := h.service.GetLessonsForCourse(r.Context(), courseID, &tutorID)
	if err != nil {
		h.Logger.Error("failed to get lessons for course", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "rights") {
			errStatus = http.StatusForbidden
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

// CreateLesson handles POST /tutor/lessons
// @Summary Create a lesson
// @Description Create a new lesson in a course owned by the authenticated tutor
// @Tags tutor
// @Accept json
// @Produce json
// @Param request body models.CreateLessonRequest true "Lesson creation request"
// @Success 201 {object} map[string]any "Lesson created successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - course does not belong to tutor"
// @Failure 404 {object} map[string]string "Course not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/lessons [post]
func (h *TutorLessonHandler) CreateLesson(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req models.CreateLessonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	lessonID, err := h.service.CreateLesson(r.Context(), &tutorID, &req)
	if err != nil {
		h.Logger.Error("failed to create lesson", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		} else if strings.Contains(err.Error(), "belongs to you") {
			errStatus = http.StatusForbidden
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]any{
		"id":      lessonID,
		"message": "lesson created successfully",
	})
}

// UpdateLesson handles PATCH /tutor/lessons/{id}
// @Summary Update a lesson
// @Description Update a lesson in a course owned by the authenticated tutor (partial update)
// @Tags tutor
// @Accept json
// @Produce json
// @Param id path int true "Lesson ID"
// @Param request body models.UpdateLessonRequest true "Lesson update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not lesson owner or not found"
// @Router /tutor/lessons/{id} [patch]
func (h *TutorLessonHandler) UpdateLesson(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	lessonIDStr := chi.URLParam(r, "id")
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid lesson ID")
		return
	}

	var req models.UpdateLessonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.service.UpdateLesson(r.Context(), lessonID, &tutorID, &req)
	if err != nil {
		h.Logger.Error("failed to update lesson", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "rights") {
			errStatus = http.StatusForbidden
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteLesson handles DELETE /tutor/lessons/{id}
// @Summary Delete a lesson
// @Description Delete a lesson in a course owned by the authenticated tutor
// @Tags tutor
// @Accept json
// @Produce json
// @Param id path int true "Lesson ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid lesson ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not lesson owner or not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/lessons/{id} [delete]
func (h *TutorLessonHandler) DeleteLesson(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	lessonIDStr := chi.URLParam(r, "id")
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid lesson ID")
		return
	}

	err = h.service.DeleteLesson(r.Context(), lessonID, &tutorID)
	if err != nil {
		h.Logger.Error("failed to delete lesson", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "rights") {
			errStatus = http.StatusForbidden
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetFullLessonInfo handles GET /tutor/lessons/{id}
// @Summary Get full lesson info
// @Description Get lesson details and list of lesson blocks for a lesson in a course owned by the authenticated tutor
// @Tags tutor
// @Accept json
// @Produce json
// @Param id path int true "Lesson ID"
// @Success 200 {object} map[string]any "Lesson with blocks"
// @Failure 400 {object} map[string]string "Invalid lesson ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not lesson owner or not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/lessons/{id} [get]
func (h *TutorLessonHandler) GetFullLessonInfo(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	lessonIDStr := chi.URLParam(r, "id")
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid lesson ID")
		return
	}

	lesson, blocks, err := h.service.GetFullLessonInfo(r.Context(), lessonID, &tutorID)
	if err != nil {
		h.Logger.Error("failed to get full lesson info", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "rights") {
			errStatus = http.StatusForbidden
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

// GetLessonsShortInfo handles GET /tutor/lessons/short
// @Summary Get lessons short info
// @Description Get list of lessons with only ID and title for a course owned by the authenticated tutor (for select options)
// @Tags tutor
// @Accept json
// @Produce json
// @Param courseId query int true "Course ID"
// @Success 200 {array} models.LessonShortInfo "List of lessons short info"
// @Failure 400 {object} map[string]string "Invalid course ID or missing courseId parameter"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not course owner or not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/lessons/short [get]
func (h *TutorLessonHandler) GetLessonsShortInfo(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	courseIDStr := r.URL.Query().Get("courseId")
	if courseIDStr == "" {
		h.RespondError(w, http.StatusBadRequest, "courseId query parameter is required")
		return
	}

	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid course ID")
		return
	}

	lessons, err := h.service.GetLessonsShortInfo(r.Context(), &courseID, &tutorID)
	if err != nil {
		h.Logger.Error("failed to get lessons short info", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "rights") {
			errStatus = http.StatusForbidden
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, lessons)
}

// CreateLessonBlock handles POST /tutor/blocks
// @Summary Create a lesson block
// @Description Create a new lesson block in a lesson owned by the authenticated tutor
// @Tags tutor
// @Accept json
// @Produce json
// @Param request body models.CreateLessonBlockRequest true "Lesson block creation request"
// @Success 201 {object} map[string]any "Lesson block created successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - lesson does not belong to tutor"
// @Failure 404 {object} map[string]string "Lesson not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/blocks [post]
func (h *TutorLessonHandler) CreateLessonBlock(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req models.CreateLessonBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	blockID, err := h.service.CreateLessonBlock(r.Context(), &tutorID, &req)
	if err != nil {
		h.Logger.Error("failed to create lesson block", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		} else if strings.Contains(err.Error(), "rights") {
			errStatus = http.StatusForbidden
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]any{
		"id":      blockID,
		"message": "lesson block created successfully",
	})
}

// UpdateLessonBlock handles PATCH /tutor/blocks/{id}
// @Summary Update a lesson block
// @Description Update a lesson block in a lesson owned by the authenticated tutor (partial update)
// @Tags tutor
// @Accept json
// @Produce json
// @Param id path int true "Block ID"
// @Param request body models.UpdateLessonBlockRequest true "Lesson block update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not block owner or not found"
// @Router /tutor/blocks/{id} [patch]
func (h *TutorLessonHandler) UpdateLessonBlock(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	blockIDStr := chi.URLParam(r, "id")
	blockID, err := strconv.Atoi(blockIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid block ID")
		return
	}

	var req models.UpdateLessonBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.service.UpdateLessonBlock(r.Context(), blockID, &tutorID, &req)
	if err != nil {
		h.Logger.Error("failed to update lesson block", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "rights") {
			errStatus = http.StatusForbidden
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteBlock handles DELETE /tutor/blocks/{id}
// @Summary Delete a lesson block
// @Description Delete a lesson block in a lesson owned by the authenticated tutor
// @Tags tutor
// @Accept json
// @Produce json
// @Param id path int true "Block ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid block ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not block owner or not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/blocks/{id} [delete]
func (h *TutorLessonHandler) DeleteBlock(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	blockIDStr := chi.URLParam(r, "id")
	blockID, err := strconv.Atoi(blockIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid block ID")
		return
	}

	err = h.service.DeleteBlock(r.Context(), blockID, &tutorID)
	if err != nil {
		h.Logger.Error("failed to delete block", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "rights") {
			errStatus = http.StatusForbidden
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTutorMedia handles GET /tutor/media
// @Summary Get tutor media
// @Description Get paginated list of media files for the authenticated tutor with optional media type filter
// @Tags tutor
// @Accept json
// @Produce json
// @Param mediaType query string false "Media type (video, audio, doc)"
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 10)"
// @Success 200 {array} models.TutorMediaResponse "List of tutor media"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/media [get]
func (h *TutorLessonHandler) GetTutorMedia(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	mediaTypeStr := r.URL.Query().Get("mediaType")
	pageStr := r.URL.Query().Get("page")
	countStr := r.URL.Query().Get("count")

	var mediaType *models.MediaType
	if mediaTypeStr != "" {
		mt := models.MediaType(mediaTypeStr)
		mediaType = &mt
	}

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

	media, err := h.service.GetTutorMedia(r.Context(), &tutorID, mediaType, page, count)
	if err != nil {
		h.Logger.Error("failed to get tutor media", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, media)
}

// CreateTutorMedia handles POST /tutor/media
// @Summary Create tutor media
// @Description Upload a new media file for the authenticated tutor
// @Tags tutor
// @Accept multipart/form-data
// @Produce json
// @Param slug formData string true "Media slug"
// @Param mediaType formData string true "Media type (video, audio, doc)"
// @Param file formData file true "Media file"
// @Success 201 {object} map[string]any "Tutor media created successfully"
// @Failure 400 {object} map[string]string "Invalid request or file missing"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/media [post]
func (h *TutorLessonHandler) CreateTutorMedia(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Parse multipart form (limit to 30MB for video)
	const maxMemory = 30 << 20 // 30MB
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	req := models.CreateTutorMediaRequest{
		TutorID:   tutorID,
		Slug:      r.FormValue("slug"),
		MediaType: models.MediaType(r.FormValue("mediaType")),
	}

	var file multipart.File
	var filename string
	file, fileHeader, err := r.FormFile("file")
	if err == nil && file != nil && fileHeader.Size > 0 {
		defer file.Close()
		filename = fileHeader.Filename
	} else if err != http.ErrMissingFile || fileHeader.Size == 0 {
		h.Logger.Error("failed to get file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to get file")
		return
	}

	mediaID, err := h.service.CreateTutorMedia(r.Context(), &req, file, filename)
	if err != nil {
		h.Logger.Error("failed to create tutor media", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]any{
		"id":      mediaID,
		"message": "tutor media created successfully",
	})
}

// DeleteTutorMedia handles DELETE /tutor/media/{id}
// @Summary Delete tutor media
// @Description Delete a media file owned by the authenticated tutor
// @Tags tutor
// @Accept json
// @Produce json
// @Param id path int true "Media ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid media ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - not media owner or permission denied"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tutor/media/{id} [delete]
func (h *TutorLessonHandler) DeleteTutorMedia(w http.ResponseWriter, r *http.Request) {
	tutorID, err := h.getTutorID(r)
	if err != nil {
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	mediaIDStr := chi.URLParam(r, "id")
	mediaID, err := strconv.Atoi(mediaIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid media ID")
		return
	}

	err = h.service.DeleteTutorMedia(r.Context(), mediaID, &tutorID)
	if err != nil {
		h.Logger.Error("failed to delete tutor media", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "permission") {
			errStatus = http.StatusForbidden
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
