// Package main é um utilitário de teste para enviar mensagens de vídeo para a fila RabbitMQ
// Este arquivo é usado para testar o microserviço enviando mensagens de processamento
package main

// Importações necessárias para o programa de teste
import (
	"encoding/json" // Para serialização da mensagem em JSON
	"fmt"           // Para formatação e impressão de texto
	"log"           // Para logging de erros
	"os"            // Para acessar argumentos da linha de comando

	amqp "github.com/rabbitmq/amqp091-go" // Cliente RabbitMQ
)

// VideoMessage representa a estrutura da mensagem que será enviada
// Deve ser idêntica à estrutura definida no pacote queue
type VideoMessage struct {
	ID       string `json:"id"`       // Identificador único do vídeo
	URL      string `json:"url"`      // URL de onde baixar o vídeo
	Filename string `json:"filename"` // Nome do arquivo do vídeo
}

// Função principal que executa o programa de teste
func main() {
	// Verifica se foram fornecidos argumentos suficientes na linha de comando
	// os.Args[0] é o nome do programa, então precisamos de pelo menos 4 elementos
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run send_message.go <video-id> <video-url> <filename>")
		fmt.Println("Example: go run send_message.go test-123 https://sample-videos.com/zip/10/mp4/SampleVideo_1280x720_1mb.mp4 sample.mp4")
		os.Exit(1) // Sai do programa com código de erro
	}

	// Extrai os argumentos da linha de comando
	videoID := os.Args[1]  // Primeiro argumento: ID do vídeo
	videoURL := os.Args[2] // Segundo argumento: URL do vídeo
	filename := os.Args[3] // Terceiro argumento: nome do arquivo

	// Conecta ao RabbitMQ usando URL padrão para desenvolvimento local
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	// defer garante que a conexão será fechada ao fim da função
	defer conn.Close()

	// Abre um canal de comunicação com RabbitMQ
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declara a fila "videos" garantindo que ela existe
	_, err = ch.QueueDeclare(
		"videos", // Nome da fila
		true,     // Durável (sobrevive a reinicializações do servidor)
		false,    // Não deletar quando não utilizada
		false,    // Não exclusiva (outros podem usar)
		false,    // Não aguardar confirmação
		nil,      // Argumentos adicionais
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// Cria a estrutura da mensagem com os dados fornecidos
	msg := VideoMessage{
		ID:       videoID,
		URL:      videoURL,
		Filename: filename,
	}

	// Serializa a mensagem para JSON
	body, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Failed to marshal message: %v", err)
	}

	// Publica a mensagem na fila "videos"
	err = ch.Publish(
		"",       // Exchange (usar o padrão)
		"videos", // Chave de roteamento (nome da fila)
		false,    // Obrigatório (retornar erro se não conseguir entregar)
		false,    // Imediato (entregar imediatamente ou falhar)
		amqp.Publishing{
			ContentType: "application/json", // Tipo do conteúdo
			Body:        body,               // Corpo da mensagem em bytes
		})
	if err != nil {
		log.Fatalf("Failed to publish a message: %v", err)
	}

	// Confirmação de sucesso
	fmt.Printf("✅ Sent message: %s\n", string(body))
}
