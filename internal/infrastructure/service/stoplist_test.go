package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStopListService_AddWord(t *testing.T) {
	tests := []struct {
		name          string
		word          string
		setupMock     func(m *MockStopListRepository)
		expectedError error
	}{
		{
			name: "Успех: слово добавлено (или уже было в базе - идемпотентность)",
			word: "запрещенка",
			setupMock: func(m *MockStopListRepository) {
				m.On("Add", mock.Anything, "запрещенка").Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "Ошибка: инфраструктурный сбой Redis при добавлении",
			word: "казино",
			setupMock: func(m *MockStopListRepository) {
				m.On("Add", mock.Anything, "казино").Return(errors.New("redis timeout"))
			},
			expectedError: errors.New("redis timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockStopListRepository)
			tt.setupMock(mockRepo)

			svc := NewStopListService(mockRepo)
			err := svc.AddWord(context.Background(), tt.word)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestStopListService_RemoveWord(t *testing.T) {
	tests := []struct {
		name          string
		word          string
		setupMock     func(m *MockStopListRepository)
		expectedError error
	}{
		{
			name: "Успех: слово удалено (или его и так не было в базе - идемпотентность)",
			word: "оружие",
			setupMock: func(m *MockStopListRepository) {
				m.On("Remove", mock.Anything, "оружие").Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "Ошибка: инфраструктурный сбой Redis при удалении",
			word: "наркотики",
			setupMock: func(m *MockStopListRepository) {
				m.On("Remove", mock.Anything, "наркотики").Return(errors.New("redis connection refused"))
			},
			expectedError: errors.New("redis connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockStopListRepository)
			tt.setupMock(mockRepo)

			svc := NewStopListService(mockRepo)
			err := svc.RemoveWord(context.Background(), tt.word)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
