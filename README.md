# ms-videos

Um microserviço orientado por eventos que processa vídeos convertendo-os para múltiplas resoluções e formato HLS.

## Funcionalidades

- Escuta fila RabbitMQ para requisições de processamento de vídeo
- Baixa vídeos de URLs públicas
- Converte vídeos para resoluções 1080p, 720p, 480p e 360p
- Fragmenta vídeos usando formato HLS (.m3u8 + segmentos .ts)
- Faz upload dos arquivos processados para armazenamento MinIO/S3
- Tratamento de desligamento gracioso
- Processa um vídeo por vez (sem processamento paralelo)

## Formato da Mensagem

```json
{
  "id": "uuid-string",
  "url": "https://example.com/video.mp4",
  "filename": "video.mp4"
}
```

## Variáveis de Ambiente

- `RABBITMQ_URL`: String de conexão RabbitMQ (padrão: `amqp://guest:guest@localhost:5672/`)
- `MINIO_ENDPOINT`: Endpoint MinIO (padrão: `localhost:9000`)
- `MINIO_ACCESS_KEY`: Chave de acesso MinIO (padrão: `minioadmin`)
- `MINIO_SECRET_KEY`: Chave secreta MinIO (padrão: `minioadmin`)
- `MINIO_BUCKET`: Nome do bucket MinIO (padrão: `videos`)

## Pré-requisitos

- Go 1.21+
- Docker e Docker Compose
- FFmpeg (para desenvolvimento local)

## Desenvolvimento

1. Clone o repositório
2. Copie o arquivo `.env` e ajuste conforme necessário
3. Inicie a infraestrutura:

```bash
docker-compose up rabbitmq minio minio-setup -d
```

4. Execute o serviço localmente:

```bash
go run cmd/ms-videos/main.go
```

## Build & Deploy

### Build Docker

1. **Construir a imagem Docker:**

```bash
docker build -t ms-videos:latest .
```

2. **Marcar para registry (opcional):**

```bash
docker tag ms-videos:latest your-registry.com/ms-videos:latest
```

3. **Enviar para registry (opcional):**

```bash
docker push your-registry.com/ms-videos:latest
```

### Build Binário Local

1. **Build para plataforma atual:**

```bash
go build -o ms-videos.exe cmd/ms-videos/main.go
```

2. **Build para Linux (para deploy no servidor):**

```bash
GOOS=linux GOARCH=amd64 go build -o ms-videos-linux cmd/ms-videos/main.go
```

### Deploy em Produção

#### Usando Docker Compose (Recomendado)

1. **Deploy de todos os serviços:**

```bash
docker-compose up -d
```

2. **Verificar status dos serviços:**

```bash
docker-compose ps
```

3. **Ver logs:**

```bash
docker-compose logs -f ms-videos
```

#### Deploy Manual

1. **Garantir que FFmpeg está instalado no servidor:**

```bash
# Ubuntu/Debian
sudo apt update && sudo apt install -y ffmpeg

# CentOS/RHEL
sudo yum install -y epel-release
sudo yum install -y ffmpeg
```

2. **Definir variáveis de ambiente de produção:**

```bash
export RABBITMQ_URL="amqp://user:password@your-rabbitmq-server:5672/"
export MINIO_ENDPOINT="your-minio-server:9000"
export MINIO_ACCESS_KEY="your-access-key"
export MINIO_SECRET_KEY="your-secret-key"
export MINIO_BUCKET="videos"
```

3. **Executar o binário:**

```bash
./ms-videos-linux
```

#### Usando systemd (Linux)

1. **Criar arquivo de serviço systemd:**

```bash
sudo nano /etc/systemd/system/ms-videos.service
```

2. **Configuração do serviço:**

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

