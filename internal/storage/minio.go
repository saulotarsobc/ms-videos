// Package storage contém implementações para armazenamento de arquivos
// Este pacote usa MinIO (compatível com Amazon S3) para armazenar arquivos processados
package storage

// Importações necessárias para armazenamento MinIO
import (
	"context" // Para controle de contexto
	"fmt"     // Para formatação de strings
	"log"     // Para logging
	"os"      // Para operações de arquivo

	"github.com/minio/minio-go/v7"                 // Cliente MinIO
	"github.com/minio/minio-go/v7/pkg/credentials" // Credenciais MinIO
)

// MinIOClient é um wrapper ao redor do cliente MinIO
// Encapsula as operações de armazenamento de objetos
type MinIOClient struct {
	client     *minio.Client // Cliente MinIO nativo
	bucketName string        // Nome do bucket onde armazenar os arquivos
}

// NewMinIOClient cria e configura um novo cliente MinIO
// Inicializa a conexão e garante que o bucket existe
func NewMinIOClient(endpoint, accessKey, secretKey, bucketName string) (*MinIOClient, error) {
	// Inicializa o cliente MinIO com as credenciais fornecidas
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""), // Credenciais estáticas
		Secure: false,                                             // Usa HTTP em vez de HTTPS para desenvolvimento local
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Cria nossa estrutura wrapper
	client := &MinIOClient{
		client:     minioClient,
		bucketName: bucketName,
	}

	// Garante que o bucket existe (cria se não existir)
	err = client.ensureBucketExists()
	if err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	log.Printf("MinIO client initialized successfully for bucket: %s", bucketName)
	return client, nil
}

// ensureBucketExists verifica se o bucket existe e o cria se necessário
// É um método privado (começa com letra minúscula)
func (mc *MinIOClient) ensureBucketExists() error {
	ctx := context.Background() // Cria contexto básico

	// Verifica se o bucket já existe
	exists, err := mc.client.BucketExists(ctx, mc.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		// Cria o bucket se ele não existir
		err = mc.client.MakeBucket(ctx, mc.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Created bucket: %s", mc.bucketName)
	} else {
		log.Printf("Bucket already exists: %s", mc.bucketName)
	}

	return nil
}

// UploadFile faz o upload de um arquivo local para o armazenamento MinIO
// filePath: caminho do arquivo local
// objectKey: nome/chave do objeto no armazenamento
// contentType: tipo MIME do arquivo (ex: "video/mp4", "application/vnd.apple.mpegurl")
func (mc *MinIOClient) UploadFile(filePath, objectKey, contentType string) error {
	ctx := context.Background() // Contexto para a operação

	// Abre o arquivo local para leitura
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	// defer garante que o arquivo será fechado ao fim da função
	defer file.Close()

	// Obtém informações do arquivo (principalmente o tamanho)
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Faz o upload do arquivo para MinIO
	_, err = mc.client.PutObject(ctx, mc.bucketName, objectKey, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: contentType, // Define o tipo de conteúdo para o arquivo
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	log.Printf("Successfully uploaded: %s", objectKey)
	return nil
}
