package rabbitmq

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/models"
	"github.com/ivanSaichkin/wb-search-top/internal/domain/ports/usecases"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn    *amqp.Connection
	ch      *amqp.Channel
	queue   string
	useCase usecases.SearchUseCase
}

func NewConsumer(url, queue string, useCase usecases.SearchUseCase) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	_, err = ch.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &Consumer{
		conn:    conn,
		ch:      ch,
		queue:   queue,
		useCase: useCase,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) {
	slog.Info("Starting RabbitMQ consumer...", slog.String("queue", c.queue))

	msgs, err := c.ch.Consume(
		c.queue, // queue
		"",      // consumer tags
		true,    // auto-ack
		false,   // exclusive
		false,   // no-local
		false,   // no-wait
		nil,     // args
	)
	if err != nil {
		slog.Error("Failed to register a consumer", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("RabbitMQ consumer stopped by context")
			return
		case d, ok := <-msgs:
			if !ok {
				slog.Warn("RabbitMQ channel closed")
				return
			}

			slog.Debug("Received a message from RabbitMQ", "body", string(d.Body))

			var event models.SearchEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				slog.Error("Error unmarshaling event", "error", err, "body", string(d.Body))
				continue
			}

			if err := c.useCase.ProcessEvent(ctx, &event); err != nil {
				slog.Error("Error processing event", "error", err, "query", event.Query)
			}
		}
	}
}

func (c *Consumer) Close() {
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	slog.Info("RabbitMQ connections closed cleanly")
}
