// Package processor contém a lógica principal para processamento de vídeos
// Responsável por baixar vídeos, converter para diferentes resoluções e gerar streams HLS
package processor

// Importações necessárias para o processamento de vídeos
import (
	"fmt"                               // Para formatação de strings
	"io"                                // Para operações de entrada/saída
	"log"                               // Para logging
	"ms-videos/internal/queue"          // Para estruturas de mensagens da fila
	"ms-videos/internal/storage"        // Para cliente de armazenamento
	"net/http"                          // Para downloads HTTP
	"os"                                // Para operações do sistema operacional
	"os/exec"                           // Para execução de comandos externos (ffmpeg)
	"path/filepath"                     // Para manipulação de caminhos de arquivos
	"strings"                           // Para manipulação de strings
)

// VideoProcessor é uma struct que encapsula a lógica de processamento de vídeos
// Em Go, structs são como classes em outras linguagens
type VideoProcessor struct {
	// storageClient é um ponteiro para o cliente MinIO
	// O * indica que é um ponteiro, não uma cópia da struct
	storageClient *storage.MinIOClient
}

// NewVideoProcessor é uma função construtora que cria uma nova instância de VideoProcessor
// Em Go, é comum usar funções New* como construtores
// O & retorna o endereço de memória da struct (cria um ponteiro)
func NewVideoProcessor(storageClient *storage.MinIOClient) *VideoProcessor {
	return &VideoProcessor{
		storageClient: storageClient,
	}
}

// ProcessVideo controla o fluxo de trabalho para processar um vídeo incluindo
// download, processamento de resoluções, criação de playlist mestre e upload
func (vp *VideoProcessor) ProcessVideo(msg queue.VideoMessage) error {
	log.Printf("Starting processing video %s", msg.ID)

	// Cria um diretório temporário para armazenar arquivos durante o processamento
	// O padrão "video_{ID}_*" ajuda a identificar facilmente os arquivos temporários
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("video_%s_*", msg.ID))
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	// defer com função anônima para limpeza automática dos arquivos temporários
	// Isso garante que os arquivos serão removidos mesmo se houver erro
	defer func() {
		log.Printf("Cleaning up temporary files for video %s", msg.ID)
		os.RemoveAll(tempDir) // Remove recursivamente o diretório e conteúdo
	}()

	// Faz o download do vídeo original da URL fornecida
	originalPath, err := vp.downloadVideo(msg.URL, tempDir, msg.Filename)
	if err != nil {
		return fmt.Errorf("failed to download video: %w", err)
	}

	// Processa o vídeo em diferentes resoluções para streaming adaptativo
	// HLS permite que o player escolha a melhor qualidade baseada na conexão
	resolutions := []string{"1080p", "720p", "480p", "360p"}
	for _, resolution := range resolutions { // range itera sobre cada elemento do slice
		log.Printf("Processing video %s to %s resolution", msg.ID, resolution)
		err := vp.processResolution(originalPath, tempDir, msg.ID, resolution)
		if err != nil {
			return fmt.Errorf("failed to process %s resolution: %w", resolution, err)
		}
	}

	// Cria a playlist mestre que referencia todas as resoluções
	// Esta é a entrada principal para o streaming HLS
	log.Printf("Creating master playlist for video %s", msg.ID)
	err = vp.createMasterPlaylist(tempDir, msg.ID, resolutions)
	if err != nil {
		return fmt.Errorf("failed to create master playlist: %w", err)
	}

	// Faz upload de todos os arquivos HLS gerados para o armazenamento
	// Isso inclui playlists (.m3u8) e segmentos de vídeo (.ts)
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

func (vp *VideoProcessor) createMasterPlaylist(tempDir, videoID string, resolutions []string) error {
	hlsDir := filepath.Join(tempDir, "hls")
	masterPath := filepath.Join(hlsDir, "master.m3u8")

	file, err := os.Create(masterPath)
	if err != nil {
		return fmt.Errorf("failed to create master playlist: %w", err)
	}
	defer file.Close()

	// Write HLS master playlist header
	_, err = file.WriteString("#EXTM3U\n")
	if err != nil {
		return err
	}
	_, err = file.WriteString("#EXT-X-VERSION:3\n\n")
	if err != nil {
		return err
	}

	// Add each resolution as a stream variant
	for _, resolution := range resolutions {
		bandwidth, width, height := getStreamInfo(resolution)
		
		// Write stream info
		streamInfo := fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n", bandwidth, width, height)
		_, err = file.WriteString(streamInfo)
		if err != nil {
			return err
		}
		
		// Write playlist path
		playlistPath := fmt.Sprintf("%s/playlist.m3u8\n", resolution)
		_, err = file.WriteString(playlistPath)
		if err != nil {
			return err
		}
		
		// Add blank line for readability
		_, err = file.WriteString("\n")
		if err != nil {
			return err
		}
	}

	log.Printf("Master playlist created successfully for video %s", videoID)
	return nil
}

func getStreamInfo(resolution string) (bandwidth, width, height int) {
	switch resolution {
	case "1080p":
		return 5000000, 1920, 1080 // 5 Mbps
	case "720p":
		return 3000000, 1280, 720  // 3 Mbps
	case "480p":
		return 1500000, 854, 480   // 1.5 Mbps
	case "360p":
		return 800000, 640, 360    // 800 Kbps
	default:
		return 1000000, 640, 360   // Default
	}
}

