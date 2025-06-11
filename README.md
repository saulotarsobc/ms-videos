# ms-videos

An event-driven microservice that processes videos by converting them to multiple resolutions and HLS format.

## Features

- Listens to RabbitMQ queue for video processing requests
- Downloads videos from public URLs
- Converts videos to 720p, 480p, and 360p resolutions
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

## Production Deployment

```bash
docker-compose up -d
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
2025/06/11 15:45:35 Processing video test-123 to 720p resolution
2025/06/11 15:45:40 Processing video test-123 to 480p resolution
2025/06/11 15:45:45 Processing video test-123 to 360p resolution
2025/06/11 15:45:50 Uploading HLS files for video test-123
2025/06/11 15:45:55 Successfully processed video test-123
```

## Output Structure

Processed videos are stored in MinIO with the following structure:

```
videos/
├── {video-id}/
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
