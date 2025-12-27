# Script to count lines of code in Fluxor project
# Usage: .\statistic.ps1

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  FLUXOR PROJECT CODE STATISTICS" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Count Go source files (excluding tests)
$goSourceFiles = Get-ChildItem -Recurse -Include *.go | Where-Object { 
    $_.FullName -notmatch '\\node_modules\\|\\vendor\\|\\.git\\|\\dist\\|\\coverage\\' -and 
    $_.Name -notmatch '_test\.go$' 
}
$goSourceLines = ($goSourceFiles | Get-Content | Measure-Object -Line).Lines
$goSourceCount = $goSourceFiles.Count

# Count Go test files
$goTestFiles = Get-ChildItem -Recurse -Include *_test.go | Where-Object { 
    $_.FullName -notmatch '\\node_modules\\|\\vendor\\|\\.git\\|\\dist\\|\\coverage\\' 
}
$goTestLines = ($goTestFiles | Get-Content | Measure-Object -Line).Lines
$goTestCount = $goTestFiles.Count

# Count TypeScript/JavaScript files
$tsJsFiles = Get-ChildItem -Recurse -Include *.ts,*.tsx,*.js | Where-Object { 
    $_.FullName -notmatch '\\node_modules\\|\\vendor\\|\\.git\\|\\dist\\|\\coverage\\' 
}
$tsJsLines = ($tsJsFiles | Get-Content | Measure-Object -Line).Lines
$tsJsCount = $tsJsFiles.Count

# Calculate totals
$totalFiles = $goSourceCount + $goTestCount + $tsJsCount
$totalLinesWithoutTests = $goSourceLines + $tsJsLines
$totalLinesWithTests = $goSourceLines + $goTestLines + $tsJsLines

# Display results
Write-Host "=== GO SOURCE CODE ===" -ForegroundColor Green
Write-Host "  Files:        $goSourceCount" -ForegroundColor White
Write-Host "  Lines:        $goSourceLines" -ForegroundColor White
Write-Host ""

Write-Host "=== GO TEST CODE ===" -ForegroundColor Yellow
Write-Host "  Files:        $goTestCount" -ForegroundColor White
Write-Host "  Lines:        $goTestLines" -ForegroundColor White
Write-Host ""

Write-Host "=== TYPESCRIPT/JAVASCRIPT ===" -ForegroundColor Magenta
Write-Host "  Files:        $tsJsCount" -ForegroundColor White
Write-Host "  Lines:        $tsJsLines" -ForegroundColor White
Write-Host ""

Write-Host "=== TOTALS ===" -ForegroundColor Cyan
Write-Host "  Total Files:            $totalFiles" -ForegroundColor White
Write-Host "  Total LOC (no tests):   $totalLinesWithoutTests" -ForegroundColor White
Write-Host "  Total LOC (with tests): $totalLinesWithTests" -ForegroundColor White
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Statistics generated at: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
Write-Host "========================================" -ForegroundColor Cyan

