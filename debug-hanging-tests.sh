#!/bin/bash
# Debug script to identify hanging tests
# Run this if CI tests are timing out

echo "========================================"
echo "Debugging Hanging Tests"
echo "========================================"
echo ""

# Test 1: Run tests with timeout and identify which one hangs
echo "[1/4] Running tests with individual timeouts..."

test_packages=$(go list ./...)
failed=()
passed=()

for pkg in $test_packages; do
    echo "Testing: $pkg"
    
    # Run test with timeout
    timeout 120 go test -v -timeout=2m -race "$pkg" 2>&1
    exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        echo "  ✓ $pkg"
        passed+=("$pkg")
    elif [ $exit_code -eq 124 ]; then
        echo "  ⚠ $pkg - TIMED OUT (likely hanging)"
        failed+=("$pkg")
    else
        echo "  ✗ $pkg"
        failed+=("$pkg")
    fi
done

echo ""
echo "[2/4] Summary:"
echo "  Passed: ${#passed[@]}"
echo "  Failed/Timeout: ${#failed[@]}"

if [ ${#failed[@]} -gt 0 ]; then
    echo ""
    echo "[3/4] Failed/Timeout packages:"
    for pkg in "${failed[@]}"; do
        echo "  - $pkg"
    done
    
    echo ""
    echo "[4/4] Running detailed test on first failed package..."
    if [ ${#failed[@]} -gt 0 ]; then
        first_failed="${failed[0]}"
        echo "Testing: $first_failed"
        go test -v -race -timeout=1m "$first_failed"
    fi
else
    echo ""
    echo "[3/4] All packages passed!"
    echo "[4/4] Running full test suite..."
    go test -v -race -timeout=10m ./...
fi

