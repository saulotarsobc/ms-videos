package main

import (
	"context"
	"log"
	"ms-videos/internal/processor"
	"ms-videos/internal/queue"
	"ms-videos/internal/storage"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Starting ms-videos microservice...")

	// Get environment variables
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	minioEndpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	minioAccessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	minioSecretKey := getEnv("MINIO_SECRET_KEY", "minioadmin")
	minioBucket := getEnv("MINIO_BUCKET", "videos")

	// Initialize storage client
	storageClient, err := storage.NewMinIOClient(
		minioEndpoint,
		minioAccessKey,
		minioSecretKey,
		minioBucket,
	)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	// Initialize video processor
	videoProcessor := processor.NewVideoProcessor(storageClient)

	// Initialize queue consumer
	queueConsumer, err := queue.NewRabbitMQConsumer(rabbitmqURL, "videos")
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ consumer: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received, initiating graceful shutdown...")
		cancel()
	}()

	// Start consuming messages
	log.Println("Starting to consume messages from videos queue...")
	err = queueConsumer.StartConsuming(ctx, videoProcessor.ProcessVideo)
	if err != nil {
		log.Printf("Consumer stopped with error: %v", err)
	}

	log.Println("Microservice shutdown complete")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
