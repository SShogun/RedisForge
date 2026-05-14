param(
    [int]$Requests = 250,
    [int]$DurationSeconds = 0,
    [string]$BaseUrl = "http://localhost:8080",
    [int]$WarmupRequests = 10
)

$ErrorActionPreference = "Stop"

function New-StatsBucket {
    return [ordered]@{
        Count     = 0
        Success   = 0
        Failed    = 0
        Latencies = New-Object System.Collections.Generic.List[double]
    }
}

function Add-Result {
    param(
        [System.Collections.IDictionary]$Bucket,
        [bool]$Ok,
        [double]$LatencyMs
    )

    $Bucket.Count++
    if ($Ok) {
        $Bucket.Success++
        $Bucket.Latencies.Add($LatencyMs)
    } else {
        $Bucket.Failed++
    }
}

function Get-Percentile {
    param(
        [double[]]$Values,
        [double]$Percentile
    )

    if ($Values.Count -eq 0) {
        return 0
    }

    $sorted = $Values | Sort-Object
    $index = [math]::Ceiling(($Percentile / 100) * $sorted.Count) - 1
    $index = [math]::Max(0, [math]::Min($index, $sorted.Count - 1))
    return $sorted[$index]
}

function Write-Stats {
    param(
        [string]$Name,
        [System.Collections.IDictionary]$Bucket
    )

    $latencies = [double[]]$Bucket.Latencies.ToArray()
    $avg = 0
    $min = 0
    $max = 0

    if ($latencies.Count -gt 0) {
        $measure = $latencies | Measure-Object -Average -Minimum -Maximum
        $avg = $measure.Average
        $min = $measure.Minimum
        $max = $measure.Maximum
    }

    $p50 = Get-Percentile -Values $latencies -Percentile 50
    $p95 = Get-Percentile -Values $latencies -Percentile 95
    $p99 = Get-Percentile -Values $latencies -Percentile 99

    Write-Host ""
    Write-Host $Name -ForegroundColor Green
    Write-Host ("  requests : {0}" -f $Bucket.Count)
    Write-Host ("  success  : {0}" -f $Bucket.Success)
    Write-Host ("  failed   : {0}" -f $Bucket.Failed)
    Write-Host ("  avg      : {0} ms" -f ([math]::Round($avg, 2)))
    Write-Host ("  min      : {0} ms" -f ([math]::Round($min, 2)))
    Write-Host ("  p50      : {0} ms" -f ([math]::Round($p50, 2)))
    Write-Host ("  p95      : {0} ms" -f ([math]::Round($p95, 2)))
    Write-Host ("  p99      : {0} ms" -f ([math]::Round($p99, 2)))
    Write-Host ("  max      : {0} ms" -f ([math]::Round($max, 2)))
}

function Invoke-TimedRequest {
    param(
        [scriptblock]$Request
    )

    $sw = [Diagnostics.Stopwatch]::StartNew()
    try {
        $response = & $Request
        $sw.Stop()
        return @{
            Ok        = $true
            LatencyMs = $sw.Elapsed.TotalMilliseconds
            Response  = $response
            Error     = $null
        }
    } catch {
        $sw.Stop()
        return @{
            Ok        = $false
            LatencyMs = $sw.Elapsed.TotalMilliseconds
            Response  = $null
            Error     = $_.Exception.Message
        }
    }
}

function New-ItemBody {
    param([int]$Index)

    $categories = @("electronics", "books", "tools", "learning", "infra")
    $tags = @("demo", "bench", "redis", "search", "streams", "json", "bloom")
    $category = $categories[$Index % $categories.Count]
    $tagA = $tags[$Index % $tags.Count]
    $tagB = $tags[($Index + 2) % $tags.Count]

    return @{
        name            = "RedisForge Item $Index"
        category        = $category
        score           = Get-Random -Minimum 1 -Maximum 100
        tags            = @($tagA, $tagB)
        idempotency_key = [guid]::NewGuid().ToString()
    } | ConvertTo-Json
}

$itemsUrl = "$BaseUrl/v1/items"
$healthUrl = "$BaseUrl/healthz"
$metricsUrl = "$BaseUrl/metrics"
$createdIds = New-Object System.Collections.Generic.List[string]

$createStats = New-StatsBucket
$getStats = New-StatsBucket
$searchStats = New-StatsBucket
$duplicateStats = New-StatsBucket
$errors = New-Object System.Collections.Generic.List[string]

Write-Host "==================================================" -ForegroundColor Cyan
Write-Host " RedisForge Benchmark & Traffic Generator" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host ("Base URL : {0}" -f $BaseUrl)
Write-Host ("Requests : {0}" -f $Requests)
if ($DurationSeconds -gt 0) {
    Write-Host ("Duration : {0} seconds" -f $DurationSeconds)
}
Write-Host ""

Write-Host "Checking service health..." -ForegroundColor Yellow
try {
    Invoke-RestMethod -Uri $healthUrl -Method Get | Out-Null
    Write-Host "Service is healthy." -ForegroundColor Green
} catch {
    Write-Host "Could not reach $healthUrl" -ForegroundColor Red
    Write-Host "Start the stack first: .\make.ps1 up"
    exit 1
}

Write-Host "Running warmup..." -ForegroundColor Yellow
for ($i = 1; $i -le $WarmupRequests; $i++) {
    $body = New-ItemBody -Index (-1 * $i)
    try {
        Invoke-RestMethod -Uri $itemsUrl -Method Post -Body $body -ContentType "application/json" | Out-Null
    } catch {
    }
}

