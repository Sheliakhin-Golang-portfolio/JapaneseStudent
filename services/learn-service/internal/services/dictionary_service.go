package services

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/japanesestudent/learn-service/internal/models"
	"go.uber.org/zap"
)

// WordRepository is the interface that wraps methods for Word table data access
type WordRepository interface {
	// GetByIDs retrieves words by their IDs
	//
	// "wordIds" parameter is used to filter words by their IDs.
	// "translationField" parameter is used to specify the field to use for translation.
	// "exampleTranslationField" parameter is used to specify the field to use for example translation.
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetByIDs(ctx context.Context, wordIds []int, translationField, exampleTranslationField string) ([]models.WordResponse, error)
	// GetExcludingIDs retrieves words not in the provided ID list
	//
	// "excludeIds" parameter is used to filter words not in the provided ID list.
	// "limit" parameter is used to specify the number of words to return.
	// "translationField" parameter is used to specify the field to use for translation.
	// "exampleTranslationField" parameter is used to specify the field to use for example translation.
	// Please reference GetByIDs method for more information about other parameters and error values.
	GetExcludingIDs(ctx context.Context, excludeIds []int, limit int, translationField, exampleTranslationField string) ([]models.WordResponse, error)
	// ValidateWordIDs checks if all word IDs exist in the database
	//
	// "wordIds" parameter is used to validate if all word IDs exist in the database.
	// Please reference GetByIDs method for more information about other parameters and error values.
	ValidateWordIDs(ctx context.Context, wordIds []int) (bool, error)
}

// DictionaryHistoryRepository is the interface that wraps methods for DictionaryHistory table data access
type DictionaryHistoryRepository interface {
	// GetOldWordIds retrieves word IDs from dictionary history where NextAppearance <= current day
	//
	// "userId" parameter is used to identify the user.
	// "limit" parameter is used to specify the number of words to return.
	// Please reference GetByIDs method for more information about other parameters and error values.
	GetOldWordIds(ctx context.Context, userId int, limit int) ([]int, error)
	// UpsertResults inserts or updates dictionary history records
	//
	// "userId" parameter is used to identify the user.
	// "results" parameter is used to submit the word learning results.
	// Please reference GetByIDs method for more information about other parameters and error values.
	UpsertResults(ctx context.Context, userId int, results []models.WordResult) error
}

// dictionaryService implements DictionaryService
type dictionaryService struct {
	wordRepo              WordRepository
	dictionaryHistoryRepo DictionaryHistoryRepository
	logger                *zap.Logger
}

// NewDictionaryService creates a new dictionary service
func NewDictionaryService(
	wordRepo WordRepository,
	dictionaryHistoryRepo DictionaryHistoryRepository,
	logger *zap.Logger,
) *dictionaryService {
	return &dictionaryService{
		wordRepo:              wordRepo,
		dictionaryHistoryRepo: dictionaryHistoryRepo,
		logger:                logger,
	}
}

// GetWordList retrieves a mixed list of old and new words for the user
//
// Please reference validateParameters method for more information about parameters and error values.
func (s *dictionaryService) GetWordList(ctx context.Context, userId, newCount, oldCount int, locale string) ([]models.WordResponse, error) {
	if err := s.validateParameters(newCount, oldCount, locale); err != nil {
		s.logger.Error("failed to validate word list", zap.Error(err))
		return nil, err
	}

	// Get old word IDs from dictionary history
	oldWordIds, err := s.dictionaryHistoryRepo.GetOldWordIds(ctx, userId, oldCount)
	if err != nil {
		s.logger.Error("failed to get old word IDs", zap.Error(err))
		return nil, fmt.Errorf("failed to get old word IDs: %w", err)
	}

	// Set locale-specific translations
	var translationField, exampleTranslationField string
	switch locale {
	case "ru":
		translationField = "russian_translation"
		exampleTranslationField = "example_russian_translation"
	case "de":
		translationField = "english_translation"
		exampleTranslationField = "example_english_translation"
	default: // "en" or default
		translationField = "german_translation"
		exampleTranslationField = "example_german_translation"
	}

	return s.getShuffledWordList(ctx, oldWordIds, newCount, translationField, exampleTranslationField)
}

