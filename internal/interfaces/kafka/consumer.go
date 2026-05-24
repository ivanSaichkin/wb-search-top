package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/models"
	"github.com/ivanSaichkin/wb-search-top/internal/domain/ports/usecases"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader  *kafka.Reader
	useCase usecases.SearchUseCase
}

func NewConsumer(brockers []string, topic, groupID string, useCase usecases.SearchUseCase) *Consumer {
	reader := kafka.NewReader(
		kafka.ReaderConfig{
			Brokers: brockers,
			Topic:   topic,
			GroupID: groupID,

			// настройки для highload
			MinBytes: 10e3,
			MaxBytes: 10e6,
		})

	return &Consumer{
		reader:  reader,
		useCase: useCase,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	log.Println("Starting Kafka consumer...")

	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("Kafka consumer stopped by context")
				return
			}
			log.Printf("Error reading kafka message: %v\n", err)
			continue
		}

		var event models.SearchEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Error unmarshaling event: %v\n", err)
			continue
		}

		if err := c.useCase.ProcessEvent(ctx, &event); err != nil {
			log.Printf("Error processing event: %v\n", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
