package repo

import (
	"context"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/models"
)

// управляет записью и агрегацией сырых логов
type SearchRepository interface {
	AddEvent(ctx context.Context, event *models.SearchEvent) error
	AggregateTopFiveMinutes(ctx context.Context) error

	GetRawTop(ctx context.Context, limit int) ([]*models.SearchTopItem, error)
}

// отвечает за динамическое управление нежелательными словами
type StopListRepository interface {
	Add(ctx context.Context, word string) error
	Remove(ctx context.Context, word string) error
	GetActiveList(ctx context.Context) (map[string]struct{}, error)
}
