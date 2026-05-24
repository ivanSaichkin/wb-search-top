package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const stopListKey = "search:stopList"

type StopListRepo struct {
	client *redis.Client
}

func NewStopListRepo(client *redis.Client) *StopListRepo {
	return &StopListRepo{client: client}
}

func (r *StopListRepo) Add(ctx context.Context, word string) error {
	return r.client.SAdd(ctx, stopListKey, word).Err()
}

func (r *StopListRepo) Remove(ctx context.Context, word string) error {
	return r.client.SRem(ctx, stopListKey, word).Err()
}

func (r *StopListRepo) Contains(ctx context.Context, word string) (bool, error) {
	return r.client.SIsMember(ctx, stopListKey, word).Result()
}

func (r *StopListRepo) GetActiveList(ctx context.Context) (map[string]struct{}, error) {
	words, err := r.client.SMembers(ctx, stopListKey).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]struct{}, len(words))
	for _, w := range words {
		result[w] = struct{}{}
	}

	return result, nil
}
