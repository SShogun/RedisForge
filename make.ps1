param(
    [Parameter(Position=0)]
    [string]$Target = "run"
)

switch ($Target) {
    "run" {
        Write-Host "Running redisforge..." -ForegroundColor Cyan
        go run ./cmd/redisforge
    }
    "build" {
        Write-Host "Building redisforge to bin/redisforge.exe..." -ForegroundColor Cyan
        if (!(Test-Path -Path "bin")) { New-Item -ItemType Directory -Path "bin" | Out-Null }
        go build -o bin/redisforge.exe ./cmd/redisforge
    }
    "test" {
        Write-Host "Running tests with race detection..." -ForegroundColor Cyan
        go test -race ./...
    }
    "lint" {
        Write-Host "Running golangci-lint..." -ForegroundColor Cyan
        golangci-lint run ./...
    }
    "tidy" {
        Write-Host "Running go mod tidy..." -ForegroundColor Cyan
        go mod tidy
    }
    "up" {
        Write-Host "Starting docker-compose stack..." -ForegroundColor Cyan
        docker-compose -f deployments/docker-compose.yml up --build -d
    }
    "down" {
        Write-Host "Stopping docker-compose stack..." -ForegroundColor Cyan
        docker-compose -f deployments/docker-compose.yml down
    }
    default {
        Write-Host "Unknown target: $Target" -ForegroundColor Red
        Write-Host "Available targets: run, build, test, lint, tidy, up, down"
    }
}
