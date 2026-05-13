# Demo script for RedisForge (PowerShell)
# One-command local demo flow for LinkedIn/X recording.

param(
    [switch]$Up,
    [switch]$Down,
    [switch]$RunDemo,
    [switch]$OpenMetrics,
    [int]$RedisPort = 6380,
    [int]$ApiPort = 8080
)

$ErrorActionPreference = "Stop"
$RedisContainer = "redis-stack-demo"

function Ensure-RedisStack {
    param([int]$Port)

    $running = docker ps --filter "name=^/$RedisContainer$" --format "{{.Names}}"
    if ($running -contains $RedisContainer) {
        Write-Host "Redis Stack container '$RedisContainer' already running on mapped port $Port."
        return
    }

    $exists = docker ps -a --filter "name=^/$RedisContainer$" --format "{{.Names}}"
    if ($exists -contains $RedisContainer) {
        Write-Host "Starting existing Redis Stack container '$RedisContainer'..."
        docker start $RedisContainer | Out-Null
        return
    }

    Write-Host "Creating Redis Stack container '$RedisContainer' on localhost:$Port ..."
    docker run -d --name $RedisContainer -p "$Port`:6379" redis/redis-stack-server:latest | Out-Null
}

function Stop-RedisStack {
    $exists = docker ps -a --filter "name=^/$RedisContainer$" --format "{{.Names}}"
    if ($exists -contains $RedisContainer) {
        Write-Host "Stopping and removing Redis Stack container '$RedisContainer'..."
        docker rm -f $RedisContainer | Out-Null
    } else {
        Write-Host "Redis Stack container '$RedisContainer' not found."
    }
}

function Wait-ForHealth {
    param([int]$Port)

    $deadline = (Get-Date).AddSeconds(25)
    while ((Get-Date) -lt $deadline) {
        try {
            $null = Invoke-RestMethod -Uri "http://localhost:$Port/healthz" -Method Get
            return $true
        } catch {
            Start-Sleep -Milliseconds 500
        }
    }
    return $false
}

if ($Up) {
    Ensure-RedisStack -Port $RedisPort
    Write-Host "Redis Stack ready."
    exit 0
}

if ($Down) {
    Get-Process -Name redisforge -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Stop-RedisStack
    Write-Host "Demo services stopped."
    exit 0
}

if ($RunDemo) {
    Ensure-RedisStack -Port $RedisPort

    Write-Host "Building RedisForge binary..."
    go build -o bin/redisforge ./cmd/redisforge

    New-Item -ItemType Directory -Path .\logs -Force | Out-Null
    Get-Process -Name redisforge -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

    # Point app to module-enabled Redis Stack.
    $env:REDIS_ADDR = "localhost:$RedisPort"

    Write-Host "Starting RedisForge on port $ApiPort (REDIS_ADDR=$env:REDIS_ADDR)..."
    Start-Process -FilePath .\bin\redisforge -WorkingDirectory (Get-Location) -RedirectStandardOutput .\logs\out.log -RedirectStandardError .\logs\err.log | Out-Null

    if (-not (Wait-ForHealth -Port $ApiPort)) {
        Write-Host "Server did not become healthy. Check logs\err.log"
        exit 1
    }

    Write-Host "Creating demo item via /v1/items ..."
    $payload = '{"name":"Demo PS1 Item","category":"demo","tags":["ps1","demo"],"score":9.8}'
    $created = Invoke-RestMethod -Method Post -Uri "http://localhost:$ApiPort/v1/items" -Body $payload -ContentType "application/json"
    $id = $created.item.id
    Write-Host "Created item id: $id"

    Write-Host "Fetching item via /v1/items/$id ..."
    $item = Invoke-RestMethod -Uri "http://localhost:$ApiPort/v1/items/$id" -Method Get
    Write-Host "Fetched item name: $($item.item.name)"

    Write-Host "Running search via /v1/items/search?q=Demo ..."
    try {
        $search = Invoke-RestMethod -Uri "http://localhost:$ApiPort/v1/items/search?q=Demo" -Method Get
        Write-Host "Search total results: $($search.total)"
    } catch {
        Write-Host "Search step skipped: $($_.Exception.Message)"
    }

    if ($OpenMetrics) {
        Write-Host "Opening metrics page in browser..."
        Start-Process "http://localhost:$ApiPort/metrics"
    }

    Write-Host "Running benchmark quick pass..."
    go test -bench=BenchmarkCacheItemRepo ./internal/repo -run=^$

    Write-Host "Demo complete."
    Write-Host "API base: http://localhost:$ApiPort"
    Write-Host "Metrics:  http://localhost:$ApiPort/metrics"
    Write-Host "Logs:     .\logs\out.log and .\logs\err.log"
    exit 0
}

Write-Host "Demo script usage:"
Write-Host "  -Up                 Start Redis Stack container for demo (default localhost:6380)"
Write-Host "  -Down               Stop demo app and remove Redis Stack demo container"
Write-Host "  -RunDemo            Build app, start server, run create/get/search, run benchmark"
Write-Host "  -OpenMetrics        Open /metrics in browser (optional with -RunDemo)"
Write-Host "  -RedisPort <port>   Redis host port (default 6380)"
Write-Host "  -ApiPort <port>     API port (default 8080)"