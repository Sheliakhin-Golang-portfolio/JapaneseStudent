package services

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/learn-service/internal/models"
)

// AdminWordRepository is the interface that wraps methods for Word table data access
type AdminWordRepository interface {
	// GetAllForAdmin retrieves a paginated list of words for admin endpoints
	//
	// "page" parameter is used to specify the page number.
	// "count" parameter is used to specify the number of items per page.
	// "search" parameter is used to search words by word, phonetic clues, or translations.

	// // If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetAllForAdmin(ctx context.Context, page, count int, search string) ([]models.Word, error)
	// Method GetByIDAdmin retrieve a word by its ID using configured repository.
	//
	// "id" parameter is used to identify the word.
	//
	// If some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetByIDAdmin(ctx context.Context, id int) (*models.Word, error)
	// Method ExistsByWord checks if a word with the same Word field exists.
	//
	// "word" parameter is used to check if a word with the same Word field exists.
	//
	// If some error will occur during data check, the error will be returned together with "false" value.
	ExistsByWord(ctx context.Context, word string) (bool, error)
	// Method ExistsByClues checks if a word with the same Phonetic Clues field exists.
	//
	// "clues" parameter is used to check if a word with the same Phonetic Clues field exists.
	//
	// If some error will occur during data check, the error will be returned together with "false" value.
	ExistsByClues(ctx context.Context, clues string) (bool, error)
	// Method Create creates a new word using configured repository.
	//
	// "word" parameter is used to create a new word.
	//
	// If some error will occur during data creation, the error will be returned.
	Create(ctx context.Context, word *models.Word) error
	// Method Update updates a word using configured repository.
	//
	// "id" parameter is used to identify the word.
	// "word" parameter is used to update the word.
	//
	// If some error will occur during data update, the error will be returned.
	Update(ctx context.Context, id int, word *models.Word) error
	// Method Delete deletes a word using configured repository.
	//
	// "id" parameter is used to identify the word.
	//
	// If some error will occur during data deletion, the error will be returned.
	Delete(ctx context.Context, id int) error
}

// dictionaryService implements DictionaryService
type adminWordService struct {
	wordRepo              AdminWordRepository
	dictionaryHistoryRepo DictionaryHistoryRepository
	mediaBaseURL          string
	apiKey                string
}

// NewAdminWordService creates a new admin word service
func NewAdminWordService(
	wordRepo AdminWordRepository,
	dictionaryHistoryRepo DictionaryHistoryRepository,
	mediaBaseURL, apiKey string,
) *adminWordService {
	return &adminWordService{
		wordRepo:              wordRepo,
		dictionaryHistoryRepo: dictionaryHistoryRepo,
		mediaBaseURL:          mediaBaseURL,
		apiKey:                apiKey,
	}
}

// GetAllForAdmin retrieves a paginated list of words for admin endpoints
func (s *adminWordService) GetAllForAdmin(ctx context.Context, page, count int, search string) ([]models.WordListItem, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}

	words, err := s.wordRepo.GetAllForAdmin(ctx, page, count, search)
	if err != nil {
		return nil, fmt.Errorf("failed to get words: %w", err)
	}

	wordList := make([]models.WordListItem, len(words))
	for i, word := range words {
		wordList[i] = models.WordListItem{
			ID:                 word.ID,
			Word:               word.Word,
			PhoneticClues:      word.PhoneticClues,
			EnglishTranslation: word.EnglishTranslation,
		}
	}
	return wordList, nil
}

// GetByIDAdmin retrieves a word by ID for admin endpoints
func (s *adminWordService) GetByIDAdmin(ctx context.Context, id int) (*models.Word, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid word id")
	}

	return s.wordRepo.GetByIDAdmin(ctx, id)
}

