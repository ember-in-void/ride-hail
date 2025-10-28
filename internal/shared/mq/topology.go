package mq

import (
	"context"
	"fmt"

	"ridehail/internal/shared/logger"
)

// SetupTopology создает все exchanges, queues и bindings согласно ТЗ
func SetupTopology(ctx context.Context, mq *RabbitMQ, log *logger.Logger) error {
	ch := mq.Channel()
	if ch == nil {
		return fmt.Errorf("rabbitmq channel not available")
	}

	// 1. Exchange: ride_topic (topic)
	if err := ch.ExchangeDeclare(
		"ride_topic", // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // args
	); err != nil {
		return fmt.Errorf("declare ride_topic: %w", err)
	}

	// 2. Exchange: driver_topic (topic)
	if err := ch.ExchangeDeclare(
		"driver_topic",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare driver_topic: %w", err)
	}

	// 3. Exchange: location_fanout (fanout)
	if err := ch.ExchangeDeclare(
		"location_fanout",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare location_fanout: %w", err)
	}

	// 4. Очереди для ride_topic
	rideQueues := []string{
		"ride.requested",
		"ride.matched",
		"ride.completed",
		"ride.cancelled",
	}
	for _, q := range rideQueues {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare queue %s: %w", q, err)
		}
		// Binding к ride_topic
		routingKey := q // ride.requested, ride.matched, etc.
		if err := ch.QueueBind(q, routingKey, "ride_topic", false, nil); err != nil {
			return fmt.Errorf("bind queue %s: %w", q, err)
		}
	}

	// 5. Очереди для driver_topic
	driverQueues := []string{
		"driver.status_changed",
		"driver.location_updated",
	}
	for _, q := range driverQueues {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare queue %s: %w", q, err)
		}
		routingKey := q
		if err := ch.QueueBind(q, routingKey, "driver_topic", false, nil); err != nil {
			return fmt.Errorf("bind queue %s: %w", q, err)
		}
	}

	// 6. Очередь для location_fanout (каждый сервис создаст свою эксклюзивную очередь при consume)
	// Здесь создаём общую очередь для примера, но в реальности fanout используется с auto-delete очередями
	if _, err := ch.QueueDeclare("location.broadcast", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare location.broadcast: %w", err)
	}
	if err := ch.QueueBind("location.broadcast", "", "location_fanout", false, nil); err != nil {
		return fmt.Errorf("bind location.broadcast: %w", err)
	}

	log.Info(logger.Entry{
		Action:  "topology_setup_complete",
		Message: "all exchanges and queues created",
	})

	return nil
}
