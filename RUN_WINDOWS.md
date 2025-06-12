# Executando ms-videos no Windows

Este guia mostra como buildar e executar o microservice ms-videos localmente no Windows para desenvolvimento.

## Pré-requisitos

### 1. Instalar Go

- Baixe e instale Go 1.21+ de: https://golang.org/dl/
- Verifique a instalação:

```powershell
go version
```

### 2. Instalar FFmpeg

#### Opção A: Usando Chocolatey (Recomendado)

```powershell
# Instalar Chocolatey (se não tiver)
Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# Instalar FFmpeg
choco install ffmpeg
```

#### Opção B: Download Manual

1. Baixe FFmpeg de: https://www.gyan.dev/ffmpeg/builds/
2. Extraia para `C:\ffmpeg`
3. Adicione `C:\ffmpeg\bin` ao PATH do sistema
4. Verifique a instalação:

```powershell
ffmpeg -version
```

### 3. Instalar Docker Desktop

- Baixe e instale Docker Desktop de: https://www.docker.com/products/docker-desktop/
- Inicie o Docker Desktop

## Setup do Projeto

### 1. Clonar o Repositório

```powershell
git clone https://github.com/saulotarsobc/ms-videos.git
cd ms-videos
```

### 2. Instalar Dependências Go

```powershell
go mod download
```

### 3. Iniciar Infraestrutura (RabbitMQ + MinIO)

```powershell
# Iniciar apenas os serviços de infraestrutura
docker-compose up rabbitmq minio minio-setup -d

# Verificar se os serviços estão rodando
docker-compose ps
```

## Build e Execução

### Opção 1: Executar com go run (Desenvolvimento)

```powershell
# Executar diretamente
go run cmd/ms-videos/main.go
```

### Opção 2: Build do Executável Windows

#### Build simples:

```powershell
# Build para Windows
go build -o ms-videos.exe cmd/ms-videos/main.go

# Executar o binário
.\ms-videos.exe
```

#### Build com informações de versão:

```powershell
# Build com flags de otimização
go build -ldflags "-s -w" -o ms-videos.exe cmd/ms-videos/main.go

# Executar
.\ms-videos.exe
```

#### Build para diferentes arquiteturas:

```powershell
# Para Windows 64-bit (padrão)
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o ms-videos-win64.exe cmd/ms-videos/main.go

# Para Windows 32-bit
$env:GOOS="windows"; $env:GOARCH="386"; go build -o ms-videos-win32.exe cmd/ms-videos/main.go

# Para Linux (caso queira testar)
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o ms-videos-linux cmd/ms-videos/main.go
```

## Configuração de Ambiente

### Variáveis de Ambiente Padrão

O projeto usa as seguintes variáveis padrão que funcionam com o docker-compose local:

```
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=videos
```

### Configurar Variáveis Customizadas (PowerShell)

Para uma sessão:

```powershell
$env:RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
$env:MINIO_ENDPOINT="localhost:9000"
$env:MINIO_ACCESS_KEY="minioadmin"
$env:MINIO_SECRET_KEY="minioadmin"
$env:MINIO_BUCKET="videos"

# Executar com as variáveis
.\ms-videos.exe
```

Usando arquivo .env (criar na raiz do projeto):

```env
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=videos
```

## Testando o Sistema

### 1. Verificar se os Serviços Estão Funcionando

#### RabbitMQ Management UI:

- URL: http://localhost:15672
- Login: `guest` / `guest`

#### MinIO Console:

- URL: http://localhost:9001
- Login: `minioadmin` / `minioadmin`

### 2. Enviar Vídeo para Processamento

```powershell
# Enviar um vídeo de teste
go run test/send_message.go test-123 "https://sample-videos.com/zip/10/mp4/SampleVideo_1280x720_1mb.mp4" sample.mp4
```

### 3. Monitorar Processamento

Observe os logs do ms-videos rodando. Você deverá ver algo como:

```
2025/06/11 16:09:33 Received video message: ID=test-123, URL=https://...
2025/06/11 16:09:34 Downloading video from URL: https://...
2025/06/11 16:09:35 Processing video test-123 to 1080p resolution
2025/06/11 16:09:40 Processing video test-123 to 720p resolution
2025/06/11 16:09:45 Processing video test-123 to 480p resolution
2025/06/11 16:09:50 Processing video test-123 to 360p resolution
2025/06/11 16:09:55 Uploading HLS files for video test-123
2025/06/11 16:10:00 Successfully processed video test-123
```

