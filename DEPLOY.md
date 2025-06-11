# Deployment Guide

This guide provides detailed instructions for deploying the ms-videos microservice in production environments.

## Quick Start

### 1. Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/saulotarsobc/ms-videos.git
cd ms-videos

# Build and deploy
docker-compose up -d

# Check status
docker-compose ps
```

### 2. Manual Build and Deploy

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o ms-videos-linux cmd/ms-videos/main.go

# Transfer to server and run
scp ms-videos-linux user@server:/opt/ms-videos/
ssh user@server
cd /opt/ms-videos
./ms-videos-linux
```

## Production Environment Setup

### Prerequisites

1. **Server Requirements:**
   - Linux server (Ubuntu 20.04+ or CentOS 8+)
   - Minimum 2 CPU cores, 4GB RAM
   - 50GB+ storage space
   - Docker and Docker Compose (for containerized deployment)

2. **External Services:**
   - RabbitMQ server
   - MinIO/S3 storage
   - Network access to video URLs

### Infrastructure Setup

#### Option 1: All-in-One Docker Compose

1. **Create production docker-compose.yml:**

```yaml
version: '3.8'

services:
  ms-videos:
    build: .
    restart: unless-stopped
    environment:
      - RABBITMQ_URL=amqp://videos:secure-password@rabbitmq:5672/
      - MINIO_ENDPOINT=minio:9000
      - MINIO_ACCESS_KEY=production-access-key
      - MINIO_SECRET_KEY=production-secret-key
      - MINIO_BUCKET=videos
    depends_on:
      - rabbitmq
      - minio
    volumes:
      - /tmp:/tmp  # For temporary files

  rabbitmq:
    image: rabbitmq:3-management
    restart: unless-stopped
    environment:
      - RABBITMQ_DEFAULT_USER=videos
      - RABBITMQ_DEFAULT_PASS=secure-password
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq
    ports:
      - "15672:15672"  # Management UI

  minio:
    image: minio/minio:latest
    restart: unless-stopped
    command: server /data --console-address ":9001"
    environment:
      - MINIO_ROOT_USER=production-access-key
      - MINIO_ROOT_PASSWORD=production-secret-key
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"
      - "9001:9001"

  minio-setup:
    image: minio/mc:latest
    depends_on:
      - minio
    entrypoint: >
      /bin/sh -c "
      sleep 10;
      mc alias set minio http://minio:9000 production-access-key production-secret-key;
      mc mb minio/videos --ignore-existing;
      mc policy set public minio/videos;
      "

volumes:
  rabbitmq_data:
  minio_data:
```

2. **Deploy:**

```bash
docker-compose -f docker-compose.prod.yml up -d
```

#### Option 2: External Services

If using external RabbitMQ and MinIO/S3:

```yaml
version: '3.8'

services:
  ms-videos:
    build: .
    restart: unless-stopped
    environment:
      - RABBITMQ_URL=amqp://user:password@your-rabbitmq.com:5672/
      - MINIO_ENDPOINT=your-minio.com:9000
      - MINIO_ACCESS_KEY=${MINIO_ACCESS_KEY}
      - MINIO_SECRET_KEY=${MINIO_SECRET_KEY}
      - MINIO_BUCKET=videos
    volumes:
      - /tmp:/tmp
```

### Binary Deployment (Without Docker)

#### 1. Server Setup

```bash
# Install FFmpeg
sudo apt update
sudo apt install -y ffmpeg

# Create user and directories
sudo useradd -r -s /bin/false videos
sudo mkdir -p /opt/ms-videos
sudo chown videos:videos /opt/ms-videos
```

#### 2. Build and Deploy

```bash
# On development machine
GOOS=linux GOARCH=amd64 go build -o ms-videos-linux cmd/ms-videos/main.go

# Transfer to server
scp ms-videos-linux user@server:/opt/ms-videos/ms-videos
ssh user@server
sudo chown videos:videos /opt/ms-videos/ms-videos
sudo chmod +x /opt/ms-videos/ms-videos
```

#### 3. Systemd Service

Create `/etc/systemd/system/ms-videos.service`:

```ini
[Unit]
Description=MS Videos Processing Service
After=network.target
Requires=network.target

[Service]
Type=simple
User=videos
Group=videos
WorkingDirectory=/opt/ms-videos
ExecStart=/opt/ms-videos/ms-videos
Restart=always
RestartSec=5
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=30

# Environment variables
Environment=RABBITMQ_URL=amqp://user:password@rabbitmq.example.com:5672/
Environment=MINIO_ENDPOINT=minio.example.com:9000
Environment=MINIO_ACCESS_KEY=your-access-key
Environment=MINIO_SECRET_KEY=your-secret-key
Environment=MINIO_BUCKET=videos

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/tmp

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=ms-videos

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable ms-videos
sudo systemctl start ms-videos
sudo systemctl status ms-videos
```

## Environment Configuration

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| RABBITMQ_URL | RabbitMQ connection string | `amqp://user:pass@host:5672/` |
| MINIO_ENDPOINT | MinIO/S3 endpoint | `minio.example.com:9000` |
| MINIO_ACCESS_KEY | Access key for storage | `access-key` |
| MINIO_SECRET_KEY | Secret key for storage | `secret-key` |
| MINIO_BUCKET | Bucket name for videos | `videos` |

### Production Environment File

Create `/opt/ms-videos/.env`:

```env
RABBITMQ_URL=amqp://videos:secure-password@rabbitmq.internal:5672/
MINIO_ENDPOINT=minio.internal:9000
MINIO_ACCESS_KEY=prod-access-key
MINIO_SECRET_KEY=prod-secret-key
MINIO_BUCKET=videos
```

## Monitoring and Maintenance

### Health Checks

1. **Service Status:**
```bash
# Docker
...