// CreateWord creates a new word
func (s *adminWordService) CreateWord(ctx context.Context, request *models.CreateWordRequest, wordAudioFile multipart.File, wordAudioFilename string, wordExampleAudioFile multipart.File, wordExampleAudioFilename string) (int, error) {
	// Check if word with same Word field exists
	exists, err := s.wordRepo.ExistsByWord(ctx, request.Word)
	if err != nil {
		return 0, fmt.Errorf("failed to check word existence: %w", err)
	}
	if exists {
		return 0, fmt.Errorf("word '%s' already exists", request.Word)
	}

	word := &models.Word{
		Word:                      request.Word,
		PhoneticClues:             request.PhoneticClues,
		RussianTranslation:        request.RussianTranslation,
		EnglishTranslation:        request.EnglishTranslation,
		GermanTranslation:         request.GermanTranslation,
		Example:                   request.Example,
		ExampleRussianTranslation: request.ExampleRussianTranslation,
		ExampleEnglishTranslation: request.ExampleEnglishTranslation,
		ExampleGermanTranslation:  request.ExampleGermanTranslation,
		EasyPeriod:                request.EasyPeriod,
		NormalPeriod:              request.NormalPeriod,
		HardPeriod:                request.HardPeriod,
		ExtraHardPeriod:           request.ExtraHardPeriod,
	}

	// Handle word audio file upload if provided
	if wordAudioFile != nil && wordAudioFilename != "" {
		audioURL, err := uploadFileToMediaService(ctx, s.mediaBaseURL, s.apiKey, "word", wordAudioFile, wordAudioFilename)
		if err != nil {
			return 0, fmt.Errorf("failed to upload word audio: %w", err)
		}
		word.WordAudio = audioURL
	}

	// Handle word example audio file upload if provided
	if wordExampleAudioFile != nil && wordExampleAudioFilename != "" {
		audioURL, err := uploadFileToMediaService(ctx, s.mediaBaseURL, s.apiKey, "word_example", wordExampleAudioFile, wordExampleAudioFilename)
		if err != nil {
			return 0, fmt.Errorf("failed to upload word example audio: %w", err)
		}
		word.WordExampleAudio = audioURL
	}

	err = s.wordRepo.Create(ctx, word)
	if err != nil {
		return 0, fmt.Errorf("failed to create word: %w", err)
	}
	return word.ID, nil
}

// UpdateWord updates a word (partial update)
func (s *adminWordService) UpdateWord(ctx context.Context, id int, request *models.UpdateWordRequest, wordAudioFile multipart.File, wordAudioFilename string, wordExampleAudioFile multipart.File, wordExampleAudioFilename string) error {
	if id <= 0 {
		return fmt.Errorf("invalid word id")
	}

	// Validate first to get more specific error messages if validation fails
	if err := s.validateWord(ctx, request); err != nil {
		return err
	}

	// Get current word to check for existing audio URLs
	currentWord, err := s.wordRepo.GetByIDAdmin(ctx, id)
	if err != nil {
		return fmt.Errorf("word not found")
	}

	word := &models.Word{
		ID:                        id,
		Word:                      request.Word,
		PhoneticClues:             request.PhoneticClues,
		RussianTranslation:        request.RussianTranslation,
		EnglishTranslation:        request.EnglishTranslation,
		GermanTranslation:         request.GermanTranslation,
		Example:                   request.Example,
		ExampleRussianTranslation: request.ExampleRussianTranslation,
		ExampleEnglishTranslation: request.ExampleEnglishTranslation,
		ExampleGermanTranslation:  request.ExampleGermanTranslation,
	}
	if request.EasyPeriod != nil {
		word.EasyPeriod = *request.EasyPeriod
	}
	if request.NormalPeriod != nil {
		word.NormalPeriod = *request.NormalPeriod
	}
	if request.HardPeriod != nil {
		word.HardPeriod = *request.HardPeriod
	}
	if request.ExtraHardPeriod != nil {
		word.ExtraHardPeriod = *request.ExtraHardPeriod
	}

	// Handle word audio file update if provided
	if wordAudioFile != nil && wordAudioFilename != "" {
		// Delete old audio file if it exists
		if currentWord.WordAudio != "" && s.mediaBaseURL != "" && s.apiKey != "" {
			fileID := extractFileIDFromURL(currentWord.WordAudio)
			if fileID != "" {
				if err := deleteFileFromMediaService(ctx, s.mediaBaseURL, s.apiKey, "word", fileID); err != nil {
					return fmt.Errorf("failed to delete old word audio: %w", err)
				}
			}
		}

		// Upload new audio file
		audioURL, err := uploadFileToMediaService(ctx, s.mediaBaseURL, s.apiKey, "word", wordAudioFile, wordAudioFilename)
		if err != nil {
			return fmt.Errorf("failed to upload word audio: %w", err)
		}
		word.WordAudio = audioURL
	}

	// Handle word example audio file update if provided
	if wordExampleAudioFile != nil && wordExampleAudioFilename != "" {
		// Delete old audio file if it exists
		if currentWord.WordExampleAudio != "" && s.mediaBaseURL != "" && s.apiKey != "" {
			fileID := extractFileIDFromURL(currentWord.WordExampleAudio)
			if fileID != "" {
				if err := deleteFileFromMediaService(ctx, s.mediaBaseURL, s.apiKey, "word_example", fileID); err != nil {
					return fmt.Errorf("failed to delete old word example audio: %w", err)
				}
			}
		}

		// Upload new audio file
		audioURL, err := uploadFileToMediaService(ctx, s.mediaBaseURL, s.apiKey, "word_example", wordExampleAudioFile, wordExampleAudioFilename)
		if err != nil {
			return fmt.Errorf("failed to upload word example audio: %w", err)
		}
		word.WordExampleAudio = audioURL
	}

	return s.wordRepo.Update(ctx, id, word)
}

