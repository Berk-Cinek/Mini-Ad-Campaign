<#
Proves the campaign budget guarantee holds under concurrent HTTP load.

Creates a campaign with budget=100, fires 2000 concurrent POST /impression/{id}
requests at it via `hey` (https://github.com/rakyll/hey), then checks GET /stats/{id}
to confirm exactly 100 impressions were accepted, spent==100, remaining==0, and the
campaign is paused.

The backend's port is NOT published in docker-compose, so:
  - Default -BaseUrl (http://localhost:8080) targets a locally-run backend, e.g.
    `go run ./cmd/server` from backend/.
  - To hit the docker-compose stack instead, go through nginx:
    -BaseUrl http://localhost:3000/api

To prove the guarantee holds across multiple backend instances sharing the same
database (the actual point of this test), scale the backend and re-run through
nginx:
    docker compose up -d --scale backend=2
    ./loadtest/budget-loadtest.ps1 -BaseUrl http://localhost:3000/api

Requires `hey` on PATH (or pass -HeyPath): go install github.com/rakyll/hey@latest

Usage:
    ./loadtest/budget-loadtest.ps1
    ./loadtest/budget-loadtest.ps1 -BaseUrl http://localhost:3000/api
#>

param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$HeyPath = "hey"
)

$ErrorActionPreference = "Stop"
$BaseUrl = $BaseUrl.TrimEnd("/")

if (-not (Get-Command $HeyPath -ErrorAction SilentlyContinue)) {
    Write-Host "ERROR: '$HeyPath' not found on PATH." -ForegroundColor Red
    Write-Host "Install it with: go install github.com/rakyll/hey@latest"
    exit 1
}

# --- Create campaign -------------------------------------------------------

$startDate = [DateTime]::UtcNow
$endDate = $startDate.AddDays(1)
$dateFormat = "yyyy-MM-ddTHH:mm:ssZ"

$createBody = @{
    title      = "loadtest-$($startDate.ToString('yyyyMMddHHmmss'))"
    budget     = 100
    start_date = $startDate.ToString($dateFormat)
    end_date   = $endDate.ToString($dateFormat)
} | ConvertTo-Json

Write-Host "Creating campaign (budget=100)..."
try {
    $campaign = Invoke-RestMethod -Method Post -Uri "$BaseUrl/campaigns" `
        -ContentType "application/json" -Body $createBody
}
catch {
    Write-Host "ERROR: failed to create campaign: $_" -ForegroundColor Red
    exit 1
}

$campaignId = $campaign.id
if (-not $campaignId) {
    Write-Host "ERROR: create response had no 'id' field: $($campaign | ConvertTo-Json)" -ForegroundColor Red
    exit 1
}
Write-Host "Created campaign id=$campaignId"

# --- Fire concurrent impressions via hey ------------------------------------

$impressionUrl = "$BaseUrl/impression/$campaignId"
Write-Host "`nRunning hey: 2000 requests, 200 concurrent, POST $impressionUrl"

$heyOutput = & $HeyPath -n 2000 -c 200 -m POST $impressionUrl 2>&1 | Out-String
Write-Host $heyOutput

$statusCounts = @{}
foreach ($line in ($heyOutput -split "`n")) {
    if ($line -match '\[(\d{3})\]\s+(\d+)\s+responses') {
        $statusCounts[$matches[1]] = [int]$matches[2]
    }
}
$got200 = if ($statusCounts.ContainsKey("200")) { $statusCounts["200"] } else { 0 }
$got409 = if ($statusCounts.ContainsKey("409")) { $statusCounts["409"] } else { 0 }
$got500 = if ($statusCounts.ContainsKey("500")) { $statusCounts["500"] } else { 0 }

Write-Host "hey status code distribution: 200=$got200 (expected 100), 409=$got409 (expected 1900), 500=$got500 (expected 0)"

# --- Check final stats -------------------------------------------------------

Write-Host "`nFetching GET /stats/$campaignId ..."
try {
    $stats = Invoke-RestMethod -Method Get -Uri "$BaseUrl/stats/$campaignId"
}
catch {
    Write-Host "ERROR: failed to fetch stats: $_" -ForegroundColor Red
    exit 1
}

$spentOk = ($stats.spent -eq 100)
$remainingOk = ($stats.remaining -eq 0)
$statusOk = ($stats.status -eq "paused")
$pass = $spentOk -and $remainingOk -and $statusOk

Write-Host ""
Write-Host "spent:     actual=$($stats.spent)     expected=100      $(if ($spentOk) { 'OK' } else { 'MISMATCH' })"
Write-Host "remaining: actual=$($stats.remaining)  expected=0        $(if ($remainingOk) { 'OK' } else { 'MISMATCH' })"
Write-Host "status:    actual=$($stats.status)  expected=paused   $(if ($statusOk) { 'OK' } else { 'MISMATCH' })"
Write-Host ""

if ($pass) {
    Write-Host "PASS" -ForegroundColor Green
    exit 0
}
else {
    Write-Host "FAIL" -ForegroundColor Red
    exit 1
}
