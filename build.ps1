# build.ps1
Write-Host "Building ms-videos for Windows..." -ForegroundColor Green

# Limpar builds anteriores
if (Test-Path "ms-videos.exe") {
    Remove-Item "ms-videos.exe"
    Write-Host "Removed previous build" -ForegroundColor Yellow
}

# Build
Write-Host "Compiling..." -ForegroundColor Yellow
go build -ldflags "-s -w" -o ms-videos.exe cmd/ms-videos/main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Executable: ms-videos.exe" -ForegroundColor Green
    $fileSize = (Get-Item ms-videos.exe).Length / 1MB
    Write-Host "File size: $([math]::Round($fileSize, 2)) MB" -ForegroundColor Yellow
    
    Write-Host "" 
    Write-Host "To run the application:" -ForegroundColor Cyan
    Write-Host "  .\ms-videos.exe" -ForegroundColor White
    Write-Host ""
    Write-Host "Make sure RabbitMQ and MinIO are running:" -ForegroundColor Cyan  
    Write-Host "  docker-compose up rabbitmq minio minio-setup -d" -ForegroundColor White
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

