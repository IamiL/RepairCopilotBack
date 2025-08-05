# Build script for microservices
# Builds api-gateway-service and tz-bot for Linux AMD64 with versioning

Write-Host "Starting build process..." -ForegroundColor Green

# Clean up old binary files
Write-Host "Cleaning up old binary files..." -ForegroundColor Yellow
Get-ChildItem -Name "api-gateway-app-v*" | Remove-Item -Force -ErrorAction SilentlyContinue
Get-ChildItem -Name "tz-bot-app-v*" | Remove-Item -Force -ErrorAction SilentlyContinue
Write-Host "Old binary files cleaned up" -ForegroundColor Green

# Set environment variables for cross-compilation
$env:GOOS = "linux"
$env:GOARCH = "amd64"

# Generate version based on current timestamp
$version = Get-Date -Format "yyyyMMdd-HHmmss"
Write-Host "Build version: $version" -ForegroundColor Cyan

# Build api-gateway-service
Write-Host "Building api-gateway-service..." -ForegroundColor Yellow
$apiGatewayBinary = "api-gateway-app-v$version"
$buildOutput = go build -o $apiGatewayBinary ./api-gateway-service/cmd/main.go 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "api-gateway-service built successfully" -ForegroundColor Green
} else {
    Write-Host "Failed to build api-gateway-service" -ForegroundColor Red
    Write-Host "Build errors:" -ForegroundColor Red
    Write-Host $buildOutput -ForegroundColor Red
    exit 1
}

# Build tz-bot
Write-Host "Building tz-bot..." -ForegroundColor Yellow
$tzBotBinary = "tz-bot-app-v$version"
$buildOutput = go build -o $tzBotBinary ./tz-bot/cmd/main.go 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "tz-bot built successfully" -ForegroundColor Green
} else {
    Write-Host "Failed to build tz-bot" -ForegroundColor Red
    Write-Host "Build errors:" -ForegroundColor Red
    Write-Host $buildOutput -ForegroundColor Red
    exit 1
}

Write-Host "Build process completed successfully!" -ForegroundColor Green
Write-Host "Generated files:" -ForegroundColor Cyan
Write-Host "  - $apiGatewayBinary (Linux AMD64)" -ForegroundColor White
Write-Host "  - $tzBotBinary (Linux AMD64)" -ForegroundColor White

# Git operations
Write-Host "Adding files to git..." -ForegroundColor Yellow
git add .
if ($LASTEXITCODE -eq 0) {
    Write-Host "Files added to git successfully" -ForegroundColor Green
} else {
    Write-Host "Failed to add files to git" -ForegroundColor Red
    exit 1
}

Write-Host "Committing changes..." -ForegroundColor Yellow
git commit -m "update"
if ($LASTEXITCODE -eq 0) {
    Write-Host "Changes committed successfully" -ForegroundColor Green
} else {
    Write-Host "Failed to commit changes" -ForegroundColor Red
    exit 1
}

Write-Host "Pushing to remote repository..." -ForegroundColor Yellow
git push
if ($LASTEXITCODE -eq 0) {
    Write-Host "Changes pushed successfully" -ForegroundColor Green
} else {
    Write-Host "Failed to push changes" -ForegroundColor Red
    exit 1
}

Write-Host "All operations completed successfully!" -ForegroundColor Green