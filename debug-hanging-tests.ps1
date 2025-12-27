# Debug script to identify hanging tests
# Run this if CI tests are timing out

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Debugging Hanging Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Test 1: Run tests with timeout and identify which one hangs
Write-Host "[1/4] Running tests with individual timeouts..." -ForegroundColor Yellow
$testPackages = go list ./... | Where-Object { $_ -like "*test*" -or (Test-Path "$_/*_test.go") }
$testPackages = go list ./...

$failed = @()
$passed = @()

foreach ($pkg in $testPackages) {
    Write-Host "Testing: $pkg" -ForegroundColor Gray
    $job = Start-Job -ScriptBlock {
        param($pkg)
        $env:GOFLAGS = "-timeout=2m"
        go test -v -timeout=2m -race $pkg 2>&1
    } -ArgumentList $pkg
    
    $result = Wait-Job $job -Timeout 150
    if ($result) {
        $output = Receive-Job $job
        Remove-Job $job
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  ✓ $pkg" -ForegroundColor Green
            $passed += $pkg
        } else {
            Write-Host "  ✗ $pkg" -ForegroundColor Red
            $failed += $pkg
            Write-Host $output -ForegroundColor Red
        }
    } else {
        Write-Host "  ⚠ $pkg - TIMED OUT (likely hanging)" -ForegroundColor Yellow
        $failed += $pkg
        Stop-Job $job
        Remove-Job $job
    }
}

Write-Host ""
Write-Host "[2/4] Summary:" -ForegroundColor Yellow
Write-Host "  Passed: $($passed.Count)" -ForegroundColor Green
Write-Host "  Failed/Timeout: $($failed.Count)" -ForegroundColor Red

if ($failed.Count -gt 0) {
    Write-Host ""
    Write-Host "[3/4] Failed/Timeout packages:" -ForegroundColor Yellow
    $failed | ForEach-Object { Write-Host "  - $_" -ForegroundColor Red }
    
    Write-Host ""
    Write-Host "[4/4] Running detailed test on first failed package..." -ForegroundColor Yellow
    if ($failed.Count -gt 0) {
        $firstFailed = $failed[0]
        Write-Host "Testing: $firstFailed" -ForegroundColor Cyan
        go test -v -race -timeout=1m $firstFailed
    }
} else {
    Write-Host ""
    Write-Host "[3/4] All packages passed!" -ForegroundColor Green
    Write-Host "[4/4] Running full test suite..." -ForegroundColor Yellow
    go test -v -race -timeout=10m ./...
}

