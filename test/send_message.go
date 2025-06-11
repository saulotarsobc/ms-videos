package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

type VideoMessage struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run send_message.go <video-id> <video-url> <filename>")
		fmt.Println("Example: go run send_message.go test-123 https://sample-videos.com/zip/10/mp4/SampleVideo_1280x720_1mb.mp4 sample.mp4")
		os.Exit(1)
	}

	videoID := os.Args[1]
	videoURL := os.Args[2]
	filename := os.Args[3]

	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare the queue
	_, err = ch.QueueDeclare(
		"videos", // name
		true,     // durable
		false,    // delete when unused
		false,    // exclusive
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// Create message
	msg := VideoMessage{
		ID:       videoID,
		URL:      videoURL,
		Filename: filename,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Failed to marshal message: %v", err)
	}

	// Publish message
	err = ch.Publish(
		"",       // exchange
		"videos", // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		log.Fatalf("Failed to publish a message: %v", err)
	}

	fmt.Printf("✅ Sent message: %s\n", string(body))
}

