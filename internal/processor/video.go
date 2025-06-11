package processor

import (
	"fmt"
	"io"
	"log"
	"ms-videos/internal/queue"
	"ms-videos/internal/storage"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type VideoProcessor struct {
	storageClient *storage.MinIOClient
}

func NewVideoProcessor(storageClient *storage.MinIOClient) *VideoProcessor {
	return &VideoProcessor{
		storageClient: storageClient,
	}
}

func (vp *VideoProcessor) ProcessVideo(msg queue.VideoMessage) error {
	log.Printf("Starting processing video %s", msg.ID)

	// Create temporary directory for this video
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("video_%s_*", msg.ID))
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		log.Printf("Cleaning up temporary files for video %s", msg.ID)
		os.RemoveAll(tempDir)
	}()

	// Download the original video
	originalPath, err := vp.downloadVideo(msg.URL, tempDir, msg.Filename)
	if err != nil {
		return fmt.Errorf("failed to download video: %w", err)
	}

	// Process video to different resolutions
	resolutions := []string{"1080p", "720p", "480p", "360p"}
	for _, resolution := range resolutions {
		log.Printf("Processing video %s to %s resolution", msg.ID, resolution)
		err := vp.processResolution(originalPath, tempDir, msg.ID, resolution)
		if err != nil {
			return fmt.Errorf("failed to process %s resolution: %w", resolution, err)
		}
	}

	// Upload all HLS files to MinIO
	log.Printf("Uploading HLS files for video %s", msg.ID)
	err = vp.uploadHLSFiles(tempDir, msg.ID)
	if err != nil {
		return fmt.Errorf("failed to upload HLS files: %w", err)
	}

	log.Printf("Successfully processed video %s", msg.ID)
	return nil
}

func (vp *VideoProcessor) downloadVideo(url, tempDir, filename string) (string, error) {
	log.Printf("Downloading video from URL: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download video: status %d", resp.StatusCode)
	}

	filePath := filepath.Join(tempDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save video: %w", err)
	}

	log.Printf("Video downloaded successfully to %s", filePath)
	return filePath, nil
}

func (vp *VideoProcessor) processResolution(inputPath, tempDir, videoID, resolution string) error {
	var height string
	switch resolution {
	case "1080p":
		height = "1080"
	case "720p":
		height = "720"
	case "480p":
		height = "480"
	case "360p":
		height = "360"
	default:
		return fmt.Errorf("unsupported resolution: %s", resolution)
	}

	// Create output directory for this resolution
	outputDir := filepath.Join(tempDir, "hls", resolution)
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate HLS files using ffmpeg
	playlistPath := filepath.Join(outputDir, "playlist.m3u8")
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=-2:%s", height), // Maintain aspect ratio
		"-c:v", "libx264",
		"-c:a", "aac",
		"-hls_time", "10", // 10 second segments
		"-hls_list_size", "0", // Keep all segments in playlist
		"-hls_segment_filename", filepath.Join(outputDir, "segment_%03d.ts"),
		"-f", "hls",
		playlistPath,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg failed for %s: %w", resolution, err)
	}

	log.Printf("Successfully processed video to %s resolution", resolution)
	return nil
}

func (vp *VideoProcessor) uploadHLSFiles(tempDir, videoID string) error {
	hlsDir := filepath.Join(tempDir, "hls")

	// Walk through all HLS files and upload them
	err := filepath.Walk(hlsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Only upload .m3u8 and .ts files
		ext := filepath.Ext(path)
		if ext != ".m3u8" && ext != ".ts" {
			return nil
		}

		// Generate object key (relative path from hls directory)
		relPath, err := filepath.Rel(hlsDir, path)
		if err != nil {
			return err
		}

		// Convert Windows paths to Unix-style for object keys
		relPath = strings.ReplaceAll(relPath, "\\", "/")
		objectKey := fmt.Sprintf("%s/%s", videoID, relPath)

		// Set content type based on file extension
		var contentType string
		switch ext {
		case ".m3u8":
			contentType = "application/vnd.apple.mpegurl"
		case ".ts":
			contentType = "video/mp2t"
		}

		log.Printf("Uploading file: %s as %s", path, objectKey)
		err = vp.storageClient.UploadFile(path, objectKey, contentType)
		if err != nil {
			return fmt.Errorf("failed to upload %s: %w", objectKey, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to upload HLS files: %w", err)
	}

	log.Printf("All HLS files uploaded successfully for video %s", videoID)
	return nil
}

