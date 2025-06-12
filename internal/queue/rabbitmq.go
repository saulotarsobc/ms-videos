// Package queue contém a implementação para consumir mensagens de uma fila RabbitMQ
// As mensagens contêm informações sobre vídeos a serem processados
package queue

// Importações necessárias para consumir mensagens da fila
import (
	"context"       // Para gerenciar contexto
	"encoding/json" // Para decodificação de mensagens JSON
	"fmt"           // Para formatação de strings
	"log"           // Para logging

	amqp "github.com/rabbitmq/amqp091-go" // Cliente RabbitMQ
)

// VideoMessage representa a estrutura da mensagem contendo dados de vídeos
// Tags JSON especificam como os campos são mapeados de/para JSON
type VideoMessage struct {
	ID       string `json:"id"`       // Identificador do vídeo
	URL      string `json:"url"`      // URL de onde baixar o vídeo
	Filename string `json:"filename"` // Nome do arquivo de vídeo
}

// RabbitMQConsumer é responsável por conectar e consumir mensagens de uma fila RabbitMQ
type RabbitMQConsumer struct {
	conn      *amqp.Connection // Conexão com RabbitMQ
	ch        *amqp.Channel    // Canal de comunicação com RabbitMQ
	queueName string           // Nome da fila
}

// NewRabbitMQConsumer cria um novo consumidor RabbitMQ
// Configura a conexão, abre o canal e declara a fila
func NewRabbitMQConsumer(amqpURL, queueName string) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(amqpURL) // Estabelece conexão com RabbitMQ
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel() // Abre um canal na conexão
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declara a fila, garantindo que ela existe
	_, err = ch.QueueDeclare(
		queueName, // Nome da fila
		true,      // Durável (sobrevive a reinicializações)
		false,     // Não deletar quando não usável
		false,     // Não exclusiva (pode ser usada por outros consumidores)
		false,     // Sem espera
		nil,       // Sem argumentos adicionais
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	// Define QoS para processar uma mensagem por vez
	err = ch.Qos(
		1,     // Número de mensagens que o consumidor prefetch
		0,     // Tamanho do prefetch (0 = desabilitado)
		false, // Global (aplica para todos os consumidores do canal)
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &RabbitMQConsumer{ // Retorna o consumidor configurado
		conn:      conn,
		ch:        ch,
		queueName: queueName,
	}, nil
}

// StartConsuming começa a consumir mensagens e processa cada uma usando o handler fornecido
// Continua consumindo até que o contexto seja cancelado ou erro ocorra
func (c *RabbitMQConsumer) StartConsuming(ctx context.Context, handler func(VideoMessage) error) error {
	msgs, err := c.ch.Consume( // Inicia o consumo de mensagens
		c.queueName, // Nome da fila
		"",          // Nome do consumidor (gerado automaticamente se vazio)
		false,       // Auto-acknowledge desabilitado (manter controle manual)
		false,       // Exclusivo (apenas um consumidor)
		false,       // No local (não compartilhar entre máquinas)
		false,       // Sem espera
		nil,         // Sem argumentos adicionais
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	for { // Loop contínuo para processar mensagens conforme chegam
		select {
		case <-ctx.Done(): // Quando o contexto for cancelado
			log.Println("Context cancelled, stopping consumer")
			c.Close()
			return ctx.Err()
		case d, ok := <-msgs: // Mensagem recebida da fila
			if !ok { // Se o canal de mensagens está fechado
				log.Println("Messages channel closed")
				return nil
			}

			var videoMsg VideoMessage                                 // Declara uma variável do tipo VideoMessage
			if err := json.Unmarshal(d.Body, &videoMsg); err != nil { // Deserializa mensagem
				log.Printf("Failed to unmarshal message: %v", err)
				d.Nack(false, false) // Não re-enfileira mensagens malformadas
				continue
			}

			log.Printf("Received video message: ID=%s, URL=%s, Filename=%s", videoMsg.ID, videoMsg.URL, videoMsg.Filename)

			// Process the message usando handler fornecido
			if err := handler(videoMsg); err != nil {
				log.Printf("Failed to process video %s: %v", videoMsg.ID, err)
				d.Nack(false, true) // Re-enfileira em caso de erro de processamento
				continue
			}

			log.Printf("Successfully processed video %s", videoMsg.ID)
			d.Ack(false) // Confirmação de processamento
		}
	}
}

// Close encerra a conexão e o canal com RabbitMQ
func (c *RabbitMQConsumer) Close() {
	if c.ch != nil {
		c.ch.Close() // Fecha o canal se aberto
	}
	if c.conn != nil {
		c.conn.Close() // Fecha a conexão se aberta
	}

	if c.conn != nil {
		c.conn.Close()
	}
}
