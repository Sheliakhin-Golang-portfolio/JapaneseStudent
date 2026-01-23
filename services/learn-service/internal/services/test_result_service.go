package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/learn-service/internal/models"
)

// CharacterLearnHistoryRepository is the interface that wraps methods for CharacterLearnHistory table data access
type CharacterLearnHistoryRepository interface {
	// Method GetByUserIDAndCharacterIDs retrieves learn history records for a user and set of character IDs.
	//
	// "characterIDs" parameter is used to identify the characters.
	//
	// Please reference GetByUserID method for more information about other parameters and error values.
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
	// LowerResultsByUserID lowers all result values by 0.01 for all CharacterLearnHistory records for a user
	//
	// "userID" parameter is used to identify the user.
	// If some error occurs during the update, the error will be returned.
	LowerResultsByUserID(ctx context.Context, userID int) error
}

// TestResultCharactersRepository is the interface for test result character repository (needed for GetTotalCount)
type TestResultCharactersRepository interface {
	// Method GetTotalCount returns the total number of characters in the database
	//
	// If some error occurs during data retrieval, the error will be returned together with "nil" value.
	GetTotalCount(ctx context.Context) (int, error)
}

// testResultService implements TestResultService
type testResultService struct {
	historyRepo CharacterLearnHistoryRepository
	charRepo    TestResultCharactersRepository
}

// NewTestResultService creates a new test result service
func NewTestResultService(historyRepo CharacterLearnHistoryRepository, charRepo TestResultCharactersRepository) *testResultService {
	return &testResultService{
		historyRepo: historyRepo,
		charRepo:    charRepo,
	}
}

// SubmitTestResults processes and saves test results
//
// For successful results alphabetType must be either "hiragana" or "katakana".
// testType must be either "reading", "writing", or "listening".
// results must be a non-empty array of TestResultItem.
// repeat parameter indicates if user wants to repeat alphabet after completing all characters ("in question" by default).
//
// Returns result with askForRepeat flag and error.
// If any of the parameters are invalid, or some error occurs during data processing, the error will be returned.
func (s *testResultService) SubmitTestResults(ctx context.Context, userID int, alphabetType, testType string, results []models.TestResultItem, repeat string) (*models.SubmitTestResultsResult, error) {
	// Set default repeat value if not provided
	if repeat == "" {
		repeat = "in question"
	}

	// Validate alphabet type
	alphabetTypeLower := strings.ToLower(alphabetType)
	if alphabetTypeLower != "hiragana" && alphabetTypeLower != "katakana" {
		return nil, fmt.Errorf("invalid alphabet type, must be 'hiragana' or 'katakana'")
	}

	// Validate test type
	testTypeLower := strings.ToLower(testType)
	if testTypeLower != "reading" && testTypeLower != "writing" && testTypeLower != "listening" {
		return nil, fmt.Errorf("invalid test type, must be 'reading', 'writing', or 'listening'")
	}

	// Extract character IDs
	characterIDs := make([]int, len(results))
	for i, result := range results {
		characterIDs[i] = result.CharacterID
	}

	// Get existing records
	existingHistories, err := s.historyRepo.GetByUserIDAndCharacterIDs(ctx, userID, characterIDs)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("invalid alphabet type or test type")
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

	// Upsert the results
	if err := s.historyRepo.Upsert(ctx, toUpdate); err != nil {
		return nil, err
	}

	// Check if user has maximum marks for all characters (only if repeat is "in question")
	askForRepeat := false
	if repeat == "in question" {
		allCompleted, err := s.checkAllCharactersCompleted(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to check character completion: %w", err)
		}
		askForRepeat = allCompleted
	}

	return &models.SubmitTestResultsResult{
		AskForRepeat: askForRepeat,
	}, nil
}

// GetUserHistory retrieves all learn history records for a user
func (s *testResultService) GetUserHistory(ctx context.Context, userID int) ([]models.UserLearnHistory, error) {
	return s.historyRepo.GetByUserID(ctx, userID)
}

// checkAllCharactersCompleted checks if user has maximum marks for all characters in all categories
func (s *testResultService) checkAllCharactersCompleted(ctx context.Context, userID int) (bool, error) {
	// Get total character count
	totalCharacters, err := s.charRepo.GetTotalCount(ctx)
	if err != nil {
		return false, err
	}

	// Get all user's character learn history records
	histories, err := s.historyRepo.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}

	// Calculate total sum of all result categories for all characters
	var totalSum float32 = 0
	for _, history := range histories {
		totalSum += history.HiraganaReadingResult +
			history.HiraganaWritingResult +
			history.HiraganaListeningResult +
			history.KatakanaReadingResult +
			history.KatakanaWritingResult +
			history.KatakanaListeningResult
	}

	// Maximum possible sum = total characters * 6 (one for each category)
	expectedMax := float32(totalCharacters) * 6.0

	// Check if total sum equals expected maximum (with small tolerance for floating point)
	const tolerance float32 = 0.001
	return totalSum >= expectedMax-tolerance && totalSum <= expectedMax+tolerance, nil
}

// DropUserMarks lowers all CharacterLearnHistory results by 0.01 for a user
func (s *testResultService) DropUserMarks(ctx context.Context, userID int) error {
	return s.historyRepo.LowerResultsByUserID(ctx, userID)
}
