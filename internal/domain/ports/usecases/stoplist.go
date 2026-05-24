package usecases

import "context"

// управляет добавлением и удалением стоп-слов
type StoplistUseCase interface {
	AddWord(ctx context.Context, word string) error
	RemoveWord(ctx context.Context, word string) error
}
