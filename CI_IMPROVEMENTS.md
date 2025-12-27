# CI Pipeline Improvements

## Issue Identified

The CI job at [run #20538384173](https://github.com/quadgatefoundation/fluxor/actions/runs/20538384173/job/58999509520) was hanging for over 11 hours, indicating a test timeout or infinite hang.

## Improvements Made

### 1. Job-Level Timeout

Added a 20-minute timeout at the job level to prevent infinite hangs:

```yaml
test:
  name: Test
  runs-on: ubuntu-latest
  timeout-minutes: 20  # ← Added
```

### 2. Enhanced Test Diagnostics

Improved the test step with:
- Better logging and progress indicators
- Test output capture to `test-output.log`
- Diagnostic information on failure:
  - Last 50 lines of test output
  - List of test files
  - Running processes (to identify hanging tests)

```yaml
- name: Run tests
  timeout-minutes: 15
  run: |
    echo "Starting tests with race detector..."
    go test -v -race ... 2>&1 | tee test-output.log || {
      echo "Last 50 lines of test output:"
      tail -n 50 test-output.log
      echo "Checking for hanging processes..."
      ps aux | grep -E "(go|test)" | head -10
      exit 1
    }
```

### 3. Debug Scripts for Local Testing

Created scripts to identify hanging tests locally:

- **`debug-hanging-tests.ps1`** (Windows PowerShell)
- **`debug-hanging-tests.sh`** (Linux/macOS)

These scripts:
- Run each test package individually with a 2-minute timeout
- Identify which package is hanging
- Provide detailed output for debugging

### 4. Updated Documentation

Enhanced `RUN_CI_LOCAL.md` with:
- Troubleshooting section for hanging tests
- Instructions for using debug scripts
- Common causes and solutions

## How to Use

### If CI is Hanging

1. **Run debug script locally:**
   ```powershell
   # Windows
   .\debug-hanging-tests.ps1
   
   # Linux/macOS
   ./debug-hanging-tests.sh
   ```

2. **Check the output** to identify which test package is hanging

3. **Run the specific package** with verbose output:
   ```bash
   go test -v -race -timeout=1m ./pkg/core
   ```

4. **Look for:**
   - `select` statements without timeouts
   - Infinite loops
   - Blocking channel operations
   - Missing context cancellation

### Manual Debugging

```bash
# Run with verbose output and capture logs
go test -v -race -timeout=2m ./pkg/... 2>&1 | tee test-output.log

# Search for potential hanging patterns
grep -r "time.Sleep\|select\|for.*{" pkg/*/*_test.go

# Check for tests without timeouts
grep -r "func Test" pkg/*/*_test.go | grep -v "timeout\|Timeout"
```

## Expected Behavior

After these improvements:

1. **CI jobs will timeout after 20 minutes** instead of hanging indefinitely
2. **Better diagnostics** will help identify which test is hanging
3. **Debug scripts** make it easy to reproduce issues locally
4. **Test output logs** are captured for analysis

## Next Steps

1. Monitor the next CI run to see if the timeout is triggered
2. If timeout occurs, use debug scripts to identify the hanging test
3. Fix the specific test that's causing the hang
4. Consider splitting long-running tests into separate jobs

## Related Files

- `.github/workflows/ci.yml` - Main CI workflow
- `debug-hanging-tests.ps1` - Windows debug script
- `debug-hanging-tests.sh` - Linux/macOS debug script
- `RUN_CI_LOCAL.md` - Local CI execution guide
- `CI_FIXES.md` - Previous CI fixes documentation

