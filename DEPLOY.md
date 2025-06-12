# Guia de Deploy

Este guia fornece instruções detalhadas para fazer deploy do microserviço ms-videos em ambientes de produção.

## Início Rápido

### 1. Docker Compose (Recomendado)

```bash
# Clonar o repositório
git clone https://github.com/saulotarsobc/ms-videos.git
cd ms-videos

# Build e deploy
docker-compose up -d

# Verificar status
docker-compose ps
```

### 2. Build Manual e Deploy

```bash
# Build para Linux
GOOS=linux GOARCH=amd64 go build -o ms-videos-linux cmd/ms-videos/main.go

# Transferir para servidor e executar
scp ms-videos-linux user@server:/opt/ms-videos/
ssh user@server
cd /opt/ms-videos
./ms-videos-linux
```

## Configuração do Ambiente de Produção

### Pré-requisitos

1. **Requisitos do Servidor:**

   - Servidor Linux (Ubuntu 20.04+ ou CentOS 8+)
   - Mínimo 2 núcleos de CPU, 4GB RAM
   - 50GB+ de espaço de armazenamento
   - Docker e Docker Compose (para deploy containerizado)

2. **Serviços Externos:**
   - Servidor RabbitMQ
   - Armazenamento MinIO/S3
   - Acesso de rede às URLs de vídeo

### Configuração da Infraestrutura

#### Opção 1: Docker Compose Tudo-em-Um

1. **Criar docker-compose.yml de produção:**

```yaml
version: "3.8"

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
      - /tmp:/tmp # For temporary files

  rabbitmq:
    image: rabbitmq:3-management
    restart: unless-stopped
    environment:
      - RABBITMQ_DEFAULT_USER=videos
      - RABBITMQ_DEFAULT_PASS=secure-password
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq
    ports:
      - "15672:15672" # Management UI

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

#### Opção 2: Serviços Externos

Se usando RabbitMQ e MinIO/S3 externos:

```yaml
version: "3.8"

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

### Deploy Binário (Sem Docker)

#### 1. Configuração do Servidor

```bash
# Instalar FFmpeg
sudo apt update
sudo apt install -y ffmpeg

# Criar usuário e diretórios
sudo useradd -r -s /bin/false videos
sudo mkdir -p /opt/ms-videos
sudo chown videos:videos /opt/ms-videos
```

#### 2. Build e Deploy

```bash
# Na máquina de desenvolvimento
GOOS=linux GOARCH=amd64 go build -o ms-videos-linux cmd/ms-videos/main.go

# Transferir para servidor
scp ms-videos-linux user@server:/opt/ms-videos/ms-videos
ssh user@server
sudo chown videos:videos /opt/ms-videos/ms-videos
sudo chmod +x /opt/ms-videos/ms-videos
```

#### 3. Serviço Systemd

Criar `/etc/systemd/system/ms-videos.service`:

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

| Variable         | Description                | Example                       |
| ---------------- | -------------------------- | ----------------------------- |
| RABBITMQ_URL     | RabbitMQ connection string | `amqp://user:pass@host:5672/` |
| MINIO_ENDPOINT   | MinIO/S3 endpoint          | `minio.example.com:9000`      |
| MINIO_ACCESS_KEY | Access key for storage     | `access-key`                  |
| MINIO_SECRET_KEY | Secret key for storage     | `secret-key`                  |
| MINIO_BUCKET     | Bucket name for videos     | `videos`                      |

### Production Environment File

Create `/opt/ms-videos/.env`:

```env
RABBITMQ_URL=amqp://videos:secure-password@rabbitmq.internal:5672/
MINIO_ENDPOINT=minio.internal:9000
MINIO_ACCESS_KEY=prod-access-key
MINIO_SECRET_KEY=prod-secret-key
MINIO_BUCKET=videos
```

## Monitoramento e Manutenção

### Verificações de Saúde

1. **Status do Serviço:**

