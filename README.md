# ms-videos

An event-driven microservice that processes videos by converting them to multiple resolutions and HLS format.

## Features

- Listens to RabbitMQ queue for video processing requests
- Downloads videos from public URLs
- Converts videos to 1080p, 720p, 480p, and 360p resolutions
- Fragments videos using HLS format (.m3u8 + .ts segments)
- Uploads processed files to MinIO/S3 storage
- Graceful shutdown handling
- One video processed at a time (no parallel processing)

## Message Format

```json
{
  "id": "uuid-string",
  "url": "https://example.com/video.mp4",
  "filename": "video.mp4"
}
```

## Environment Variables

- `RABBITMQ_URL`: RabbitMQ connection string (default: `amqp://guest:guest@localhost:5672/`)
- `MINIO_ENDPOINT`: MinIO endpoint (default: `localhost:9000`)
- `MINIO_ACCESS_KEY`: MinIO access key (default: `minioadmin`)
- `MINIO_SECRET_KEY`: MinIO secret key (default: `minioadmin`)
- `MINIO_BUCKET`: MinIO bucket name (default: `videos`)

## Prerequisites

- Go 1.21+
- Docker and Docker Compose
- FFmpeg (for local development)

## Development

1. Clone the repository
2. Copy `.env` file and adjust if needed
3. Start the infrastructure:
   ```bash
   docker-compose up rabbitmq minio minio-setup -d
   ```
4. Run the service locally:
   ```bash
   go run cmd/ms-videos/main.go
   ```

## Build & Deploy

### Docker Build

1. **Build the Docker image:**
   ```bash
   docker build -t ms-videos:latest .
   ```

2. **Tag for registry (optional):**
   ```bash
   docker tag ms-videos:latest your-registry.com/ms-videos:latest
   ```

3. **Push to registry (optional):**
   ```bash
   docker push your-registry.com/ms-videos:latest
   ```

### Local Binary Build

1. **Build for current platform:**
   ```bash
   go build -o ms-videos cmd/ms-videos/main.go
   ```

2. **Build for Linux (for server deployment):**
   ```bash
   GOOS=linux GOARCH=amd64 go build -o ms-videos-linux cmd/ms-videos/main.go
   ```

### Production Deployment

#### Using Docker Compose (Recommended)

1. **Deploy all services:**
   ```bash
   docker-compose up -d
   ```

2. **Check services status:**
   ```bash
   docker-compose ps
   ```

3. **View logs:**
   ```bash
   docker-compose logs -f ms-videos
   ```

#### Manual Deployment

1. **Ensure FFmpeg is installed on the server:**
   ```bash
   # Ubuntu/Debian
   sudo apt update && sudo apt install -y ffmpeg
   
   # CentOS/RHEL
   sudo yum install -y epel-release
   sudo yum install -y ffmpeg
   ```

2. **Set production environment variables:**
   ```bash
   export RABBITMQ_URL="amqp://user:password@your-rabbitmq-server:5672/"
   export MINIO_ENDPOINT="your-minio-server:9000"
   export MINIO_ACCESS_KEY="your-access-key"
   export MINIO_SECRET_KEY="your-secret-key"
   export MINIO_BUCKET="videos"
   ```

3. **Run the binary:**
   ```bash
   ./ms-videos-linux
   ```

#### Using systemd (Linux)

1. **Create systemd service file:**
   ```bash
   sudo nano /etc/systemd/system/ms-videos.service
   ```

2. **Service configuration:**
   ```ini
   [Unit]
   Description=MS Videos Service
   After=network.target
   
   [Service]
   Type=simple
   User=videos
   WorkingDirectory=/opt/ms-videos
   ExecStart=/opt/ms-videos/ms-videos
   Restart=always
   RestartSec=5
   
   Environment=RABBITMQ_URL=amqp://user:password@rabbitmq-server:5672/
   Environment=MINIO_ENDPOINT=minio-server:9000
   Environment=MINIO_ACCESS_KEY=your-access-key
   Environment=MINIO_SECRET_KEY=your-secret-key
   Environment=MINIO_BUCKET=videos
   
   [Install]
   WantedBy=multi-user.target
   ```

