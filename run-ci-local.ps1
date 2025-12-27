# Local CI Runner - Simulates GitHub Actions CI workflow
# This script runs the same steps as .github/workflows/ci.yml

param(
    [string]$Job = "all",
    [switch]$SkipTests = $false,
    [switch]$SkipBuild = $false,
    [switch]$SkipBenchmark = $false,
    [switch]$SkipSecurity = $false,
    [switch]$SkipExamples = $false,
    [switch]$SkipDocker = $false
)

$ErrorActionPreference = "Stop"
$GO_VERSION = "1.24"

function Write-Step {
    param([string]$Message)
    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host $Message -ForegroundColor Cyan
    Write-Host "========================================`n" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-InfoMsg {
    param([string]$Message)
    Write-Host "ℹ $Message" -ForegroundColor Yellow
}

# Check Go installation
Write-Step "Checking Go installation"
$goVersion = go version 2>&1
if (-not $?) {
    Write-Error "Go is not installed or not in PATH"
    exit 1
}
Write-Success "Go found: $goVersion"

# Test Job
if ($Job -eq "all" -or $Job -eq "test") {
    if (-not $SkipTests) {
        Write-Step "Running Test Job"
        
        Write-InfoMsg "Downloading dependencies..."
        go mod download
        if (-not $?) {
            Write-Error "Failed to download dependencies"
            exit 1
        }
        Write-Success "Dependencies downloaded"
        
        Write-InfoMsg "Verifying dependencies..."
        go mod verify
        if (-not $?) {
            Write-Error "Dependency verification failed"
            exit 1
        }
        Write-Success "Dependencies verified"
        
        Write-InfoMsg "Running go vet..."
        go vet ./...
        if (-not $?) {
            Write-Error "go vet found issues"
            exit 1
        }
        Write-Success "go vet passed"
        
        Write-InfoMsg "Checking go fmt..."
        $unformatted = gofmt -s -l .
        if ($unformatted) {
            Write-Error "The following files are not formatted:"
            Write-Host $unformatted
            exit 1
        }
        Write-Success "All files are properly formatted"
        
        Write-InfoMsg "Running tests with race detector..."
        $testTimeout = "10m"
        go test -v -race -coverprofile=coverage.out -covermode=atomic -timeout $testTimeout ./...
        if (-not $?) {
            Write-Error "Tests failed"
            Write-InfoMsg "Listing test files for debugging..."
            Get-ChildItem -Recurse -Filter "*_test.go" | Select-Object -First 20 FullName
            exit 1
        }
        Write-Success "All tests passed"
        
        if (Test-Path "coverage.out") {
            Write-InfoMsg "Coverage report generated: coverage.out"
            $coverage = go tool cover -func=coverage.out | Select-String "total:"
            Write-Host $coverage
        }
    }
}

# Build Job
if ($Job -eq "all" -or $Job -eq "build") {
    if (-not $SkipBuild) {
        Write-Step "Running Build Job"
        
        # Create bin directory
        if (-not (Test-Path "bin")) {
            New-Item -ItemType Directory -Path "bin" | Out-Null
        }
        
        Write-InfoMsg "Building main binary..."
        go build -v -o bin/fluxor.exe ./cmd/main.go
        if (-not $?) {
            Write-Error "Failed to build main binary"
            exit 1
        }
        Write-Success "Main binary built: bin/fluxor.exe"
        
        Write-InfoMsg "Building example binary..."
        go build -v -o bin/fluxor-example.exe ./cmd/example/main.go
        if (-not $?) {
            Write-Error "Failed to build example binary"
            exit 1
        }
        Write-Success "Example binary built: bin/fluxor-example.exe"
        
        Write-InfoMsg "Building enterprise binary..."
        go build -v -o bin/fluxor-enterprise.exe ./cmd/enterprise/main.go
        if (-not $?) {
            Write-Error "Failed to build enterprise binary"
            exit 1
        }
        Write-Success "Enterprise binary built: bin/fluxor-enterprise.exe"
        
        Write-InfoMsg "Build artifacts:"
        Get-ChildItem bin/*.exe | ForEach-Object {
            Write-Host "  - $($_.Name) ($([math]::Round($_.Length / 1KB, 2)) KB)"
        }
    }
}

# Benchmark Job
if ($Job -eq "all" -or $Job -eq "benchmark") {
    if (-not $SkipBenchmark) {
        Write-Step "Running Benchmark Job"
        
        Write-InfoMsg "Running benchmarks..."
        go test -bench=. -benchmem -run=^$ ./... | Tee-Object -FilePath benchmark.txt
        if (-not $?) {
            Write-Error "Benchmarks failed"
            exit 1
        }
        Write-Success "Benchmarks completed. Results saved to benchmark.txt"
    }
}

# Security Job
if ($Job -eq "all" -or $Job -eq "security") {
    if (-not $SkipSecurity) {
        Write-Step "Running Security Scan Job"
        
        # Check if gosec is installed
        $gosecInstalled = Get-Command gosec -ErrorAction SilentlyContinue
        if (-not $gosecInstalled) {
            Write-InfoMsg "gosec not found. Installing..."
            go install github.com/securego/gosec/v2/cmd/gosec@latest
            if (-not $?) {
                Write-Error "Failed to install gosec"
                Write-InfoMsg "Skipping security scan. Install manually: go install github.com/securego/gosec/v2/cmd/gosec@latest"
            } else {
                Write-Success "gosec installed"
            }
        }
        
        if (Get-Command gosec -ErrorAction SilentlyContinue) {
            Write-InfoMsg "Running gosec security scanner..."
            gosec -no-fail -fmt sarif -out results.sarif ./cmd/... ./pkg/...
            if (Test-Path "results.sarif") {
                Write-Success "Security scan completed. Results saved to results.sarif"
            } else {
                Write-InfoMsg "No SARIF file generated (no issues found or scan skipped)"
            }
        }
    }
}

# Test Examples Job
if ($Job -eq "all" -or $Job -eq "test-examples") {
    if (-not $SkipExamples) {
        Write-Step "Running Test Examples Job"
        
        $examples = @(
            @{Path = "examples/load-balancing"; Name = "load-balancing"},
            @{Path = "examples/fluxor-project/all-in-one"; Name = "all-in-one"},
            @{Path = "examples/fluxor-project/api-gateway"; Name = "api-gateway"},
            @{Path = "examples/fluxor-project/payment-service"; Name = "payment-service"},
            @{Path = "examples/todo-api"; Name = "todo-api"},
            @{Path = "examples/workflow-demo"; Name = "workflow-demo"}
        )
        
        foreach ($example in $examples) {
            if (Test-Path $example.Path) {
                Write-InfoMsg "Testing $($example.Name) example..."
                Push-Location $example.Path
                try {
                    if (Test-Path "main.go") {
                        go build -o "/tmp/$($example.Name).exe" main.go
                        if ($?) {
                            Write-Success "$($example.Name) example builds successfully"
                        } else {
                            Write-Error "$($example.Name) example build failed"
                            Pop-Location
                            exit 1
                        }
                    } else {
                        Write-InfoMsg "$($example.Name) example: main.go not found, skipping"
                    }
                } finally {
                    Pop-Location
                }
            } else {
                Write-InfoMsg "$($example.Name) example not found, skipping"
            }
        }
        
        Write-InfoMsg "Running go vet on examples..."
        Get-ChildItem -Path examples -Recurse -Filter "*.go" | 
            ForEach-Object { $_.DirectoryName } | 
            Sort-Object -Unique | 
            ForEach-Object {
                if ((Test-Path "$_/go.mod") -or (Test-Path "$_/main.go")) {
                    Write-InfoMsg "Checking $_..."
                    Push-Location $_
                    go vet ./... 2>&1 | Out-Null
                    Pop-Location
                }
            }
    }
}

# Docker Job
if ($Job -eq "all" -or $Job -eq "docker") {
    if (-not $SkipDocker) {
        Write-Step "Running Docker Build Job"
        
        $dockerInstalled = Get-Command docker -ErrorAction SilentlyContinue
        if (-not $dockerInstalled) {
            Write-InfoMsg "Docker is not installed. Skipping Docker build."
            Write-InfoMsg "Install Docker Desktop for Windows to enable Docker builds."
        } else {
            Write-InfoMsg "Building Docker image (enterprise)..."
            docker build -f ./cmd/enterprise/Dockerfile -t fluxor-enterprise:latest .
            if (-not $?) {
                Write-Error "Docker build failed"
                exit 1
            }
            Write-Success "Docker image built: fluxor-enterprise:latest"
            
            Write-InfoMsg "Testing Docker image..."
            docker images | Select-String "fluxor-enterprise"
        }
    }
}

Write-Step "CI Run Complete"
Write-Success "All selected jobs completed successfully!"

