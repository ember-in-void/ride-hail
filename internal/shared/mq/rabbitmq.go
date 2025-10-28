package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ridehail/internal/shared/config"
	"ridehail/internal/shared/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQ представляет подключение к RabbitMQ с автореконнектом
type RabbitMQ struct {
	url    string
	conn   *amqp.Connection
	ch     *amqp.Channel
	log    *logger.Logger
	mu     sync.RWMutex
	closed bool
}

// NewRabbitMQ создает подключение к RabbitMQ с retry
func NewRabbitMQ(ctx context.Context, cfg config.MQConfig, log *logger.Logger) (*RabbitMQ, error) {
	url := cfg.AMQPURL()

	mq := &RabbitMQ{
		url: url,
		log: log,
	}

	// Retry логика: максимум 10 попыток с экспоненциальной задержкой
	maxRetries := 10
	retryDelay := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Info(logger.Entry{
			Action:  "rabbitmq_connection_attempt",
			Message: fmt.Sprintf("attempt %d/%d", attempt, maxRetries),
			Additional: map[string]any{
				"host": cfg.Host,
				"port": cfg.Port,
			},
		})

		if err := mq.connect(ctx); err != nil {
			log.Error(logger.Entry{
				Action:  "rabbitmq_connection_attempt_failed",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
				Additional: map[string]any{
					"attempt":      attempt,
					"max_retries":  maxRetries,
					"retry_in_sec": retryDelay.Seconds(),
				},
			})

			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
			}

			// Экспоненциальная задержка с jitter
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
				retryDelay = time.Duration(float64(retryDelay) * 1.5)
				if retryDelay > 30*time.Second {
					retryDelay = 30 * time.Second
				}
			}
			continue
		}

		log.Info(logger.Entry{
			Action:  "rabbitmq_connected",
			Message: fmt.Sprintf("connected to %s:%d", cfg.Host, cfg.Port),
			Additional: map[string]any{
				"attempt": attempt,
			},
		})

		return mq, nil
	}

	return nil, fmt.Errorf("unexpected error: retry loop completed without success")
}

func (mq *RabbitMQ) connect(ctx context.Context) error {
	conn, err := amqp.Dial(mq.url)
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	// QoS для равномерного распределения нагрузки
	if err := ch.Qos(10, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("set qos: %w", err)
	}

	mq.mu.Lock()
	mq.conn = conn
	mq.ch = ch
	mq.mu.Unlock()

	return nil
}

// Channel возвращает активный канал
func (mq *RabbitMQ) Channel() *amqp.Channel {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.ch
}

// Publish публикует сообщение в exchange
func (mq *RabbitMQ) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	mq.mu.RLock()
	ch := mq.ch
	mq.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("rabbitmq channel not available")
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return ch.PublishWithContext(
		publishCtx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

// Consume начинает чтение сообщений из очереди
func (mq *RabbitMQ) Consume(ctx context.Context, queue, consumer string, handler func(amqp.Delivery)) error {
	mq.mu.RLock()
	ch := mq.ch
	mq.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("rabbitmq channel not available")
	}

	msgs, err := ch.Consume(
		queue,
		consumer,
		false, // auto-ack = false
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("start consuming: %w", err)
	}

	mq.log.Info(logger.Entry{
		Action:  "consumer_started",
		Message: fmt.Sprintf("consuming from queue: %s", queue),
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					mq.log.Info(logger.Entry{
						Action:  "consumer_stopped",
						Message: queue,
					})
					return
				}
				handler(msg)
			}
		}
	}()

	return nil
}

// Close закрывает подключение к RabbitMQ
func (mq *RabbitMQ) Close() {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.closed {
		return
	}

	mq.closed = true

	if mq.ch != nil {
		_ = mq.ch.Close()
	}
	if mq.conn != nil {
		_ = mq.conn.Close()
	}

	mq.log.Info(logger.Entry{Action: "rabbitmq_closed", Message: "connection closed"})
}