3. **Enable and start service:**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable ms-videos
   sudo systemctl start ms-videos
   sudo systemctl status ms-videos
   ```

### Production Considerations

- **Resource Requirements:** Ensure adequate CPU, memory, and disk space for video processing
- **FFmpeg:** Must be available in the system PATH
- **Network:** Stable internet connection for downloading source videos
- **Storage:** Configure MinIO/S3 with appropriate backup and retention policies
- **Monitoring:** Implement logging and monitoring for the service
- **Security:** Use strong credentials and secure network configurations
- **Scaling:** Consider horizontal scaling for high-volume processing

### Environment Configuration for Production

Create a `.env` file for production:

```env
RABBITMQ_URL=amqp://production-user:strong-password@rabbitmq.example.com:5672/
MINIO_ENDPOINT=minio.example.com:9000
MINIO_ACCESS_KEY=production-access-key
MINIO_SECRET_KEY=production-secret-key
MINIO_BUCKET=videos
```

## Services

- **ms-videos**: Main microservice
- **RabbitMQ**: Message queue (Management UI: http://localhost:15672)
- **MinIO**: Object storage (Console: http://localhost:9001)

## Testing

### Running the Application

1. **Start the infrastructure:**

   ```bash
   docker-compose up rabbitmq minio minio-setup -d
   ```

2. **Run the microservice:**

   ```bash
   go run cmd/ms-videos/main.go
   ```

3. **Send a test video for processing:**
   ```bash
   go run test/send_message.go test-123 "https://sample-videos.com/zip/10/mp4/SampleVideo_1280x720_1mb.mp4" sample.mp4
   ```

### Test Script Usage

The `test/send_message.go` script allows you to send video processing requests to the RabbitMQ queue:

```bash
go run test/send_message.go <video-id> <video-url> <filename>
```

**Parameters:**

- `video-id`: Unique identifier for the video (will be used as folder name in MinIO)
- `video-url`: Public URL to download the video file
- `filename`: Original filename for the video

**Examples:**

```bash
# Process a sample video
go run test/send_message.go video-001 "https://www.learningcontainer.com/wp-content/uploads/2020/05/sample-mp4-file.mp4" sample.mp4
```

### Monitoring

After sending a message, you can monitor the processing:

1. **Watch the microservice logs** for processing updates
2. **RabbitMQ Management UI** (http://localhost:15672) - Login: `guest/guest`
   - View queue status and message consumption
3. **MinIO Console** (http://localhost:9001) - Login: `minioadmin/minioadmin`
   - Check processed video files in the `videos` bucket

### Expected Output

When processing is successful, you'll see logs like:

```
2025/06/11 15:45:33 Received video message: ID=test-123, URL=https://...
2025/06/11 15:45:34 Downloading video from URL: https://...
2025/06/11 15:45:35 Processing video test-123 to 1080p resolution
2025/06/11 15:45:40 Processing video test-123 to 720p resolution
2025/06/11 15:45:45 Processing video test-123 to 480p resolution
2025/06/11 15:45:50 Processing video test-123 to 360p resolution
2025/06/11 15:45:55 Uploading HLS files for video test-123
2025/06/11 15:46:00 Successfully processed video test-123
```

## Output Structure

Processed videos are stored in MinIO with the following structure:

```
videos/
├── {video-id}/
│   ├── 1080p/
│   │   ├── playlist.m3u8
│   │   ├── segment_000.ts
│   │   ├── segment_001.ts
│   │   └── ...
│   ├── 720p/
│   │   ├── playlist.m3u8
│   │   ├── segment_000.ts
│   │   ├── segment_001.ts
│   │   └── ...
│   ├── 480p/
│   │   ├── playlist.m3u8
│   │   └── segments...
│   └── 360p/
│       ├── playlist.m3u8
│       └── segments...
```
