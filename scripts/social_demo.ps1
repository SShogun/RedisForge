# Social demo script for short recording (LinkedIn/X)
# Focused flow: create -> get -> open metrics.

param(
    [int]$ApiPort = 8080,
    [switch]$OpenMetrics
)

$ErrorActionPreference = "Stop"

function Require-HealthyApi {
    param([int]$Port)

    try {
        $null = Invoke-RestMethod -Uri "http://localhost:$Port/healthz" -Method Get
    } catch {
        throw "API is not healthy on http://localhost:$Port. Start it first via: .\scripts\demo.ps1 -RunDemo"
    }
}

Require-HealthyApi -Port $ApiPort

New-Item -ItemType Directory -Path .\logs -Force | Out-Null
$transcriptPath = ".\logs\social_demo_$(Get-Date -Format 'yyyyMMdd_HHmmss').log"
Start-Transcript -Path $transcriptPath | Out-Null

Write-Host "Step 1/3: Create item"
$payload = '{"name":"Social Demo Item","category":"demo","tags":["social","x","linkedin"],"score":9.7}'
$created = Invoke-RestMethod -Method Post -Uri "http://localhost:$ApiPort/v1/items" -Body $payload -ContentType "application/json"
$id = $created.item.id
$created | ConvertTo-Json -Depth 6

Write-Host "Step 2/3: Get item (cache demonstration)"
$fetched = Invoke-RestMethod -Method Get -Uri "http://localhost:$ApiPort/v1/items/$id"
$fetched | ConvertTo-Json -Depth 6

Write-Host "Step 3/3: Open metrics"
if ($OpenMetrics) {
    Start-Process "http://localhost:$ApiPort/metrics"
}
Write-Host "Metrics URL: http://localhost:$ApiPort/metrics"
Write-Host "Recording log: $transcriptPath"

Stop-Transcript | Out-Null
