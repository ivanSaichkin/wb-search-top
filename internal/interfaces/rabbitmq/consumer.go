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

	// Инициализируем инфраструктуру для Dead Letter Queue
	dlxName := queue + ".dlx"
	dlqName := queue + ".dlq"

	err = ch.ExchangeDeclare(
		dlxName,  // name
		"fanout", // type (fanout отправит сообщение во все привязанные очереди)
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	_, err = ch.QueueDeclare(
		dlqName, // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	// // Связываем DLQ с DLX
	err = ch.QueueBind(dlqName, "", dlxName, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	args := amqp.Table{
		"x-dead-letter-exchange": dlxName,
	}

	_, err = ch.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,  // аргументы для DLQ
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
	slog.Info("Starting RabbitMQ consumer with Manual Ack...", slog.String("queue", c.queue))

	msgs, err := c.ch.Consume(
		c.queue, // queue
		"",      // consumer tags
		false,   // auto-ack FALSE
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
			// Если пришел битый JSON, мы логируем ошибку и отправляем Nack с requeue=false.
			// Сообщение автоматически улетает в search_events.dlq
			if err := json.Unmarshal(d.Body, &event); err != nil {
				slog.Error("Error unmarshaling event, sending to DLQ", "error", err, "body", string(d.Body))
				if err := d.Nack(false, false); err != nil {
					slog.Error("Error send to DLQ", "error", err)
				}
				continue
			}

			// Если бизнес-логика ответила ошибкой, мы также изолируем сообщение в DLQ,
			// чтобы не зацикливать обработку поврежденных данных.
			if err := c.useCase.ProcessEvent(ctx, &event); err != nil {
				slog.Error("Error processing event, sending to DLQ", "error", err, "query", event.Query)
				if err := d.Nack(false, false); err != nil {
					slog.Error("Error send to DLQ", "error", err)
				}
				continue
			}

			// Успешный исход — подтверждаем обработку
			d.Ack(false)
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
