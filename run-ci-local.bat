@echo off
REM Local CI Runner - Simulates GitHub Actions CI workflow
REM This script runs the same steps as .github/workflows/ci.yml

setlocal enabledelayedexpansion

echo ========================================
echo Running Local CI - Test Job
echo ========================================
echo.

echo [1/6] Checking Go installation...
go version
if errorlevel 1 (
    echo ERROR: Go is not installed or not in PATH
    exit /b 1
)
echo.

echo [2/6] Downloading dependencies...
go mod download
if errorlevel 1 (
    echo ERROR: Failed to download dependencies
    exit /b 1
)
echo.

echo [3/6] Verifying dependencies...
go mod verify
if errorlevel 1 (
    echo ERROR: Dependency verification failed
    exit /b 1
)
echo.

echo [4/6] Running go vet...
go vet ./...
if errorlevel 1 (
    echo ERROR: go vet found issues
    exit /b 1
)
echo.

echo [5/6] Checking go fmt...
for /f %%i in ('gofmt -s -l .') do (
    echo ERROR: The following files are not formatted:
    gofmt -s -l .
    exit /b 1
)
echo.

echo [6/6] Running tests with race detector...
go test -v -race -coverprofile=coverage.out -covermode=atomic -timeout 10m ./...
if errorlevel 1 (
    echo ERROR: Tests failed
    exit /b 1
)
echo.

if exist coverage.out (
    echo Coverage report generated: coverage.out
    go tool cover -func=coverage.out | findstr "total:"
)

echo.
echo ========================================
echo Test Job Complete - All tests passed!
echo ========================================

