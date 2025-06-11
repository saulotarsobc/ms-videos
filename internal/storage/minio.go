package storage

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	client     *minio.Client
	bucketName string
}

func NewMinIOClient(endpoint, accessKey, secretKey, bucketName string) (*MinIOClient, error) {
	// Initialize MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // Use HTTP for local development
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	client := &MinIOClient{
		client:     minioClient,
		bucketName: bucketName,
	}

	// Ensure bucket exists
	err = client.ensureBucketExists()
	if err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	log.Printf("MinIO client initialized successfully for bucket: %s", bucketName)
	return client, nil
}

func (mc *MinIOClient) ensureBucketExists() error {
	ctx := context.Background()

	// Check if bucket exists
	exists, err := mc.client.BucketExists(ctx, mc.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		// Create bucket
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

func (mc *MinIOClient) UploadFile(filePath, objectKey, contentType string) error {
	ctx := context.Background()

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Upload the file
	_, err = mc.client.PutObject(ctx, mc.bucketName, objectKey, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	log.Printf("Successfully uploaded: %s", objectKey)
	return nil
}

