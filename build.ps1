# Build script for microservices
# Builds api-gateway-service and tz-bot for Linux AMD64

Write-Host "Starting build process..." -ForegroundColor Green

# Set environment variables for cross-compilation
$env:GOOS = "linux"
$env:GOARCH = "amd64"

# Build api-gateway-service
Write-Host "Building api-gateway-service..." -ForegroundColor Yellow
try {
    go build -o api-gateway-app ./api-gateway-service/cmd/main.go
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ api-gateway-service built successfully" -ForegroundColor Green
    } else {
        Write-Host "✗ Failed to build api-gateway-service" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "✗ Error building api-gateway-service: $_" -ForegroundColor Red
    exit 1
}

# Build tz-bot
Write-Host "Building tz-bot..." -ForegroundColor Yellow
try {
    go build -o tz-bot-app ./tz-bot/cmd/main.go
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ tz-bot built successfully" -ForegroundColor Green
    } else {
        Write-Host "✗ Failed to build tz-bot" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "✗ Error building tz-bot: $_" -ForegroundColor Red
    exit 1
}

Write-Host "Build process completed successfully!" -ForegroundColor Green
Write-Host "Generated files:" -ForegroundColor Cyan
Write-Host "  - api-gateway-app (Linux AMD64)" -ForegroundColor White
Write-Host "  - tz-bot-app (Linux AMD64)" -ForegroundColor White