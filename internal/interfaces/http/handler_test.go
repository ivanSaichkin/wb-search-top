package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSearchUseCase struct {
	mock.Mock
}

func (m *MockSearchUseCase) GetFilteredTop(ctx context.Context, limit int) ([]*models.SearchTopItem, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SearchTopItem), args.Error(1)
}

func (m *MockSearchUseCase) ProcessEvent(ctx context.Context, event *models.SearchEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockSearchUseCase) RunAggregatorWorker(ctx context.Context, interval time.Duration) {
	m.Called(ctx, interval)
}

type MockStoplistUseCase struct {
	mock.Mock
}

func (m *MockStoplistUseCase) AddWord(ctx context.Context, word string) error {
	return m.Called(ctx, word).Error(0)
}

func (m *MockStoplistUseCase) RemoveWord(ctx context.Context, word string) error {
	return m.Called(ctx, word).Error(0)
}

func TestHandler_GetTop(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		setupMock      func(m *MockSearchUseCase)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "Успех: получаем топ с кастомным лимитом",
			query: "?limit=2",
			setupMock: func(m *MockSearchUseCase) {
				items := []*models.SearchTopItem{
					{Query: "iphone", UniqueHits: 100},
					{Query: "macbook", UniqueHits: 50},
				}
				m.On("GetFilteredTop", mock.Anything, 2).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"query":"iphone","unique_hits":100},{"query":"macbook","unique_hits":50}]`,
		},
		{
			name:  "Успех: используем дефолтный лимит 10, если параметр не передан",
			query: "",
			setupMock: func(m *MockSearchUseCase) {
				m.On("GetFilteredTop", mock.Anything, 10).Return([]*models.SearchTopItem{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		{
			name:  "Ошибка: usecase вернул внутреннюю ошибку",
			query: "?limit=5",
			setupMock: func(m *MockSearchUseCase) {
				m.On("GetFilteredTop", mock.Anything, 5).Return(nil, errors.New("internal cluster error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal Server Error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := new(MockSearchUseCase)
			tt.setupMock(mockUC)

			h := NewHandler(mockUC, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/top"+tt.query, nil)
			w := httptest.NewRecorder()

			h.GetTop(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				if tt.expectedStatus == http.StatusOK {
					assert.JSONEq(t, tt.expectedBody, w.Body.String())
				} else {
					assert.Equal(t, tt.expectedBody, w.Body.String())
				}
			}

			mockUC.AssertExpectations(t)
		})
	}
}

func TestHandler_AddStopWord(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		setupMock      func(m *MockStoplistUseCase)
		expectedStatus int
	}{
		{
			name: "Успех: слово добавлено",
			body: `{"word": "казино"}`,
			setupMock: func(m *MockStoplistUseCase) {
				m.On("AddWord", mock.Anything, "казино").Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Ошибка: невалидный JSON",
			body:           `{"word": "казино"`,
			setupMock:      func(m *MockStoplistUseCase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Ошибка: usecase вернул внутреннюю ошибку",
			body: `{"word": "запрещенка"}`,
			setupMock: func(m *MockStoplistUseCase) {
				m.On("AddWord", mock.Anything, "запрещенка").Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := new(MockStoplistUseCase)
			tt.setupMock(mockUC)

			h := NewHandler(nil, mockUC)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/stoplist", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.AddStopWord(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockUC.AssertExpectations(t)
		})
	}
}

func TestHandler_RemoveStopWord(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		setupMock      func(m *MockStoplistUseCase)
		expectedStatus int
	}{
		{
			name:  "Успех: слово удалено",
			query: "?word=оружие",
			setupMock: func(m *MockStoplistUseCase) {
				m.On("RemoveWord", mock.Anything, "оружие").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Ошибка: отсутствует обязательный параметр query",
			query:          "",
			setupMock:      func(m *MockStoplistUseCase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "Ошибка: usecase вернул ошибку при удалении",
			query: "?word=оружие",
			setupMock: func(m *MockStoplistUseCase) {
				m.On("RemoveWord", mock.Anything, "оружие").Return(errors.New("redis timeout"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := new(MockStoplistUseCase)
			tt.setupMock(mockUC)

			h := NewHandler(nil, mockUC)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/stoplist"+tt.query, nil)
			w := httptest.NewRecorder()

			h.RemoveStopWord(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockUC.AssertExpectations(t)
		})
	}
}
