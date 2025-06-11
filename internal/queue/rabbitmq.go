package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type VideoMessage struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

type RabbitMQConsumer struct {
	conn      *amqp.Connection
	ch        *amqp.Channel
	queueName string
}

func NewRabbitMQConsumer(amqpURL, queueName string) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare the queue
	_, err = ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	// Set QoS to process one message at a time
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &RabbitMQConsumer{
		conn:      conn,
		ch:        ch,
		queueName: queueName,
	}, nil
}

func (c *RabbitMQConsumer) StartConsuming(ctx context.Context, handler func(VideoMessage) error) error {
	msgs, err := c.ch.Consume(
		c.queueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping consumer")
			c.Close()
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				log.Println("Messages channel closed")
				return nil
			}

			var videoMsg VideoMessage
			if err := json.Unmarshal(d.Body, &videoMsg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				d.Nack(false, false) // Don't requeue malformed messages
				continue
			}

			log.Printf("Received video message: ID=%s, URL=%s, Filename=%s", videoMsg.ID, videoMsg.URL, videoMsg.Filename)

			// Process the message
			if err := handler(videoMsg); err != nil {
				log.Printf("Failed to process video %s: %v", videoMsg.ID, err)
				d.Nack(false, true) // Requeue on processing error
				continue
			}

			log.Printf("Successfully processed video %s", videoMsg.ID)
			d.Ack(false)
		}
	}
}

func (c *RabbitMQConsumer) Close() {
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