// validateParameters validates the parameters for the GetWordList method
//
// For successful results:
//
// - newCount and oldCount must be between 10 and 40
//
// - locale must be "en", "ru", or "de"
//
// If wrong parameters will be used or some error will occur during validation, the error will be returned.
func (s *dictionaryService) validateParameters(newCount, oldCount int, locale string) error {
	errChan := make(chan error, 3)
	// Validate counts
	go func() {
		if newCount < 10 || newCount > 40 {
			errChan <- fmt.Errorf("newWordCount must be between 10 and 40")
			return
		}
		errChan <- nil
	}()
	go func() {
		if oldCount < 10 || oldCount > 40 {
			errChan <- fmt.Errorf("oldWordCount must be between 10 and 40")
			return
		}
		errChan <- nil
	}()

	// Validate locale
	go func() {
		if locale != "en" && locale != "ru" && locale != "de" {
			errChan <- fmt.Errorf("invalid locale: %s, must be 'en', 'ru', or 'de'", locale)
			return
		}
		errChan <- nil
	}()

	for range 3 {
		err := <-errChan
		if err != nil {
			s.logger.Error("failed to validate word list", zap.Error(err))
			return err
		}
	}
	return nil
}

// getShuffledWordList concurrently gets a shuffled list of old and new words for the user
func (s *dictionaryService) getShuffledWordList(ctx context.Context, oldWordIds []int, newCount int, translationField, exampleTranslationField string) ([]models.WordResponse, error) {
	// Prepare for concurrent operations
	var allWords []models.WordResponse
	wordsChan := make(chan []models.WordResponse, 2)
	wordsErrChan := make(chan error, 1)

	// Get old words
	go func() {
		if len(oldWordIds) == 0 {
			wordsChan <- []models.WordResponse{}
			return
		}
		words, err := s.wordRepo.GetByIDs(ctx, oldWordIds, translationField, exampleTranslationField)
		if err != nil {
			s.logger.Error("failed to get old words", zap.Error(err))
			wordsErrChan <- err
			return
		}
		wordsChan <- words
	}()

	// Get new words (not in old word IDs list)
	go func() {
		words, err := s.wordRepo.GetExcludingIDs(ctx, oldWordIds, newCount, translationField, exampleTranslationField)
		if err != nil {
			s.logger.Error("failed to get new words", zap.Error(err))
			wordsErrChan <- err
			return
		}
		wordsChan <- words
	}()

	// Combine and shuffle
	for range 2 {
		select {
		case words := <-wordsChan:
			allWords = append(allWords, words...)
		case err := <-wordsErrChan:
			s.logger.Error("failed to get words", zap.Error(err))
			return nil, err
		}
	}
	rand.Shuffle(len(allWords), func(i, j int) {
		allWords[i], allWords[j] = allWords[j], allWords[i]
	})
	return allWords, nil
}

// SubmitWordResults validates and upserts word learning results
//
// For successful results:
//
// - All word IDs must exist in the Word table
//
// - Period values must be between 1 and 30
func (s *dictionaryService) SubmitWordResults(ctx context.Context, userId int, results []models.WordResult) error {
	if len(results) == 0 {
		return fmt.Errorf("results list cannot be empty")
	}

	// Extract word IDs for validation
	wordIds := make([]int, len(results))
	for i, result := range results {
		wordIds[i] = result.WordID
		// Validate period
		if result.Period < 1 || result.Period > 30 {
			return fmt.Errorf("period must be between 1 and 30, got: %d", result.Period)
		}
	}

	// Validate all word IDs exist
	valid, err := s.wordRepo.ValidateWordIDs(ctx, wordIds)
	if err != nil {
		s.logger.Error("failed to validate word IDs", zap.Error(err))
		return fmt.Errorf("failed to validate word IDs: %w", err)
	}
	if !valid {
		return fmt.Errorf("one or more word IDs do not exist")
	}

	// Upsert results
	return s.dictionaryHistoryRepo.UpsertResults(ctx, userId, results)
}