3. **Habilitar e iniciar serviço:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable ms-videos
sudo systemctl start ms-videos
sudo systemctl status ms-videos
```

### Considerações de Produção

- **Requisitos de Recursos:** Garantir CPU, memória e espaço em disco adequados para processamento de vídeo
- **FFmpeg:** Deve estar disponível no PATH do sistema
- **Rede:** Conexão à internet estável para baixar vídeos de origem
- **Armazenamento:** Configurar MinIO/S3 com políticas adequadas de backup e retenção
- **Monitoramento:** Implementar logging e monitoramento para o serviço
- **Segurança:** Usar credenciais fortes e configurações de rede seguras
- **Escalabilidade:** Considerar escalabilidade horizontal para processamento de alto volume

### Configuração de Ambiente para Produção

Criar arquivo `.env` para produção:

```env
RABBITMQ_URL=amqp://production-user:strong-password@rabbitmq.example.com:5672/
MINIO_ENDPOINT=minio.example.com:9000
MINIO_ACCESS_KEY=production-access-key
MINIO_SECRET_KEY=production-secret-key
MINIO_BUCKET=videos
```

## Serviços

- **ms-videos**: Microserviço principal
- **RabbitMQ**: Fila de mensagens (Interface de Gestão: http://localhost:15672)
- **MinIO**: Armazenamento de objetos (Console: http://localhost:9001)

## Testes

### Executando a Aplicação

1. **Iniciar a infraestrutura:**

```bash
docker-compose up rabbitmq minio minio-setup -d
```

2. **Executar o microserviço:**

```bash
go run cmd/ms-videos/main.go
```

3. **Enviar um vídeo de teste para processamento:**

```bash
go run test/send_message.go test-123 "https://sample-videos.com/zip/10/mp4/SampleVideo_1280x720_1mb.mp4" test-123.mp4;
# http://localhost:9000/videos/test-123/master.m3u8
go run test/send_message.go jw-01 "https://akamd1.jw-cdn.org/sg2/p/640e48e/2/o/jwbvod25_T_20_r720P.mp4" jw-01.mp4;
# http://localhost:9000/videos/jw-01/master.m3u8
```

> Teste em [livepush.io](https://livepush.io/hlsplayer/index.html)

### Uso do Script de Teste

O script `test/send_message.go` permite enviar requisições de processamento de vídeo para a fila RabbitMQ:

```bash
go run test/send_message.go <video-id> <video-url> <filename>
```

**Parâmetros:**

- `video-id`: Identificador único para o vídeo (será usado como nome da pasta no MinIO)
- `video-url`: URL pública para baixar o arquivo de vídeo
- `filename`: Nome original do arquivo de vídeo

**Exemplos:**

```bash
# Processar um vídeo de exemplo
go run test/send_message.go video-001 "https://www.learningcontainer.com/wp-content/uploads/2020/05/sample-mp4-file.mp4" sample.mp4
```

### Monitoramento

Após enviar uma mensagem, você pode monitorar o processamento:

1. **Acompanhar os logs do microserviço** para atualizações de processamento
2. **Interface de Gestão RabbitMQ** (http://localhost:15672) - Login: `guest/guest`
   - Ver status da fila e consumo de mensagens
3. **Console MinIO** (http://localhost:9001) - Login: `minioadmin/minioadmin`
   - Verificar arquivos de vídeo processados no bucket `videos`

### Saída Esperada

Quando o processamento for bem-sucedido, você verá logs como:

```
2025/06/11 15:45:33 Received video message: ID=test-123, URL=https://...
2025/06/11 15:45:34 Downloading video from URL: https://...
2025/06/11 15:45:35 Processing video test-123 to 1080p resolution
2025/06/11 15:45:40 Processing video test-123 to 720p resolution
2025/06/11 15:45:45 Processing video test-123 to 480p resolution
2025/06/11 15:45:50 Processing video test-123 to 360p resolution
2025/06/11 15:45:55 Creating master playlist for video test-123
2025/06/11 15:45:56 Uploading HLS files for video test-123
2025/06/11 15:46:00 Successfully processed video test-123
```

## Estrutura de Saída

Vídeos processados são armazenados no MinIO com a seguinte estrutura:

```
videos/
├── {video-id}/
│   ├── master.m3u8          # Master playlist (adaptive bitrate)
│   ├── 1080p/
│   │   ├── playlist.m3u8    # 1080p stream playlist
│   │   ├── segment_000.ts
│   │   ├── segment_001.ts
│   │   └── ...
│   ├── 720p/
│   │   ├── playlist.m3u8    # 720p stream playlist
│   │   ├── segment_000.ts
│   │   ├── segment_001.ts
│   │   └── ...
│   ├── 480p/
│   │   ├── playlist.m3u8    # 480p stream playlist
│   │   └── segments...
│   └── 360p/
│       ├── playlist.m3u8    # 360p stream playlist
│       └── segments...
```

## Adaptive Bitrate Streaming

O sistema gera um **master playlist** (`master.m3u8`) que permite streaming adaptativo. Este arquivo contém informações sobre todas as resoluções disponíveis e suas respectivas larguras de banda:

```m3u8
#EXTM3U
#EXT-X-VERSION:3

#EXT-X-STREAM-INF:BANDWIDTH=5000000,RESOLUTION=1920x1080
1080p/playlist.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=3000000,RESOLUTION=1280x720
720p/playlist.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=1500000,RESOLUTION=854x480
480p/playlist.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=800000,RESOLUTION=640x360
360p/playlist.m3u8
```

### Como usar:

1. **Para streaming adaptativo**: Use o arquivo `master.m3u8`

   - URL: `https://your-minio-server/videos/{video-id}/master.m3u8`
   - O player automaticamente escolhe a melhor resolução baseada na conexão

2. **Para resolução específica**: Use o playlist da resolução desejada
   - URL: `https://your-minio-server/videos/{video-id}/720p/playlist.m3u8`

### Larguras de Banda:

- **1080p**: 5 Mbps (1920x1080)
- **720p**: 3 Mbps (1280x720)
- **480p**: 1.5 Mbps (854x480)
- **360p**: 800 Kbps (640x360)