### 4. Verificar Resultado no MinIO

1. Acesse http://localhost:9001
2. Navegue para o bucket `videos`
3. Verifique se a pasta `test-123` foi criada com as subpastas:
   - `1080p/`
   - `720p/`
   - `480p/`
   - `360p/`

## Scripts Úteis

### Script de Build (PowerShell)

Crie um arquivo `build.ps1`:

```powershell
# build.ps1
Write-Host "Building ms-videos for Windows..." -ForegroundColor Green

# Limpar builds anteriores
if (Test-Path "ms-videos.exe") {
    Remove-Item "ms-videos.exe"
}

# Build
go build -ldflags "-s -w" -o ms-videos.exe cmd/ms-videos/main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Executable: ms-videos.exe" -ForegroundColor Green
    Write-Host "File size: $((Get-Item ms-videos.exe).Length / 1MB) MB" -ForegroundColor Yellow
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
```

Executar:

```powershell
.\build.ps1
```

### Script de Execução Completa

Crie um arquivo `run-dev.ps1`:

```powershell
# run-dev.ps1
Write-Host "Starting ms-videos development environment..." -ForegroundColor Green

# Verificar se Docker está rodando
try {
    docker version | Out-Null
    Write-Host "Docker is running" -ForegroundColor Green
} catch {
    Write-Host "Docker is not running. Please start Docker Desktop." -ForegroundColor Red
    exit 1
}

# Iniciar infraestrutura
Write-Host "Starting RabbitMQ and MinIO..." -ForegroundColor Yellow
docker-compose up rabbitmq minio minio-setup -d

# Aguardar serviços iniciarem
Write-Host "Waiting for services to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# Build da aplicação
Write-Host "Building application..." -ForegroundColor Yellow
go build -o ms-videos.exe cmd/ms-videos/main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "Starting ms-videos..." -ForegroundColor Green
    Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow
    Write-Host "RabbitMQ UI: http://localhost:15672 (guest/guest)" -ForegroundColor Cyan
    Write-Host "MinIO Console: http://localhost:9001 (minioadmin/minioadmin)" -ForegroundColor Cyan

    # Executar aplicação
    .\ms-videos.exe
} else {
    Write-Host "Build failed!" -ForegroundColor Red
}
```

Executar:

```powershell
.\run-dev.ps1
```

## Troubleshooting

### Problema: FFmpeg não encontrado

```
ffmpeg failed: exec: "ffmpeg": executable file not found in %PATH%
```

**Solução:**

1. Verifique se FFmpeg está instalado: `ffmpeg -version`
2. Se não estiver no PATH, adicione manualmente ou reinstale

### Problema: Erro de conexão com RabbitMQ

```
failed to connect to RabbitMQ: dial tcp 127.0.0.1:5672: connectex: No connection could be made
```

**Solução:**

1. Verifique se o Docker está rodando
2. Execute: `docker-compose up rabbitmq -d`
3. Aguarde alguns segundos e tente novamente

### Problema: Erro de conexão com MinIO

```
failed to upload to MinIO
```

**Solução:**

1. Verifique se MinIO está rodando: `docker-compose ps`
2. Acesse http://localhost:9001 para confirmar
3. Execute: `docker-compose up minio minio-setup -d`

### Problema: Porta em uso

```
Port 5672 is already in use
```

**Solução:**

1. Pare outros serviços RabbitMQ
2. Ou altere a porta no docker-compose.yml

## Comandos Úteis

```powershell
# Ver logs dos containers
docker-compose logs -f rabbitmq
docker-compose logs -f minio

# Parar todos os serviços
docker-compose down

# Limpar volumes (cuidado: apaga dados)
docker-compose down -v

# Rebuild da imagem
docker-compose build ms-videos

# Ver status dos serviços
docker-compose ps

# Conectar no container (debug)
docker-compose exec rabbitmq bash
docker-compose exec minio sh
```

## Performance no Windows

### Dicas para Melhor Performance:

1. **Use SSD** para armazenamento temporário
2. **Configurar exclusões no Windows Defender** para a pasta do projeto
3. **Usar WSL2** se disponível (melhor performance I/O)
4. **Configurar Docker** para usar mais CPU/RAM se necessário

### Configuração Docker Desktop:

- Resources → Advanced
- CPUs: 4+ (se disponível)
- Memory: 4GB+
- Disk image size: 60GB+

Agora você pode fazer o build e executar o ms-videos localmente no Windows para desenvolvimento!
