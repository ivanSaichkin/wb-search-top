package rabbitmq

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/models"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/mock"
)

type MockSearchUseCase struct {
	mock.Mock
}

func (m *MockSearchUseCase) ProcessEvent(ctx context.Context, event *models.SearchEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockSearchUseCase) GetFilteredTop(ctx context.Context, limit int) ([]*models.SearchTopItem, error) {
	return nil, nil
}

func (m *MockSearchUseCase) RunAggregatorWorker(ctx context.Context, interval time.Duration) {}

type MockAcknowledger struct {
	mock.Mock
}

func (m *MockAcknowledger) Ack(tag uint64, multiple bool) error {
	return m.Called(tag, multiple).Error(0)
}

func (m *MockAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	return m.Called(tag, multiple, requeue).Error(0)
}

func (m *MockAcknowledger) Reject(tag uint64, requeue bool) error {
	return m.Called(tag, requeue).Error(0)
}

func TestConsumer_processDelivery(t *testing.T) {
	tests := []struct {
		name         string
		body         []byte
		setupUseCase func(uc *MockSearchUseCase)
		setupAck     func(ack *MockAcknowledger)
	}{
		{
			name: "Успех: валидный JSON, UseCase отработал без ошибок -> Ack",
			body: []byte(`{"query": "ноутбук", "user_id": "123", "timestamp": "2026-05-26T15:00:00Z"}`),
			setupUseCase: func(uc *MockSearchUseCase) {
				uc.On("ProcessEvent", mock.Anything, mock.MatchedBy(func(e *models.SearchEvent) bool {
					return e.Query == "ноутбук" && e.UserID == "123"
				})).Return(nil)
			},
			setupAck: func(ack *MockAcknowledger) {
				ack.On("Ack", uint64(1), false).Return(nil)
			},
		},
		{
			name: "Ошибка парсинга: битый JSON -> Nack (requeue=false)",
			body: []byte(`{"query": "ноутбук", "user_id": `),
			setupUseCase: func(uc *MockSearchUseCase) {
			},
			setupAck: func(ack *MockAcknowledger) {
				ack.On("Nack", uint64(1), false, false).Return(nil)
			},
		},
		{
			name: "Ошибка UseCase: бизнес-логика вернула ошибку -> Nack (requeue=false)",
			body: []byte(`{"query": "айфон", "user_id": "999"}`),
			setupUseCase: func(uc *MockSearchUseCase) {
				uc.On("ProcessEvent", mock.Anything, mock.Anything).Return(errors.New("db error"))
			},
			setupAck: func(ack *MockAcknowledger) {
				ack.On("Nack", uint64(1), false, false).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := new(MockSearchUseCase)
			tt.setupUseCase(mockUC)

			mockAck := new(MockAcknowledger)
			tt.setupAck(mockAck)

			c := &Consumer{
				useCase: mockUC,
				queue:   "test_queue",
			}

			delivery := amqp.Delivery{
				Acknowledger: mockAck,
				DeliveryTag:  1,
				Body:         tt.body,
			}

			c.processDelivery(context.Background(), delivery)

			mockUC.AssertExpectations(t)
			mockAck.AssertExpectations(t)
		})
	}
}
