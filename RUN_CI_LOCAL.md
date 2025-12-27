# Running GitHub CI Locally

This guide explains how to run the GitHub Actions CI workflow locally on your machine.

## Quick Start

### Option 1: Simple Batch Script (Windows)

Run the test job:
```cmd
run-ci-local.bat
```

### Option 2: PowerShell Script (Windows)

Run specific jobs:
```powershell
# Run all jobs
.\run-ci-local.ps1

# Run only test job
.\run-ci-local.ps1 -Job test

# Run only build job
.\run-ci-local.ps1 -Job build

# Run only benchmark job
.\run-ci-local.ps1 -Job benchmark

# Run only security scan
.\run-ci-local.ps1 -Job security

# Run only examples test
.\run-ci-local.ps1 -Job test-examples

# Skip specific jobs when running all
.\run-ci-local.ps1 -SkipTests -SkipDocker
```

### Option 3: Using `act` (Full GitHub Actions Simulation)

`act` is a tool that runs GitHub Actions workflows locally using Docker. It provides the most accurate simulation of the CI environment.

#### Installation

**Windows (using Chocolatey):**
```cmd
choco install act-cli
```

**Windows (using Scoop):**
```cmd
scoop install act
```

**Or download from:**
https://github.com/nektos/act/releases

#### Prerequisites

- Docker Desktop for Windows must be installed and running
- WSL2 (Windows Subsystem for Linux) is recommended

#### Usage

```bash
# List all available workflows
act -l

# Run the CI workflow (all jobs)
act

# Run a specific job
act -j test
act -j build
act -j benchmark
act -j security
act -j test-examples
act -j docker

# Run on a specific event
act push
act pull_request

# Run with specific matrix values
act -j test --matrix go-version:1.24

# Use a larger runner image (recommended for better compatibility)
act -P ubuntu-latest=catthehacker/ubuntu:act-latest
```

#### Limitations

- Some actions may not work perfectly (e.g., `actions/upload-artifact` saves to local directory)
- Docker-in-Docker may require additional setup
- Secrets need to be provided via `.secrets` file or environment variables

## Manual CI Steps

If you prefer to run CI steps manually, here are the commands from the workflow:

### Test Job

```bash
# Download dependencies
go mod download

# Verify dependencies
go mod verify

# Run go vet
go vet ./...

# Check formatting
gofmt -s -l .

# Run tests
go test -v -race -coverprofile=coverage.out -covermode=atomic -timeout 10m ./...

# View coverage
go tool cover -func=coverage.out
```

### Build Job

```bash
# Create bin directory
mkdir -p bin

# Build binaries
go build -v -o bin/fluxor ./cmd/main.go
go build -v -o bin/fluxor-example ./cmd/example/main.go
go build -v -o bin/fluxor-enterprise ./cmd/enterprise/main.go
```

### Benchmark Job

```bash
go test -bench=. -benchmem -run=^$ ./... | tee benchmark.txt
```

### Security Scan

```bash
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run scan
gosec -no-fail -fmt sarif -out results.sarif ./cmd/... ./pkg/...
```

### Test Examples

```bash
# Test each example
cd examples/load-balancing && go build -o /tmp/load-balancing main.go
cd examples/fluxor-project/all-in-one && go build -o /tmp/all-in-one main.go
cd examples/fluxor-project/api-gateway && go build -o /tmp/api-gateway .
cd examples/fluxor-project/payment-service && go build -o /tmp/payment-service .
```

### Docker Build

```bash
docker build -f ./cmd/enterprise/Dockerfile -t fluxor-enterprise:latest .
docker images | grep fluxor-enterprise
```

## Troubleshooting

### CI Job Hanging/Timeout

If your CI job is hanging or timing out (like the run at https://github.com/quadgatefoundation/fluxor/actions/runs/20538384173/job/58999509520), use the debug scripts to identify which test is hanging:

**Windows (PowerShell):**
```powershell
.\debug-hanging-tests.ps1
```

**Linux/macOS:**
```bash
chmod +x debug-hanging-tests.sh
./debug-hanging-tests.sh
```

**Manual debugging:**
```bash
# Run tests with verbose output and identify hanging test
go test -v -race -timeout=2m ./pkg/... 2>&1 | tee test-output.log

# Run specific package to isolate issue
go test -v -race -timeout=1m ./pkg/core

# Check for tests with long timeouts or infinite loops
grep -r "time.Sleep\|select\|for.*{" pkg/*/*_test.go
```

**Common causes:**
- Tests waiting on channels that never receive
- `select` statements without timeouts
- Infinite loops in test code
- Race detector finding issues and hanging

### PowerShell Script Issues

If the PowerShell script doesn't work, try:
1. Check PowerShell execution policy: `Get-ExecutionPolicy`
2. Run with bypass: `powershell -ExecutionPolicy Bypass -File .\run-ci-local.ps1`
3. Use the batch file instead: `run-ci-local.bat`

### Docker/act Issues

- Ensure Docker Desktop is running
- For WSL2 issues, see: https://docs.docker.com/desktop/wsl/
- Try using a different runner image: `act -P ubuntu-latest=catthehacker/ubuntu:act-latest`

### Go Version

The CI uses Go 1.24. Ensure your local Go version is compatible:
```bash
go version
```

### Test Timeouts

If tests are timing out:
1. Check for hanging tests using debug scripts
2. Review test code for `select` statements without timeouts
3. Check for infinite loops or blocking operations
4. Consider splitting long-running tests into separate jobs

## CI Workflow Overview

The CI workflow (`.github/workflows/ci.yml`) includes:

1. **test** - Runs tests, vet, fmt checks, and generates coverage
2. **build** - Builds all binaries (main, example, enterprise)
3. **benchmark** - Runs performance benchmarks
4. **security** - Runs gosec security scanner
5. **test-examples** - Tests all example projects
6. **docker** - Builds Docker images

Jobs run in parallel where possible, with `build`, `benchmark`, and `docker` depending on `test` completing successfully.

