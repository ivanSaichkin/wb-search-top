package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSearchRepository struct {
	mock.Mock
}

func (m *MockSearchRepository) AddEvent(ctx context.Context, event *models.SearchEvent) error {
	return m.Called(ctx, event).Error(0)
}

func (m *MockSearchRepository) AggregateTopFiveMinutes(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockSearchRepository) GetRawTop(ctx context.Context, limit int) ([]*models.SearchTopItem, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SearchTopItem), args.Error(1)
}

type MockStopListRepository struct {
	mock.Mock
}

func (m *MockStopListRepository) Add(ctx context.Context, word string) error {
	return m.Called(ctx, word).Error(0)
}

func (m *MockStopListRepository) Remove(ctx context.Context, word string) error {
	return m.Called(ctx, word).Error(0)
}

func (m *MockStopListRepository) Contains(ctx context.Context, word string) (bool, error) {
	args := m.Called(ctx, word)
	return args.Bool(0), args.Error(1)
}

func (m *MockStopListRepository) GetActiveList(ctx context.Context) (map[string]struct{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]struct{}), args.Error(1)
}

func TestSearchService_ProcessEvent(t *testing.T) {
	tests := []struct {
		name          string
		event         *models.SearchEvent
		setupMock     func(sr *MockSearchRepository)
		expectedError error
	}{
		{
			name: "Успех: валидное событие отправляется в репозиторий",
			event: &models.SearchEvent{
				Query:     "ноутбук",
				UserID:    "user_1",
				Timestamp: time.Now(),
			},
			setupMock: func(sr *MockSearchRepository) {
				sr.On("AddEvent", mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "Игнор: пустой Query возвращает nil и не ходит в базу",
			event: &models.SearchEvent{
				Query:     "",
				UserID:    "user_2",
				Timestamp: time.Now(),
			},
			setupMock: func(sr *MockSearchRepository) {
			},
			expectedError: nil,
		},
		{
			name: "Игнор: пустой UserID возвращает nil и не ходит в базу",
			event: &models.SearchEvent{
				Query:     "телефон",
				UserID:    "",
				Timestamp: time.Now(),
			},
			setupMock: func(sr *MockSearchRepository) {
			},
			expectedError: nil,
		},
		{
			name: "Ошибка: репозиторий вернул ошибку при сохранении",
			event: &models.SearchEvent{
				Query:     "планшет",
				UserID:    "user_3",
				Timestamp: time.Now(),
			},
			setupMock: func(sr *MockSearchRepository) {
				sr.On("AddEvent", mock.Anything, mock.Anything).Return(errors.New("db error"))
			},
			expectedError: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSearchRepo := new(MockSearchRepository)
			mockStopListRepo := new(MockStopListRepository)

			tt.setupMock(mockSearchRepo)

			svc := NewSearchService(mockSearchRepo, mockStopListRepo)
			err := svc.ProcessEvent(context.Background(), tt.event)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			mockSearchRepo.AssertExpectations(t)
		})
	}
}

func TestSearchService_GetFilteredTop(t *testing.T) {
	tests := []struct {
		name          string
		limit         int
		setupMocks    func(sr *MockSearchRepository, sl *MockStopListRepository)
		expectedItems []*models.SearchTopItem
		expectedError error
	}{
		{
			name:  "Успех: фильтрация стоп-слов и лимитирование",
			limit: 2,
			setupMocks: func(sr *MockSearchRepository, sl *MockStopListRepository) {
				stopList := map[string]struct{}{
					"запрет": {},
				}
				sl.On("GetActiveList", mock.Anything).Return(stopList, nil)

				rawTop := []*models.SearchTopItem{
					{Query: "купить телефон", UniqueHits: 100},
					{Query: "этот запрет", UniqueHits: 80},
					{Query: "ноутбук", UniqueHits: 50},
					{Query: "наушники", UniqueHits: 30},
				}
				sr.On("GetRawTop", mock.Anything, 10).Return(rawTop, nil)
			},
			expectedItems: []*models.SearchTopItem{
				{Query: "купить телефон", UniqueHits: 100},
				{Query: "ноутбук", UniqueHits: 50},
			},
			expectedError: nil,
		},
		{
			name:  "Ошибка: упал редис со стоп-листом",
			limit: 5,
			setupMocks: func(sr *MockSearchRepository, sl *MockStopListRepository) {
				sl.On("GetActiveList", mock.Anything).Return(nil, errors.New("stoplist error"))
			},
			expectedItems: nil,
			expectedError: errors.New("stoplist error"),
		},
		{
			name:  "Ошибка: упал основной редис",
			limit: 5,
			setupMocks: func(sr *MockSearchRepository, sl *MockStopListRepository) {
				sl.On("GetActiveList", mock.Anything).Return(map[string]struct{}{}, nil)
				sr.On("GetRawTop", mock.Anything, 25).Return(nil, errors.New("raw top error"))
			},
			expectedItems: nil,
			expectedError: errors.New("raw top error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSearchRepo := new(MockSearchRepository)
			mockStopListRepo := new(MockStopListRepository)

			tt.setupMocks(mockSearchRepo, mockStopListRepo)

			svc := NewSearchService(mockSearchRepo, mockStopListRepo)
			items, err := svc.GetFilteredTop(context.Background(), tt.limit)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
				assert.Nil(t, items)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedItems, items)
			}

			mockSearchRepo.AssertExpectations(t)
			mockStopListRepo.AssertExpectations(t)
		})
	}
}