Write-Host "Generating benchmark traffic..." -ForegroundColor Yellow
$startedAt = Get-Date
$deadline = $null
if ($DurationSeconds -gt 0) {
    $deadline = $startedAt.AddSeconds($DurationSeconds)
}

$i = 0
while ($true) {
    $i++

    if ($DurationSeconds -le 0 -and $i -gt $Requests) {
        break
    }
    if ($DurationSeconds -gt 0 -and (Get-Date) -ge $deadline) {
        break
    }

    $body = New-ItemBody -Index $i
    $create = Invoke-TimedRequest -Request {
        Invoke-RestMethod -Uri $itemsUrl -Method Post -Body $body -ContentType "application/json"
    }
    Add-Result -Bucket $createStats -Ok $create.Ok -LatencyMs $create.LatencyMs

    if ($create.Ok -and $create.Response.item.id) {
        $createdIds.Add([string]$create.Response.item.id)
    } elseif (-not $create.Ok -and $errors.Count -lt 5) {
        $errors.Add("create: $($create.Error)")
    }

    if ($createdIds.Count -gt 0) {
        $id = $createdIds[(Get-Random -Minimum 0 -Maximum $createdIds.Count)]
        $get = Invoke-TimedRequest -Request {
            Invoke-RestMethod -Uri "$itemsUrl/$id" -Method Get
        }
        Add-Result -Bucket $getStats -Ok $get.Ok -LatencyMs $get.LatencyMs
        if (-not $get.Ok -and $errors.Count -lt 5) {
            $errors.Add("get: $($get.Error)")
        }
    }

    $query = switch ($i % 4) {
        0 { "RedisForge" }
        1 { "@category:{electronics}" }
        2 { "@category:{learning}" }
        default { "@tags:{redis}" }
    }

    $search = Invoke-TimedRequest -Request {
        Invoke-RestMethod -Uri "$itemsUrl/search?q=$([uri]::EscapeDataString($query))" -Method Get
    }
    Add-Result -Bucket $searchStats -Ok $search.Ok -LatencyMs $search.LatencyMs
    if (-not $search.Ok -and $errors.Count -lt 5) {
        $errors.Add("search: $($search.Error)")
    }

    if ($i % 25 -eq 0) {
        $elapsed = ((Get-Date) - $startedAt).TotalSeconds
        Write-Host ("Processed {0} loops in {1}s..." -f $i, [math]::Round($elapsed, 1))
    }
}

Write-Host "Checking duplicate idempotency path..." -ForegroundColor Yellow
$duplicateKey = [guid]::NewGuid().ToString()
$duplicateBody = @{
    name            = "Duplicate Check"
    category        = "infra"
    score           = 42
    tags            = @("bloom", "duplicate")
    idempotency_key = $duplicateKey
} | ConvertTo-Json

$firstDuplicate = Invoke-TimedRequest -Request {
    Invoke-RestMethod -Uri $itemsUrl -Method Post -Body $duplicateBody -ContentType "application/json"
}
Add-Result -Bucket $duplicateStats -Ok $firstDuplicate.Ok -LatencyMs $firstDuplicate.LatencyMs

$secondDuplicate = Invoke-TimedRequest -Request {
    Invoke-RestMethod -Uri $itemsUrl -Method Post -Body $duplicateBody -ContentType "application/json"
}
$duplicateWasRejected = -not $secondDuplicate.Ok
Add-Result -Bucket $duplicateStats -Ok $duplicateWasRejected -LatencyMs $secondDuplicate.LatencyMs

$finishedAt = Get-Date
$elapsedSeconds = [math]::Max(0.001, ($finishedAt - $startedAt).TotalSeconds)
$totalOps = $createStats.Count + $getStats.Count + $searchStats.Count + $duplicateStats.Count
$successfulOps = $createStats.Success + $getStats.Success + $searchStats.Success + $duplicateStats.Success
$failedOps = $createStats.Failed + $getStats.Failed + $searchStats.Failed + $duplicateStats.Failed
$throughput = $totalOps / $elapsedSeconds

Write-Host ""
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host " Benchmark Results" -ForegroundColor Green
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host ("Started       : {0}" -f $startedAt)
Write-Host ("Finished      : {0}" -f $finishedAt)
Write-Host ("Elapsed       : {0}s" -f ([math]::Round($elapsedSeconds, 2)))
Write-Host ("Created items : {0}" -f $createdIds.Count)
Write-Host ("Total ops     : {0}" -f $totalOps)
Write-Host ("Successful    : {0}" -f $successfulOps)
Write-Host ("Failed        : {0}" -f $failedOps)
Write-Host ("Throughput    : {0} ops/sec" -f ([math]::Round($throughput, 2)))

Write-Stats -Name "Create: Bloom + RedisJSON + Streams" -Bucket $createStats
Write-Stats -Name "Get: cache-aside read path" -Bucket $getStats
Write-Stats -Name "Search: RediSearch queries" -Bucket $searchStats
Write-Stats -Name "Duplicate: Bloom idempotency path" -Bucket $duplicateStats

if ($errors.Count -gt 0) {
    Write-Host ""
    Write-Host "Sample errors" -ForegroundColor Yellow
    foreach ($err in $errors) {
        Write-Host "  $err"
    }
}

Write-Host ""
Write-Host "What this populated:" -ForegroundColor Yellow
Write-Host "  - create path latency and audit stream events"
Write-Host "  - cache-aside get traffic"
Write-Host "  - RediSearch category/tag/text queries"
Write-Host "  - Bloom duplicate/idempotency behavior"
Write-Host ""
Write-Host "Open Grafana: http://localhost:3000" -ForegroundColor Green
Write-Host "Login: admin / admin"
Write-Host "Raw metrics: $metricsUrl"
