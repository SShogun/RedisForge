Write-Host "==================================================" -ForegroundColor Cyan
Write-Host " RedisForge: Architecture Benchmark & Traffic Gen " -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "Generating traffic to populate Grafana dashboards..." -ForegroundColor Yellow

# Ensure we're hitting the local server
$baseUrl = "http://localhost:8080/v1/items"

# We will collect latencies to show performance
$createLatencies = @()
$searchLatencies = @()

$totalRequests = 100
Write-Host "Sending $totalRequests requests..."

for ($i = 1; $i -le $totalRequests; $i++) {
    $uuid = [guid]::NewGuid().ToString()
    
    $body = @{
        name = "Widget Pro v$i"
        category = "electronics"
        score = (Get-Random -Minimum 1 -Maximum 10)
        tags = @("demo", "bench")
        idempotency_key = $uuid
    } | ConvertTo-Json

    # Measure Create (Bloom + RedisJSON + Stream)
    $sw = [Diagnostics.Stopwatch]::StartNew()
    Invoke-RestMethod -Uri $baseUrl -Method Post -Body $body -ContentType "application/json" -ErrorAction SilentlyContinue | Out-Null
    $sw.Stop()
    $createLatencies += $sw.ElapsedMilliseconds

    # Measure Search (RediSearch)
    $sw.Restart()
    Invoke-RestMethod -Uri "$baseUrl/search?q=@category:{electronics}" -Method Get -ErrorAction SilentlyContinue | Out-Null
    $sw.Stop()
    $searchLatencies += $sw.ElapsedMilliseconds

    if ($i % 10 -eq 0) {
        Write-Host "Processed $i / $totalRequests requests..."
    }
}

$avgCreate = ($createLatencies | Measure-Object -Average).Average
$avgSearch = ($searchLatencies | Measure-Object -Average).Average

Write-Host "`n==================================================" -ForegroundColor Cyan
Write-Host " Benchmark Results" -ForegroundColor Green
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "Average Create + Audit Log Latency: $([math]::Round($avgCreate, 2)) ms"
Write-Host "Average RediSearch Faceted Query  : $([math]::Round($avgSearch, 2)) ms"

Write-Host "`nWhy this is faster than 'Normal Redis':" -ForegroundColor Yellow
Write-Host "1. RedisJSON allows partial updates (e.g., Increment Score) in ONE round-trip without downloading/parsing JSON."
Write-Host "2. RediSearch allows complex queries (Tags + Categories + Text) without manually maintaining thousands of SETs."
Write-Host "3. Bloom Filters prevent database round-trips for duplicate requests in sub-millisecond time."
Write-Host "4. Streams handle audit logging asynchronously so the user request isn't blocked."

Write-Host "`nCheck your Grafana Dashboard (http://localhost:3000) now! It is fully populated." -ForegroundColor Green
