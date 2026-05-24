package models

import "time"

// SearchEvent описывает контракт входящего поискового события из брокера
// UserID для борьбы с накрутками парсеров
type SearchEvent struct {
	Query     string    `json:"query"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// SearchTopItem представляет одну позицию в итоговом топе
type SearchTopItem struct {
	Query      string `json:"query"`
	UniqueHits int64  `json:"unique_hits"` // вес запроса на основе уникальных пользователей
}
