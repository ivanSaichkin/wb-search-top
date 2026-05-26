package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/models"
	"github.com/ivanSaichkin/wb-search-top/internal/domain/ports/repo"
	"github.com/ivanSaichkin/wb-search-top/internal/infrastructure/metrics"
)

type SearchService struct {
	searchRepo repo.SearchRepository
	stopRepo   repo.StopListRepository
}

func NewSearchService(searchRepo repo.SearchRepository, stopRepo repo.StopListRepository) *SearchService {
	return &SearchService{
		searchRepo: searchRepo,
		stopRepo:   stopRepo,
	}
}

// принимает сырое сообщение из брокера и передает в хранилище
func (s *SearchService) ProcessEvent(ctx context.Context, event *models.SearchEvent) error {
	if event.Query == "" || event.UserID == "" {
		return nil
	}

	return s.searchRepo.AddEvent(ctx, event)
}

// возвращает чистый Топ-N запросов, отфильтрованный от стоп-слов
func (s *SearchService) GetFilteredTop(ctx context.Context, limit int) ([]*models.SearchTopItem, error) {
	stopList, err := s.stopRepo.GetActiveList(ctx)
	if err != nil {
		return nil, err
	}

	// запрашиваем сырой топ с болшим запасом
	fetchLimit := limit * 5
	rawTop, err := s.searchRepo.GetRawTop(ctx, fetchLimit)
	if err != nil {
		return nil, err
	}

	filteredTop := make([]*models.SearchTopItem, 0, limit)
	for _, item := range rawTop {
		if isStopWordsInQuery(item.Query, stopList) {
			continue
		}

		filteredTop = append(filteredTop, item)

		if len(filteredTop) == limit {
			break
		}
	}

	return filteredTop, nil
}

// запускает фоновый процесс пересчета топа
func (s *SearchService) RunAggregatorWorker(ctx context.Context, interval time.Duration) {
	slog.Info("Starting aggregator worker...", slog.Duration("interval", interval))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Aggregator worker stopped")
			return
		case <-ticker.C:
			startTime := time.Now()

			err := s.searchRepo.AggregateTopFiveMinutes(ctx)

			metrics.AggregationDuration.Observe(time.Since(startTime).Seconds())

			if err != nil {
				slog.Error("Error aggregating top", "error", err)
			} else {
				slog.Debug("Top aggregated successfully")
			}
		}
	}
}

func isStopWordsInQuery(query string, stoplist map[string]struct{}) bool {
	words := strings.SplitSeq(query, " ")
	for word := range words {
		if _, exists := stoplist[word]; exists {
			return true
		}
	}

	return false
}
