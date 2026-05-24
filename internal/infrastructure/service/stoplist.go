package service

import (
	"context"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/ports/repo"
)

type StopListService struct {
	repo repo.StopListRepository
}

func NewStopListService(repo repo.StopListRepository) *StopListService {
	return &StopListService{repo: repo}
}

func (s *StopListService) AddWord(ctx context.Context, word string) error {
	return s.repo.Add(ctx, word)
}

func (s *StopListService) RemoveWord(ctx context.Context, word string) error {
	return s.repo.Remove(ctx, word)
}
