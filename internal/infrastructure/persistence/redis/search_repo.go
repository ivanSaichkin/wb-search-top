package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/models"
	"github.com/redis/go-redis/v9"
)

const (
	rawTopKey = "search:raw_top_5min"
	bucketTTL = 6 * time.Minute // храним бакеты чуть дольше 5 минут для безопасности
)

type SearchRepo struct {
	client *redis.Client
}

func NewSearchRepo(client *redis.Client) *SearchRepo {
	return &SearchRepo{client: client}
}

// сохраняет событие в бакет текущей минуты
func (r *SearchRepo) AddEvent(ctx context.Context, event models.SearchEvent) error {
	minuteBucket := event.Timestamp.Truncate(time.Minute).Unix()

	hllKey := fmt.Sprintf("search:hll:%d:%s", minuteBucket, event.Query)
	activeQueriesKey := fmt.Sprintf("search:active:%d", minuteBucket)

	pipe := r.client.Pipeline()

	pipe.PFAdd(ctx, hllKey, event.UserID)
	pipe.Expire(ctx, hllKey, bucketTTL)

	pipe.SAdd(ctx, activeQueriesKey, event.Query)
	pipe.Expire(ctx, activeQueriesKey, bucketTTL)

	_, err := pipe.Exec(ctx)
	return err
}

// собирает данные за последние 5 минут и формирует ZSET
func (r *SearchRepo) AggregateTopFiveMinutes(ctx context.Context) error {
	now := time.Now().Truncate(time.Minute)

	var activeKeys []string
	for i := 0; i < 5; i++ {
		bucketTime := now.Add(-time.Duration(i) * time.Minute).Unix()
		activeKeys = append(activeKeys, fmt.Sprintf("search:active:%d", bucketTime))
	}

	queries, err := r.client.SUnion(ctx, activeKeys...).Result()
	if err != nil {
		return err
	}

	if len(queries) == 0 {
		return r.client.Del(ctx, rawTopKey).Err()
	}

	var zMembers []redis.Z
	for _, query := range queries {
		var hllKeys []string
		for i := 0; i < 5; i++ {
			bucketTime := now.Add(-time.Duration(i) * time.Minute).Unix()
			hllKeys = append(hllKeys, fmt.Sprintf("search:hll:%d:%s", bucketTime, query))
		}

		count, err := r.client.PFCount(ctx, hllKeys...).Result()
		if err != nil {
			continue // TODO
		}

		zMembers = append(zMembers, redis.Z{
			Score:  float64(count),
			Member: query,
		})
	}

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, rawTopKey)
	pipe.ZAdd(ctx, rawTopKey, zMembers...)
	_, err = pipe.Exec(ctx)

	return err
}

func (r *SearchRepo) GetRawTop(ctx context.Context, limit int) ([]*models.SearchTopItem, error) {
	result, err := r.client.ZRevRangeWithScores(ctx, rawTopKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	items := make([]*models.SearchTopItem, 0, len(result))
	for _, res := range result {
		items = append(items, &models.SearchTopItem{
			Query:      res.Member.(string),
			UniqueHits: int64(res.Score),
		})
	}

	return items, nil
}
