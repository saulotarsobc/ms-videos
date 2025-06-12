// Package main é o ponto de entrada do microserviço de processamento de vídeos
// Este microserviço recebe mensagens de uma fila RabbitMQ contendo informações sobre vídeos
// e os processa para gerar streams HLS (HTTP Live Streaming) em diferentes resoluções
package main

// Importação das bibliotecas necessárias
import (
	"context"                           // Para controle de contexto e cancelamento
	"log"                               // Para logging/registros do sistema
	"ms-videos/internal/processor"      // Pacote interno para processamento de vídeos
	"ms-videos/internal/queue"          // Pacote interno para comunicação com filas
	"ms-videos/internal/storage"        // Pacote interno para armazenamento de arquivos
	"os"                                // Para interação com sistema operacional
	"os/signal"                         // Para captura de sinais do sistema
	"syscall"                           // Para constantes de sinais do sistema
)

// Função principal do programa - ponto de entrada da aplicação
func main() {
	log.Println("Starting ms-videos microservice...")

	// Obter variáveis de ambiente com valores padrão caso não estejam definidas
	// getEnv é uma função auxiliar que verifica se a variável existe, senão usa o valor padrão
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	minioEndpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	minioAccessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	minioSecretKey := getEnv("MINIO_SECRET_KEY", "minioadmin")
	minioBucket := getEnv("MINIO_BUCKET", "videos")

	// Inicializar cliente de armazenamento MinIO
	// MinIO é um sistema de armazenamento de objetos compatível com Amazon S3
	storageClient, err := storage.NewMinIOClient(
		minioEndpoint,
		minioAccessKey,
		minioSecretKey,
		minioBucket,
	)
	// Se houver erro na inicialização, o programa termina com log fatal
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	// Inicializar processador de vídeos
	// Injeta o cliente de armazenamento no processador (padrão de injeção de dependência)
	videoProcessor := processor.NewVideoProcessor(storageClient)

	// Inicializar consumidor da fila RabbitMQ
	// RabbitMQ é um broker de mensagens que permite comunicação assíncrona entre serviços
	queueConsumer, err := queue.NewRabbitMQConsumer(rabbitmqURL, "videos")
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ consumer: %v", err)
	}

	// Criar contexto para shutdown gracioso
	// Context em Go é usado para controlar cancelamento e timeouts
	ctx, cancel := context.WithCancel(context.Background())
	// defer garante que cancel() será executado quando main() terminar
	defer cancel()

	// Configurar captura de sinais para shutdown gracioso
	// Cria um canal para receber sinais do sistema operacional
	sigChan := make(chan os.Signal, 1)
	// Registra quais sinais queremos capturar (Ctrl+C e SIGTERM)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Executar em goroutine separada para não bloquear o programa principal
	// Goroutines são threads leves em Go
	go func() {
		// Aguarda receber um sinal
		<-sigChan
		log.Println("Shutdown signal received, initiating graceful shutdown...")
		// Cancela o contexto para sinalizar que o programa deve parar
		cancel()
	}()

	// Iniciar consumo de mensagens da fila
	// Passa o contexto e uma função callback para processar cada vídeo
	log.Println("Starting to consume messages from videos queue...")
	err = queueConsumer.StartConsuming(ctx, videoProcessor.ProcessVideo)
	if err != nil {
		log.Printf("Consumer stopped with error: %v", err)
	}

	log.Println("Microservice shutdown complete")
}

// Função auxiliar para obter variáveis de ambiente com valor padrão
// Em Go, funções podem retornar múltiplos valores
func getEnv(key, defaultValue string) string {
	// os.LookupEnv retorna o valor da variável e um boolean indicando se ela existe
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	// Se a variável não existir, retorna o valor padrão
	return defaultValue
}