```bash
# Docker
docker-compose ps
docker-compose logs -f ms-videos

# Systemd
sudo systemctl status ms-videos
sudo journalctl -u ms-videos -f
```

2. **Verificação de Componentes:**

```bash
# RabbitMQ (Interface de Gestão: http://your-server:15672)
curl -u guest:guest http://localhost:15672/api/overview

# MinIO (Console: http://your-server:9001)
curl http://localhost:9000/minio/health/live
```

### Logs

1. **Acessar Logs:**

```bash
# Docker
docker-compose logs ms-videos
docker-compose logs --tail=100 -f ms-videos

# Systemd
sudo journalctl -u ms-videos
sudo journalctl -u ms-videos --since "1 hour ago"
```

### Backup

1. **Backup do MinIO:**

```bash
# Usando mc (MinIO Client)
mc mirror minio/videos /backup/videos-$(date +%Y%m%d)
```

2. **Backup de Configurações:**

```bash
# Docker Compose
cp docker-compose.yml /backup/
cp .env /backup/

# Systemd
cp /etc/systemd/system/ms-videos.service /backup/
cp /opt/ms-videos/.env /backup/
```

### Atualizações

1. **Atualização Docker:**

```bash
# Baixar atualizações
git pull origin main

# Rebuild e redeploy
docker-compose build ms-videos
docker-compose up -d ms-videos
```

2. **Atualização Binário:**

```bash
# Build nova versão
GOOS=linux GOARCH=amd64 go build -o ms-videos-linux cmd/ms-videos/main.go

# Parar serviço
sudo systemctl stop ms-videos

# Substituir binário
scp ms-videos-linux user@server:/opt/ms-videos/ms-videos

# Reiniciar serviço
sudo systemctl start ms-videos
sudo systemctl status ms-videos
```

## Solução de Problemas

### Problemas Comuns

1. **FFmpeg não encontrado:**

```bash
# Verificar instalação
which ffmpeg
ffmpeg -version

# Instalar se necessário
sudo apt install -y ffmpeg
```

2. **Erro de conexão RabbitMQ:**

```bash
# Verificar conectividade
telnet rabbitmq-server 5672

# Verificar credenciais
curl -u username:password http://rabbitmq-server:15672/api/overview
```

3. **Erro de conexão MinIO:**

```bash
# Verificar conectividade
curl http://minio-server:9000/minio/health/live

# Testar acesso
mc alias set test http://minio-server:9000 access-key secret-key
mc ls test/
```

4. **Espaço insuficiente:**

```bash
# Verificar espaço em disco
df -h

# Limpar arquivos temporários
sudo find /tmp -name "video_*" -type d -mtime +1 -exec rm -rf {} +
```

### Logs de Debug

Para habilitar logs mais detalhados, adicione estas variáveis de ambiente:

```env
LOG_LEVEL=debug
DEBUG=true
```

## Considerações de Segurança

1. **Credenciais Fortes:**
   - Use senhas complexas para RabbitMQ
   - Gere chaves de acesso seguras para MinIO
   - Rotate credenciais periodicamente

2. **Rede:**
   - Configure firewall adequadamente
   - Use HTTPS para interfaces web
   - Restrinja acesso a portas de gestão

3. **Arquivos:**
   - Configure permissões adequadas
   - Use usuário dedicado sem privilégios
   - Implement backup e retenção de dados

## Performance e Escalabilidade

1. **Recursos por Vídeo:**
   - CPU: 1-2 cores por processamento
   - RAM: 2-4GB por vídeo
   - Disco: 3-5x o tamanho do vídeo original

2. **Escalabilidade Horizontal:**
   - Execute múltiplas instâncias do microserviço
   - Use load balancer para RabbitMQ
   - Configure cluster MinIO para alta disponibilidade

3. **Otimizações:**
   - Use SSDs para armazenamento temporário
   - Configure QoS adequado no RabbitMQ
   - Implemente limpeza automática de arquivos temporários
