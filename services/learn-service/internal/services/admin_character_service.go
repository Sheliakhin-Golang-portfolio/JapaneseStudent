package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/japanesestudent/learn-service/internal/models"
)

// CharactersRepository is the interface that wraps methods for Characters table data access
type AdminCharactersRepository interface {
	// Method GetAllForAdmin retrieve all hiragana/katakana characters from a database.
	//
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetAllForAdmin(ctx context.Context) ([]models.Character, error)
	// Method GetByIDAdmin retrieve a full information about character by its ID using configured repository.
	//
	// "id" parameter is used to identify the character.
	//
	// If some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetByIDAdmin(ctx context.Context, id int) (*models.Character, error)
	// Method ExistsByVowelConsonant checks if a character with the same vowel and consonant exists.
	//
	// "vowel" parameter is used to check if a character with the same vowel exists.
	// "consonant" parameter is used to check if a character with the same consonant exists.
	//
	// If some error will occur during data check, the error will be returned together with "false" value.
	ExistsByVowelConsonant(ctx context.Context, vowel, consonant string) (bool, error)
	// Method ExistsByKatakanaOrHiragana checks if a character with the same katakana or hiragana exists.
	//
	// "katakana" parameter is used to check if a character with the same katakana exists.
	// "hiragana" parameter is used to check if a character with the same hiragana exists.
	//
	// If some error will occur during data check, the error will be returned together with "false" value.
	ExistsByKatakanaOrHiragana(ctx context.Context, katakana, hiragana string) (bool, error)
	// Method Create creates a new character using configured repository.
	//
	// "character" parameter is used to create a new character.
	//
	// If some error will occur during data creation, the error will be returned together with "nil" value.
	Create(ctx context.Context, character *models.Character) error
	// Method Update updates a character using configured repository.
	//
	// "id" parameter is used to identify the character.
	// "character" parameter is used to update the character.
	//
	// If some error will occur during data update, the error will be returned together with "nil" value.
	Update(ctx context.Context, id int, character *models.Character) error
	// Method Delete deletes a character using configured repository.
	//
	// "id" parameter is used to identify the character.
	//
	// If some error will occur during data deletion, the error will be returned together with "nil" value.
	Delete(ctx context.Context, id int) error
}

type adminService struct {
	repo         AdminCharactersRepository
	mediaBaseURL string
	apiKey       string
}

// NewAdminService creates a new admin service
func NewAdminService(repo AdminCharactersRepository, mediaBaseURL, apiKey string) *adminService {
	return &adminService{
		repo:         repo,
		mediaBaseURL: mediaBaseURL,
		apiKey:       apiKey,
	}
}

// GetAllForAdmin retrieves all characters for admin endpoints
func (s *adminService) GetAllForAdmin(ctx context.Context) ([]models.CharacterListItem, error) {
	characters, err := s.repo.GetAllForAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all characters for admin: %w", err)
	}
	items := make([]models.CharacterListItem, len(characters))
	for i, char := range characters {
		items[i] = models.CharacterListItem{
			ID:        char.ID,
			Consonant: char.Consonant,
			Vowel:     char.Vowel,
			Katakana:  char.Katakana,
			Hiragana:  char.Hiragana,
		}
	}
	return items, nil
}

// GetByIDAdmin retrieves a character by ID for admin endpoints
func (s *adminService) GetByIDAdmin(ctx context.Context, id int) (*models.Character, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid character id")
	}

	return s.repo.GetByIDAdmin(ctx, id)
}

// CreateCharacter creates a new character
func (s *adminService) CreateCharacter(ctx context.Context, request *models.CreateCharacterRequest, audioFile multipart.File, audioFilename string) (int, error) {
	// Perform validation before creating a new character
	if err := s.checkCreateCharacterValidation(ctx, request); err != nil {
		return 0, err
	}

	character := &models.Character{
		Consonant:      request.Consonant,
		Vowel:          request.Vowel,
		EnglishReading: request.EnglishReading,
		RussianReading: request.RussianReading,
		Katakana:       request.Katakana,
		Hiragana:       request.Hiragana,
	}

	// Handle audio file upload if provided
	if audioFile != nil && audioFilename != "" {
		audioURL, err := s.uploadAudio(ctx, audioFile, audioFilename)
		if err != nil {
			return 0, fmt.Errorf("failed to upload audio: %w", err)
		}
		character.Audio = audioURL
	}

	err := s.repo.Create(ctx, character)
	if err != nil {
		return 0, err
	}
	return character.ID, nil
}

