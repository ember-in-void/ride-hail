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

// NewRabbitMQ создает подключение к RabbitMQ
func NewRabbitMQ(ctx context.Context, cfg config.MQConfig, log *logger.Logger) (*RabbitMQ, error) {
	url := cfg.AMQPURL()

	mq := &RabbitMQ{
		url: url,
		log: log,
	}

	if err := mq.connect(ctx); err != nil {
		return nil, err
	}

	log.Info(logger.Entry{
		Action:  "rabbitmq_connected",
		Message: fmt.Sprintf("connected to %s:%d", cfg.Host, cfg.Port),
	})

	return mq, nil
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