// validateWord validates a word for update
func (s *adminWordService) validateWord(ctx context.Context, request *models.UpdateWordRequest) error {
	// Prepare for concurrent check
	validErrChan := make(chan error, 3)

	// Check if word with same Word field exists
	go func() {
		if request.Word != "" {
			exists, err := s.wordRepo.ExistsByWord(ctx, request.Word)
			if err != nil {
				validErrChan <- fmt.Errorf("failed to check word existence: %w", err)
				return
			}
			if exists {
				validErrChan <- fmt.Errorf("word '%s' already exists", request.Word)
				return
			}
		}
		validErrChan <- nil
	}()

	// Check if word with same Phonetic Clues field exists
	go func() {
		if request.PhoneticClues != "" {
			exists, err := s.wordRepo.ExistsByClues(ctx, request.PhoneticClues)
			if err != nil {
				validErrChan <- fmt.Errorf("failed to check clues existence: %w", err)
				return
			}
			if exists {
				validErrChan <- fmt.Errorf("phonetic clues '%s' already exists", request.PhoneticClues)
				return
			}
		}
		validErrChan <- nil
	}()

	go func() {
		if request.EasyPeriod != nil && (*request.EasyPeriod < 1 || *request.EasyPeriod > 30) {
			validErrChan <- fmt.Errorf("easy period must be between 1 and 30")
			return
		}
		if request.NormalPeriod != nil && (*request.NormalPeriod < 1 || *request.NormalPeriod > 30) {
			validErrChan <- fmt.Errorf("normal period must be between 1 and 30")
			return
		}
		if request.HardPeriod != nil && (*request.HardPeriod < 1 || *request.HardPeriod > 30) {
			validErrChan <- fmt.Errorf("hard period must be between 1 and 30")
			return
		}
		if request.ExtraHardPeriod != nil && (*request.ExtraHardPeriod < 1 || *request.ExtraHardPeriod > 30) {
			validErrChan <- fmt.Errorf("extra hard period must be between 1 and 30")
			return
		}
		validErrChan <- nil
	}()

	for range 3 {
		err := <-validErrChan
		if err != nil {
			return fmt.Errorf("validation error: %w", err)
		}
	}

	return nil
}

// DeleteWord deletes a word by ID
func (s *adminWordService) DeleteWord(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid word id")
	}

	// Get word first to retrieve audio URLs
	word, err := s.wordRepo.GetByIDAdmin(ctx, id)
	if err != nil {
		return fmt.Errorf("word not found")
	}

	// Delete word audio file from media service if audio URL exists
	if word.WordAudio != "" && s.mediaBaseURL != "" && s.apiKey != "" {
		fileID := extractFileIDFromURL(word.WordAudio)
		if fileID != "" {
			if err := deleteFileFromMediaService(ctx, s.mediaBaseURL, s.apiKey, "word", fileID); err != nil {
				return fmt.Errorf("word audio file has not been deleted: %w", err)
			}
		}
	}

	// Delete word example audio file from media service if audio URL exists
	if word.WordExampleAudio != "" && s.mediaBaseURL != "" && s.apiKey != "" {
		fileID := extractFileIDFromURL(word.WordExampleAudio)
		if fileID != "" {
			if err := deleteFileFromMediaService(ctx, s.mediaBaseURL, s.apiKey, "word_example", fileID); err != nil {
				return fmt.Errorf("word example audio file has not been deleted: %w", err)
			}
		}
	}

	return s.wordRepo.Delete(ctx, id)
}
