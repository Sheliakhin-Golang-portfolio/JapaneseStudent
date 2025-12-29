package models

import "encoding/json"

// BlockType represents the type of a lesson block
type BlockType string

const (
	BlockTypeVideo    BlockType = "video"
	BlockTypeAudio    BlockType = "audio"
	BlockTypeText     BlockType = "text"
	BlockTypeDocument BlockType = "document"
	BlockTypeList     BlockType = "list"
)

// LessonBlock represents a block within a lesson
type LessonBlock struct {
	ID         int             `json:"id"`
	LessonID   int             `json:"lessonId"`
	BlockType  BlockType       `json:"blockType"`
	BlockOrder int             `json:"blockOrder"`
	BlockData  json.RawMessage `json:"blockData"`
}

// LessonBlockResponse represents a lesson block in API responses
type LessonBlockResponse struct {
	ID         int             `json:"id,omitempty"`
	BlockType  BlockType       `json:"blockType"`
	BlockOrder int             `json:"blockOrder"`
	BlockData  json.RawMessage `json:"blockData"`
}

// CreateLessonBlockRequest represents a request to create a lesson block
type CreateLessonBlockRequest struct {
	LessonID   int             `json:"lessonId" example:"1"`
	BlockType  BlockType       `json:"blockType" example:"video"`
	BlockOrder int             `json:"blockOrder" example:"1"`
	BlockData  json.RawMessage `json:"blockData" example:"{\"video\": \"video_url\"}"`
}

// UpdateLessonBlockRequest represents a request to update a lesson block (partial update)
type UpdateLessonBlockRequest struct {
	LessonID   *int             `json:"lessonId,omitempty" example:"1"`
	BlockType  BlockType        `json:"blockType,omitempty" example:"video"`
	BlockOrder *int             `json:"blockOrder,omitempty" example:"1"`
	BlockData  *json.RawMessage `json:"blockData,omitempty" example:"{\"video\": \"video_url\"}"`
}
