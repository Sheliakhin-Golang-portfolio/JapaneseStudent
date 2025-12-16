package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/japanesestudent/learn-service/internal/models"
	"go.uber.org/zap"
)

// CharacterLearnHistoryRepository is the interface that wraps methods for CharacterLearnHistory table data access
type CharacterLearnHistoryRepository interface {
	// Method GetByUserIDAndCharacterIDs retrieves learn history records for a user and set of character IDs.
	//
	// "userID" parameter is used to identify the user.
	// "characterIDs" parameter is used to identify the characters.
	// If no records are found, an empty slice will be returned.
	// If some error occurs during data retrieval, the error will be returned.
	GetByUserIDAndCharacterIDs(ctx context.Context, userID int, characterIDs []int) ([]models.CharacterLearnHistory, error)
	// Method GetByUserID retrieves all learn history records for a user.
	//
	// "userID" parameter is used to identify the user.
	// If no records are found, an empty slice will be returned.
	// If some error occurs during data retrieval, the error will be returned.
	GetByUserID(ctx context.Context, userID int) ([]models.UserLearnHistory, error)
	// Method Upsert updates or creates a list of learn history records.
	//
	// "histories" parameter is used to update or create a list of learn history records.
	// If some error occurs during data upsert, the error will be returned.
	Upsert(ctx context.Context, histories []models.CharacterLearnHistory) error
}

// testResultService implements TestResultService
type testResultService struct {
	historyRepo CharacterLearnHistoryRepository
	logger      *zap.Logger
}

// NewTestResultService creates a new test result service
func NewTestResultService(historyRepo CharacterLearnHistoryRepository, logger *zap.Logger) *testResultService {
	return &testResultService{
		historyRepo: historyRepo,
		logger:      logger,
	}
}

// SubmitTestResults processes and saves test results
//
// For successful results alphabetType must be either "hiragana" or "katakana".
// testType must be either "reading", "writing", or "listening".
// results must be a non-empty array of TestResultItem.
// If any of the parameters are invalid, or some error occurs during data processing, the error will be returned.
func (s *testResultService) SubmitTestResults(ctx context.Context, userID int, alphabetType, testType string, results []models.TestResultItem) error {
	// Validate alphabet type
	alphabetTypeLower := strings.ToLower(alphabetType)
	if alphabetTypeLower != "hiragana" && alphabetTypeLower != "katakana" {
		return fmt.Errorf("invalid alphabet type, must be 'hiragana' or 'katakana'")
	}

	// Validate test type
	testTypeLower := strings.ToLower(testType)
	if testTypeLower != "reading" && testTypeLower != "writing" && testTypeLower != "listening" {
		return fmt.Errorf("invalid test type, must be 'reading', 'writing', or 'listening'")
	}

	// Extract character IDs
	characterIDs := make([]int, len(results))
	for i, result := range results {
		characterIDs[i] = result.CharacterID
	}

	// Get existing records
	existingHistories, err := s.historyRepo.GetByUserIDAndCharacterIDs(ctx, userID, characterIDs)
	if err != nil {
		return fmt.Errorf("failed to get existing histories: %w", err)
	}

	// Create a map of existing histories by character ID
	existingMap := make(map[int]*models.CharacterLearnHistory)
	for i := range existingHistories {
		existingMap[existingHistories[i].CharacterID] = &existingHistories[i]
	}

	// Determine which column to update based on alphabet type and test type
	var updateField func(*models.CharacterLearnHistory, float32)

	switch {
	case alphabetTypeLower == "hiragana" && testTypeLower == "reading":
		updateField = func(h *models.CharacterLearnHistory, val float32) { h.HiraganaReadingResult = val }
	case alphabetTypeLower == "hiragana" && testTypeLower == "writing":
		updateField = func(h *models.CharacterLearnHistory, val float32) { h.HiraganaWritingResult = val }
	case alphabetTypeLower == "hiragana" && testTypeLower == "listening":
		updateField = func(h *models.CharacterLearnHistory, val float32) { h.HiraganaListeningResult = val }
	case alphabetTypeLower == "katakana" && testTypeLower == "reading":
		updateField = func(h *models.CharacterLearnHistory, val float32) { h.KatakanaReadingResult = val }
	case alphabetTypeLower == "katakana" && testTypeLower == "writing":
		updateField = func(h *models.CharacterLearnHistory, val float32) { h.KatakanaWritingResult = val }
	case alphabetTypeLower == "katakana" && testTypeLower == "listening":
		updateField = func(h *models.CharacterLearnHistory, val float32) { h.KatakanaListeningResult = val }
	default:
		return fmt.Errorf("invalid alphabet type or test type")
	}

	// Prepare histories for batch insert or update
	var toUpdate []models.CharacterLearnHistory

	for _, result := range results {
		var resultValue float32 = 0
		if result.Passed {
			resultValue = 1
		}

		if existing, ok := existingMap[result.CharacterID]; ok {
			// Update existing record
			updateField(existing, resultValue)
			toUpdate = append(toUpdate, *existing)
		} else {
			// Create new record
			history := models.CharacterLearnHistory{
				UserID:      userID,
				CharacterID: result.CharacterID,
			}
			updateField(&history, resultValue)
			toUpdate = append(toUpdate, history)
		}
	}

	return s.historyRepo.Upsert(ctx, toUpdate)
}

// GetUserHistory retrieves all learn history records for a user
func (s *testResultService) GetUserHistory(ctx context.Context, userID int) ([]models.UserLearnHistory, error) {
	return s.historyRepo.GetByUserID(ctx, userID)
}
