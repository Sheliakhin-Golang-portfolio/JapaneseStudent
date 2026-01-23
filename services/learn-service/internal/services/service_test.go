package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
)

// mockHistoryRepository is a mock implementation of CharacterLearnHistoryRepository
type mockHistoryRepository struct {
	histories     []models.CharacterLearnHistory
	userHistories []models.UserLearnHistory
	err           error
}

func (m *mockHistoryRepository) GetByUserIDAndCharacterIDs(ctx context.Context, userID int, characterIDs []int) ([]models.CharacterLearnHistory, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.histories, nil
}

func (m *mockHistoryRepository) GetByUserID(ctx context.Context, userID int) ([]models.UserLearnHistory, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.userHistories, nil
}

func (m *mockHistoryRepository) Upsert(ctx context.Context, histories []models.CharacterLearnHistory) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockHistoryRepository) LowerResultsByUserID(ctx context.Context, userID int) error {
	return m.err
}

// mockCharactersRepository is a mock implementation of TestResultCharactersRepository
type mockCharactersRepository struct {
	totalCount int
	err        error
}

func (m *mockCharactersRepository) GetTotalCount(ctx context.Context) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.totalCount, nil
}

// ... existing code ...

func TestTestResultService_DropUserMarks(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		mockRepo      *mockHistoryRepository
		expectedError bool
		errorContains string
	}{
		{
			name:   "success drop user marks",
			userID: 1,
			mockRepo: &mockHistoryRepository{
				err: nil,
			},
			expectedError: false,
		},
		{
			name:   "database error",
			userID: 1,
			mockRepo: &mockHistoryRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			errorContains: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCharRepo := &mockCharactersRepository{totalCount: 46}
			svc := NewTestResultService(tt.mockRepo, mockCharRepo)
			ctx := context.Background()

			err := svc.DropUserMarks(ctx, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
