package usecases

import (
	"context"
	"time"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/models"
)

// управляет записью запросов, получением и обновлением топа
type SearchUseCase interface {
	ProcessEvent(ctx context.Context, event *models.SearchEvent) error
	GetFilteredTop(ctx context.Context, limit int) ([]*models.SearchTopItem, error)

	RunAggregatorWorker(ctx context.Context, interval time.Duration)
}
