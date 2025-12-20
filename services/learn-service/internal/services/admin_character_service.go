package services

import (
	"context"
	"fmt"

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
	repo AdminCharactersRepository
}

// NewAdminService creates a new admin service
func NewAdminService(repo AdminCharactersRepository) *adminService {
	return &adminService{
		repo: repo,
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
func (s *adminService) CreateCharacter(ctx context.Context, request *models.CreateCharacterRequest) (int, error) {
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
func (s *adminService) UpdateCharacter(ctx context.Context, id int, request *models.UpdateCharacterRequest) error {
	if id <= 0 {
		return fmt.Errorf("invalid character id")
	}

	if err := s.checkUpdateCharacterValidation(ctx, id, request); err != nil {
		return err
	}

	character := &models.Character{
		ID:             id,
		Consonant:      request.Consonant,
		Vowel:          request.Vowel,
		EnglishReading: request.EnglishReading,
		RussianReading: request.RussianReading,
		Katakana:       request.Katakana,
		Hiragana:       request.Hiragana,
	}
	return s.repo.Update(ctx, id, character)
}

// checkUpdateCharacterValidation checks the validity of the character update request
func (s *adminService) checkUpdateCharacterValidation(ctx context.Context, id int, request *models.UpdateCharacterRequest) error {
	validationErrors := make(chan error, 2)
	go func() {
		if request.Consonant != "" || request.Vowel != "" {
			character, err := s.repo.GetByIDAdmin(ctx, id)
			if err != nil {
				validationErrors <- fmt.Errorf("failed to get character by id: %w", err)
				return
			}
			if request.Consonant == "" {
				request.Consonant = character.Consonant
			}
			if request.Vowel == "" {
				request.Vowel = character.Vowel
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
	return s.repo.Delete(ctx, id)
}