// checkCreateCharacterValidation checks the validity of the character creation request
func (s *adminService) checkCreateCharacterValidation(ctx context.Context, request *models.CreateCharacterRequest) error {
	validationErrors := make(chan error, 2)
	go func() {
		exists, err := s.repo.ExistsByVowelConsonant(ctx, request.Vowel, request.Consonant)
		if err != nil {
			validationErrors <- fmt.Errorf("failed to check character existence: %w", err)
			return
		}
		if exists {
			validationErrors <- fmt.Errorf("character with vowel '%s' and consonant '%s' already exists", request.Vowel, request.Consonant)
			return
		}
		validationErrors <- nil
	}()
	go func() {
		exists, err := s.repo.ExistsByKatakanaOrHiragana(ctx, request.Katakana, request.Hiragana)
		if err != nil {
			validationErrors <- fmt.Errorf("failed to check character existence: %w", err)
			return
		}
		if exists {
			validationErrors <- fmt.Errorf("character with katakana '%s' or hiragana '%s' already exists", request.Katakana, request.Hiragana)
			return
		}
		validationErrors <- nil
	}()

	for range 2 {
		err := <-validationErrors
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateCharacter updates a character (partial update)
func (s *adminService) UpdateCharacter(ctx context.Context, id int, request *models.UpdateCharacterRequest, audioFile multipart.File, audioFilename string) error {
	if id <= 0 {
		return fmt.Errorf("invalid character id")
	}

	// Get current character to check for existing audio URL
	currentCharacter, err := s.repo.GetByIDAdmin(ctx, id)
	if err != nil {
		return fmt.Errorf("character not found")
	}

	if err := s.checkUpdateCharacterValidation(ctx, currentCharacter, request); err != nil {
		return err
	}

	// Handle audio file update if provided
	newAudioURL := ""
	if audioFile != nil && audioFilename != "" {
		// Delete old audio file if it exists
		if currentCharacter.Audio != "" && s.mediaBaseURL != "" && s.apiKey != "" {
			fileID := s.extractFileIDFromAudioURL(currentCharacter.Audio)
			if fileID != "" {
				if err := s.deleteAudioFromMediaService(ctx, fileID); err != nil {
					return fmt.Errorf("failed to delete old audio: %w", err)
				}
			}
		}

		// Upload new audio file
		audioURL, err := s.uploadAudio(ctx, audioFile, audioFilename)
		if err != nil {
			return fmt.Errorf("failed to upload audio: %w", err)
		}
		newAudioURL = audioURL
	}

	characterToUpdate := &models.Character{
		ID: id,
	}
	if request.Consonant != "" {
		characterToUpdate.Consonant = request.Consonant
	}
	if request.Vowel != "" {
		characterToUpdate.Vowel = request.Vowel
	}
	if request.EnglishReading != "" {
		characterToUpdate.EnglishReading = request.EnglishReading
	}
	if request.RussianReading != "" {
		characterToUpdate.RussianReading = request.RussianReading
	}
	if request.Katakana != "" {
		characterToUpdate.Katakana = request.Katakana
	}
	if request.Hiragana != "" {
		characterToUpdate.Hiragana = request.Hiragana
	}
	if newAudioURL != "" {
		characterToUpdate.Audio = newAudioURL
	}

	return s.repo.Update(ctx, id, characterToUpdate)
}

// checkUpdateCharacterValidation checks the validity of the character update request
func (s *adminService) checkUpdateCharacterValidation(ctx context.Context, currentCharacter *models.Character, request *models.UpdateCharacterRequest) error {
	validationErrors := make(chan error, 2)
	go func() {
		if request.Consonant != "" || request.Vowel != "" {
			if request.Consonant == "" {
				request.Consonant = currentCharacter.Consonant
			}
			if request.Vowel == "" {
				request.Vowel = currentCharacter.Vowel
			}
		}
		if request.Consonant != "" && request.Vowel != "" {
			exists, err := s.repo.ExistsByVowelConsonant(ctx, request.Vowel, request.Consonant)
			if err != nil {
				validationErrors <- fmt.Errorf("failed to check character existence: %w", err)
				return
			}
			if exists {
				validationErrors <- fmt.Errorf("character with vowel '%s' and consonant '%s' already exists", request.Vowel, request.Consonant)
				return
			}
		}
		validationErrors <- nil
	}()
	go func() {
		if request.Katakana != "" || request.Hiragana != "" {
			exists, err := s.repo.ExistsByKatakanaOrHiragana(ctx, request.Katakana, request.Hiragana)
			if err != nil {
				validationErrors <- fmt.Errorf("failed to check character existence: %w", err)
				return
			}
			if exists {
				validationErrors <- fmt.Errorf("character with katakana '%s' or hiragana '%s' already exists", request.Katakana, request.Hiragana)
				return
			}
		}
		validationErrors <- nil
	}()

	for range 2 {
		err := <-validationErrors
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteCharacter deletes a character by ID
func (s *adminService) DeleteCharacter(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid character id")
	}

	// Get character first to retrieve audio URL
	character, err := s.repo.GetByIDAdmin(ctx, id)
	if err != nil {
		return fmt.Errorf("character not found")
	}

	// Delete audio file from media service if audio URL exists
	if character.Audio != "" && s.mediaBaseURL != "" && s.apiKey != "" {
		fileID := s.extractFileIDFromAudioURL(character.Audio)
		if fileID != "" {
			if err := s.deleteAudioFromMediaService(ctx, fileID); err != nil {
				return fmt.Errorf("audio file has not been deleted: %w", err)
			}
		}
	}

	return s.repo.Delete(ctx, id)
}

// uploadAudio uploads an audio file to the media-service using io.Pipe for streaming
func (s *adminService) uploadAudio(ctx context.Context, audioFile multipart.File, audioFilename string) (string, error) {
	if s.mediaBaseURL == "" {
		return "", fmt.Errorf("MEDIA_BASE_URL is not configured")
	}
	if s.apiKey == "" {
		return "", fmt.Errorf("API_KEY is not configured")
	}

	// Create a pipe for streaming
	pr, pw := io.Pipe()
	defer pr.Close()

	// Create multipart writer
	writer := multipart.NewWriter(pw)

	// Start goroutine to write file to pipe
	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		defer writer.Close()

		// Create form field for file
		part, err := writer.CreateFormFile("file", audioFilename)
		if err != nil {
			errChan <- fmt.Errorf("failed to create form file: %w", err)
			return
		}

		// Copy file content to form
		_, err = io.Copy(part, audioFile)
		if err != nil {
			errChan <- fmt.Errorf("failed to copy file: %w", err)
			return
		}

		errChan <- nil
	}()

	// Build upload URL
	uploadURL := fmt.Sprintf("%s/media/character", s.mediaBaseURL)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, pr)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", s.apiKey)

	// Make HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// We will not wait for goroutine to complete. Instead it will finish when we close the pipe.
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors from goroutine
	if err := <-errChan; err != nil {
		return "", err
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media-service returned error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Read audio URL from response
	audioURL, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return strings.TrimSpace(string(audioURL)), nil
}

// deleteAudioFromMediaService sends a DELETE request to media service to delete the audio file
func (s *adminService) deleteAudioFromMediaService(ctx context.Context, fileID string) error {
	if s.mediaBaseURL == "" || s.apiKey == "" {
		return nil // Skip if media service is not configured
	}

	// Construct the delete URL: {mediaBaseURL}/media/character/{fileID}
	deleteURL := strings.TrimSuffix(s.mediaBaseURL, "/") + "/media/character/" + fileID

	// Create DELETE request
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	// Set API key header
	req.Header.Set("X-API-Key", s.apiKey)

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("media service returned status %d", resp.StatusCode)
	}

	return nil
}

// extractFileIDFromAudioURL extracts the file ID (filename) from the audio URL
// The audio URL format is expected to be like: http://.../media/audio/{fileID}
// Returns the last part of the URL path as the file ID
func (s *adminService) extractFileIDFromAudioURL(audioURL string) string {
	if audioURL == "" {
		return ""
	}

	// Parse URL to handle it properly
	parsedURL, err := url.Parse(audioURL)
	if err != nil {
		// If URL parsing fails, try to extract from string directly
		// Remove query parameters and fragments
		parts := strings.Split(audioURL, "?")
		parts = strings.Split(parts[0], "#")
		urlPath := parts[0]

		// Extract the last part of the path by splitting on "/"
		pathParts := strings.Split(strings.Trim(urlPath, "/"), "/")
		if len(pathParts) > 0 {
			return pathParts[len(pathParts)-1]
		}
		return ""
	}

	// Extract the last part of the path
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) > 0 {
		return pathParts[len(pathParts)-1]
	}

	return ""
}
