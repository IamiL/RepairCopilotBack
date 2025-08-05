# Build script for microservices
# Builds api-gateway-service and tz-bot for Linux AMD64

Write-Host "Starting build process..." -ForegroundColor Green

# Set environment variables for cross-compilation
$env:GOOS = "linux"
$env:GOARCH = "amd64"

# Build api-gateway-service
Write-Host "Building api-gateway-service..." -ForegroundColor Yellow
$buildOutput = go build -o api-gateway-app ./api-gateway-service/cmd/main.go 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ api-gateway-service built successfully" -ForegroundColor Green
} else {
    Write-Host "✗ Failed to build api-gateway-service" -ForegroundColor Red
    Write-Host "Build errors:" -ForegroundColor Red
    Write-Host $buildOutput -ForegroundColor Red
    exit 1
}

# Build tz-bot
Write-Host "Building tz-bot..." -ForegroundColor Yellow
$buildOutput = go build -o tz-bot-app ./tz-bot/cmd/main.go 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ tz-bot built successfully" -ForegroundColor Green
} else {
    Write-Host "✗ Failed to build tz-bot" -ForegroundColor Red
    Write-Host "Build errors:" -ForegroundColor Red
    Write-Host $buildOutput -ForegroundColor Red
    exit 1
}

Write-Host "Build process completed successfully!" -ForegroundColor Green
Write-Host "Generated files:" -ForegroundColor Cyan
Write-Host "  - api-gateway-app (Linux AMD64)" -ForegroundColor White
Write-Host "  - tz-bot-app (Linux AMD64)" -ForegroundColor White