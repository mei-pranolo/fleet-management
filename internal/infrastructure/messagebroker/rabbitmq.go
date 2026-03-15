package messagebroker

import (
	"encoding/json"
	"fmt"
	"log"

	"fleet-management/internal/config"
	"fleet-management/internal/module/vehicle"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQClient wraps an AMQP connection and channel for publishing and consuming
type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	cfg     *config.RabbitMQConfig
}

// NewRabbitMQClient establishes a connection, declares the exchange and queue, and binds them
func NewRabbitMQClient(cfg *config.RabbitMQConfig) (*RabbitMQClient, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare durable exchange
	if err := ch.ExchangeDeclare(
		cfg.Exchange,
		cfg.ExchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare durable queue
	if _, err := ch.QueueDeclare(
		cfg.Queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	if err := ch.QueueBind(
		cfg.Queue,
		cfg.RoutingKey,
		cfg.Exchange,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	return &RabbitMQClient{
		conn:    conn,
		channel: ch,
		cfg:     cfg,
	}, nil
}

// Publish serializes a GeofenceEvent and publishes it to the configured exchange
func (c *RabbitMQClient) Publish(event *vehicle.GeofenceEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal geofence event: %w", err)
	}

	err = c.channel.Publish(
		c.cfg.Exchange,
		c.cfg.RoutingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish geofence event: %w", err)
	}
	log.Printf("[RabbitMQ] published event: %v", event)

	return nil
}

// Consume starts consuming messages from the geofence_alerts queue
func (c *RabbitMQClient) Consume(handler func(event *vehicle.GeofenceEvent)) error {
	msgs, err := c.channel.Consume(
		c.cfg.Queue,
		c.cfg.ConsumerTag,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for d := range msgs {
			var event vehicle.GeofenceEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("[RabbitMQ] failed to unmarshal message: %v", err)
				d.Nack(false, false)
				continue
			}
			handler(&event)
			d.Ack(false)
		}
	}()

	return nil
}

// Close gracefully closes the channel and connection
func (c *RabbitMQClient) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
