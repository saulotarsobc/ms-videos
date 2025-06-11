# run-dev.ps1
Write-Host "Starting ms-videos development environment..." -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Green

# Verificar se Docker está rodando
Write-Host "Checking Docker..." -ForegroundColor Yellow
try {
    docker version | Out-Null
    Write-Host "✓ Docker is running" -ForegroundColor Green
} catch {
    Write-Host "✗ Docker is not running. Please start Docker Desktop." -ForegroundColor Red
    Write-Host "  Download: https://www.docker.com/products/docker-desktop/" -ForegroundColor Cyan
    exit 1
}

# Verificar se Go está instalado
Write-Host "Checking Go..." -ForegroundColor Yellow
try {
    $goVersion = go version
    Write-Host "✓ $goVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Go is not installed or not in PATH" -ForegroundColor Red
    Write-Host "  Download: https://golang.org/dl/" -ForegroundColor Cyan
    exit 1
}

# Verificar se FFmpeg está instalado
Write-Host "Checking FFmpeg..." -ForegroundColor Yellow
try {
    ffmpeg -version 2>$null | Out-Null
    Write-Host "✓ FFmpeg is available" -ForegroundColor Green
} catch {
    Write-Host "✗ FFmpeg is not installed or not in PATH" -ForegroundColor Red
    Write-Host "  Install with: choco install ffmpeg" -ForegroundColor Cyan
    Write-Host "  Or download from: https://www.gyan.dev/ffmpeg/builds/" -ForegroundColor Cyan
    exit 1
}

Write-Host ""
Write-Host "Starting infrastructure services..." -ForegroundColor Yellow

# Parar serviços existentes (se houver)
docker-compose down 2>$null | Out-Null

# Iniciar infraestrutura
docker-compose up rabbitmq minio minio-setup -d

if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Failed to start infrastructure services" -ForegroundColor Red
    exit 1
}

# Aguardar serviços iniciarem
Write-Host "Waiting for services to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 15

# Verificar status dos serviços
Write-Host "Checking services status..." -ForegroundColor Yellow
$services = docker-compose ps --services --filter "status=running"
if ($services -contains "rabbitmq" -and $services -contains "minio") {
    Write-Host "✓ All services are running" -ForegroundColor Green
} else {
    Write-Host "✗ Some services failed to start" -ForegroundColor Red
    docker-compose ps
    exit 1
}

Write-Host ""
Write-Host "Building application..." -ForegroundColor Yellow
go build -o ms-videos.exe cmd/ms-videos/main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Build successful" -ForegroundColor Green
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "   ms-videos Development Environment   " -ForegroundColor Green  
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Services URLs:" -ForegroundColor Cyan
    Write-Host "  RabbitMQ UI: http://localhost:15672 (guest/guest)" -ForegroundColor White
    Write-Host "  MinIO Console: http://localhost:9001 (minioadmin/minioadmin)" -ForegroundColor White
    Write-Host ""
    Write-Host "Test video processing:" -ForegroundColor Cyan
    Write-Host '  go run test/send_message.go test-123 "https://sample-videos.com/zip/10/mp4/SampleVideo_1280x720_1mb.mp4" sample.mp4' -ForegroundColor White
    Write-Host ""
    Write-Host "Press Ctrl+C to stop the application" -ForegroundColor Yellow
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    
    # Executar aplicação
    .\ms-videos.exe
} else {
    Write-Host "✗ Build failed!" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Shutting down services..." -ForegroundColor Yellow
docker-compose down
Write-Host "Development environment stopped." -ForegroundColor Green

