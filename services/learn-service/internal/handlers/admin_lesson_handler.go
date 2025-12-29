package handlers

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// AdminLessonHandler handles HTTP requests for admin lesson operations
type AdminLessonHandler struct {
	handlers.BaseHandler
	tutorLessonService TutorLessonService
}

// NewAdminLessonHandler creates a new admin lesson handler
func NewAdminLessonHandler(tutorLessonService TutorLessonService, logger *zap.Logger) *AdminLessonHandler {
	return &AdminLessonHandler{
		tutorLessonService: tutorLessonService,
		BaseHandler:        handlers.BaseHandler{Logger: logger},
	}
}

// RegisterRoutes registers all admin lesson handler routes
func (h *AdminLessonHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
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

// GetCourses handles GET /admin/courses
// @Summary Get list of courses
// @Description Get paginated list of courses with optional tutor ID, complexity level and search filters
// @Tags admin
// @Accept json
// @Produce json
// @Param tutorId query int false "Filter by tutor ID"
// @Param complexityLevel query string false "Complexity level (ab, b, i, ui, a) or full name"
// @Param search query string false "Search query"
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 10)"
// @Success 200 {array} models.CourseListItem "List of courses"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/courses [get]
func (h *AdminLessonHandler) GetCourses(w http.ResponseWriter, r *http.Request) {
	tutorIDStr := r.URL.Query().Get("tutorId")
	complexityLevelStr := r.URL.Query().Get("complexityLevel")
	search := r.URL.Query().Get("search")
	pageStr := r.URL.Query().Get("page")
	countStr := r.URL.Query().Get("count")

	var tutorID *int
	if tutorIDStr != "" {
		id, err := strconv.Atoi(tutorIDStr)
		if err != nil {
			h.RespondError(w, http.StatusBadRequest, "invalid tutor ID")
			return
		}
		tutorID = &id
	}
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

	courses, err := h.tutorLessonService.GetCourses(r.Context(), tutorID, complexityLevel, search, page, count)
	if err != nil {
		h.Logger.Error("failed to get courses", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, courses)
}

// CreateCourse handles POST /admin/courses
// @Summary Create a course
// @Description Create a new course (admin can create for any tutor)
// @Tags admin
// @Accept json
// @Produce json
// @Param request body models.CreateCourseRequest true "Course creation request"
// @Success 201 {object} map[string]any "Course created successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/courses [post]
func (h *AdminLessonHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	courseID, err := h.tutorLessonService.CreateCourse(r.Context(), &req)
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

// UpdateCourse handles PATCH /admin/courses/{id}
// @Summary Update a course
// @Description Update a course (partial update, admin can update any course)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Param request body models.UpdateCourseRequest true "Course update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 404 {object} map[string]string "Course not found"
// @Router /admin/courses/{id} [patch]
func (h *AdminLessonHandler) UpdateCourse(w http.ResponseWriter, r *http.Request) {
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

	err = h.tutorLessonService.UpdateCourse(r.Context(), courseID, nil, &req)
	if err != nil {
		h.Logger.Error("failed to update course", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteCourse handles DELETE /admin/courses/{id}
// @Summary Delete a course
// @Description Delete a course (admin can delete any course)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid course ID"
// @Failure 404 {object} map[string]string "Course not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/courses/{id} [delete]
func (h *AdminLessonHandler) DeleteCourse(w http.ResponseWriter, r *http.Request) {
	courseIDStr := chi.URLParam(r, "id")
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid course ID")
		return
	}

	err = h.tutorLessonService.DeleteCourse(r.Context(), courseID, nil)
	if err != nil {
		h.Logger.Error("failed to delete course", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCoursesShortInfo handles GET /admin/courses/short
// @Summary Get courses short info
// @Description Get list of all courses with only ID and title (for select options)
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {array} models.CourseShortInfo "List of courses short info"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/courses/short [get]
func (h *AdminLessonHandler) GetCoursesShortInfo(w http.ResponseWriter, r *http.Request) {
	courses, err := h.tutorLessonService.GetCoursesShortInfo(r.Context(), nil)
	if err != nil {
		h.Logger.Error("failed to get courses short info", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, courses)
}

// GetLessonsForCourse handles GET /admin/courses/{id}/lessons
// @Summary Get lessons for a course
// @Description Get course details and list of lessons for any course
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Success 200 {object} map[string]any "Course with lessons"
// @Failure 400 {object} map[string]string "Invalid course ID"
// @Failure 404 {object} map[string]string "Course not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/courses/{id}/lessons [get]
func (h *AdminLessonHandler) GetLessonsForCourse(w http.ResponseWriter, r *http.Request) {
	courseIDStr := chi.URLParam(r, "id")
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid course ID")
		return
	}

	course, lessons, err := h.tutorLessonService.GetLessonsForCourse(r.Context(), courseID, nil)
	if err != nil {
		h.Logger.Error("failed to get lessons for course", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
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

// CreateLesson handles POST /admin/lessons
// @Summary Create a lesson
// @Description Create a new lesson in any course (admin can create for any course)
// @Tags admin
// @Accept json
// @Produce json
// @Param request body models.CreateLessonRequest true "Lesson creation request"
// @Success 201 {object} map[string]any "Lesson created successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 404 {object} map[string]string "Course not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/lessons [post]
func (h *AdminLessonHandler) CreateLesson(w http.ResponseWriter, r *http.Request) {
	var req models.CreateLessonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	lessonID, err := h.tutorLessonService.CreateLesson(r.Context(), nil, &req)
	if err != nil {
		h.Logger.Error("failed to create lesson", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]any{
		"id":      lessonID,
		"message": "lesson created successfully",
	})
}

// UpdateLesson handles PATCH /admin/lessons/{id}
// @Summary Update a lesson
// @Description Update a lesson (partial update, admin can update any lesson)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Lesson ID"
// @Param request body models.UpdateLessonRequest true "Lesson update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 404 {object} map[string]string "Lesson not found"
// @Router /admin/lessons/{id} [patch]
func (h *AdminLessonHandler) UpdateLesson(w http.ResponseWriter, r *http.Request) {
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

	err = h.tutorLessonService.UpdateLesson(r.Context(), lessonID, nil, &req)
	if err != nil {
		h.Logger.Error("failed to update lesson", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteLesson handles DELETE /admin/lessons/{id}
// @Summary Delete a lesson
// @Description Delete a lesson (admin can delete any lesson)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Lesson ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid lesson ID"
// @Failure 404 {object} map[string]string "Lesson not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/lessons/{id} [delete]
func (h *AdminLessonHandler) DeleteLesson(w http.ResponseWriter, r *http.Request) {
	lessonIDStr := chi.URLParam(r, "id")
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid lesson ID")
		return
	}

	err = h.tutorLessonService.DeleteLesson(r.Context(), lessonID, nil)
	if err != nil {
		h.Logger.Error("failed to delete lesson", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetFullLessonInfo handles GET /admin/lessons/{id}
// @Summary Get full lesson info
// @Description Get lesson details and list of lesson blocks for any lesson
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Lesson ID"
// @Success 200 {object} map[string]any "Lesson with blocks"
// @Failure 400 {object} map[string]string "Invalid lesson ID"
// @Failure 404 {object} map[string]string "Lesson not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/lessons/{id} [get]
func (h *AdminLessonHandler) GetFullLessonInfo(w http.ResponseWriter, r *http.Request) {
	lessonIDStr := chi.URLParam(r, "id")
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid lesson ID")
		return
	}

	lesson, blocks, err := h.tutorLessonService.GetFullLessonInfo(r.Context(), lessonID, nil)
	if err != nil {
		h.Logger.Error("failed to get full lesson info", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
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

// GetLessonsShortInfo handles GET /admin/lessons/short
// @Summary Get lessons short info
// @Description Get list of lessons with only ID and title for a course (optional, if course ID not provided, the lessons retrieved for all courses)
// @Tags admin
// @Accept json
// @Produce json
// @Param courseId query int false "Course ID"
// @Success 200 {array} models.LessonShortInfo "List of lessons short info"
// @Failure 400 {object} map[string]string "Invalid course ID or missing courseId parameter"
// @Failure 404 {object} map[string]string "Course not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/lessons/short [get]
func (h *AdminLessonHandler) GetLessonsShortInfo(w http.ResponseWriter, r *http.Request) {
	courseIDStr := r.URL.Query().Get("courseId")
	var courseID *int
	if courseIDStr != "" {
		id, err := strconv.Atoi(courseIDStr)
		if err != nil {
			h.RespondError(w, http.StatusBadRequest, "invalid course ID")
			return
		}
		courseID = &id
	}

	lessons, err := h.tutorLessonService.GetLessonsShortInfo(r.Context(), courseID, nil)
	if err != nil {
		h.Logger.Error("failed to get lessons short info", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, lessons)
}

// CreateLessonBlock handles POST /admin/blocks
// @Summary Create a lesson block
// @Description Create a new lesson block in any lesson (admin can create for any lesson)
// @Tags admin
// @Accept json
// @Produce json
// @Param request body models.CreateLessonBlockRequest true "Lesson block creation request"
// @Success 201 {object} map[string]interface{} "Lesson block created successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 404 {object} map[string]string "Lesson not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/blocks [post]
func (h *AdminLessonHandler) CreateLessonBlock(w http.ResponseWriter, r *http.Request) {
	var req models.CreateLessonBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	blockID, err := h.tutorLessonService.CreateLessonBlock(r.Context(), nil, &req)
	if err != nil {
		h.Logger.Error("failed to create lesson block", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      blockID,
		"message": "lesson block created successfully",
	})
}

// UpdateLessonBlock handles PATCH /admin/blocks/{id}
// @Summary Update a lesson block
// @Description Update a lesson block (partial update, admin can update any block)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Block ID"
// @Param request body models.UpdateLessonBlockRequest true "Lesson block update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 404 {object} map[string]string "Block not found"
// @Router /admin/blocks/{id} [patch]
func (h *AdminLessonHandler) UpdateLessonBlock(w http.ResponseWriter, r *http.Request) {
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

	err = h.tutorLessonService.UpdateLessonBlock(r.Context(), blockID, nil, &req)
	if err != nil {
		h.Logger.Error("failed to update lesson block", zap.Error(err))
		errStatus := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteBlock handles DELETE /admin/blocks/{id}
// @Summary Delete a lesson block
// @Description Delete a lesson block (admin can delete any block)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Block ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid block ID"
// @Failure 404 {object} map[string]string "Block not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/blocks/{id} [delete]
func (h *AdminLessonHandler) DeleteBlock(w http.ResponseWriter, r *http.Request) {
	blockIDStr := chi.URLParam(r, "id")
	blockID, err := strconv.Atoi(blockIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid block ID")
		return
	}

	err = h.tutorLessonService.DeleteBlock(r.Context(), blockID, nil)
	if err != nil {
		h.Logger.Error("failed to delete block", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTutorMedia handles GET /admin/media
// @Summary Get tutor media
// @Description Get paginated list of media files with optional tutor ID and media type filters
// @Tags admin
// @Accept json
// @Produce json
// @Param tutorId query int false "Filter by tutor ID"
// @Param mediaType query string false "Media type (video, audio, doc)"
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 10)"
// @Success 200 {array} models.TutorMediaResponse "List of tutor media"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/media [get]
func (h *AdminLessonHandler) GetTutorMedia(w http.ResponseWriter, r *http.Request) {
	tutorIDStr := r.URL.Query().Get("tutorId")
	mediaTypeStr := r.URL.Query().Get("mediaType")
	pageStr := r.URL.Query().Get("page")
	countStr := r.URL.Query().Get("count")

	var tutorID *int
	if tutorIDStr != "" {
		id, err := strconv.Atoi(tutorIDStr)
		if err != nil {
			h.RespondError(w, http.StatusBadRequest, "invalid tutor ID")
			return
		}
		tutorID = &id
	}
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

	media, err := h.tutorLessonService.GetTutorMedia(r.Context(), tutorID, mediaType, page, count)
	if err != nil {
		h.Logger.Error("failed to get tutor media", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, media)
}

// CreateTutorMedia handles POST /admin/media
// @Summary Create tutor media
// @Description Upload a new media file for a tutor (admin can create for any tutor)
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param tutorId formData int true "Tutor ID"
// @Param slug formData string true "Media slug"
// @Param mediaType formData string true "Media type (video, audio, doc)"
// @Param file formData file true "Media file"
// @Success 201 {object} map[string]any "Tutor media created successfully"
// @Failure 400 {object} map[string]string "Invalid request or file missing"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/media [post]
func (h *AdminLessonHandler) CreateTutorMedia(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 30 << 20 // 30MB
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	tutorIDStr := r.FormValue("tutorId")
	tutorID, err := strconv.Atoi(tutorIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid tutor ID")
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

	mediaID, err := h.tutorLessonService.CreateTutorMedia(r.Context(), &req, file, filename)
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

// DeleteTutorMedia handles DELETE /admin/media/{id}
// @Summary Delete tutor media
// @Description Delete a media file (admin can delete any media)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Media ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid media ID"
// @Failure 404 {object} map[string]string "Media not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/media/{id} [delete]
func (h *AdminLessonHandler) DeleteTutorMedia(w http.ResponseWriter, r *http.Request) {
	mediaIDStr := chi.URLParam(r, "id")
	mediaID, err := strconv.Atoi(mediaIDStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid media ID")
		return
	}

	err = h.tutorLessonService.DeleteTutorMedia(r.Context(), mediaID, nil)
	if err != nil {
		h.Logger.Error("failed to delete tutor media", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
