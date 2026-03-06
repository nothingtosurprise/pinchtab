# Testing

## Quick Start with pdev

The `pdev` developer toolkit is the easiest way to run checks and tests:

```bash
./pdev                    # Interactive picker
./pdev test               # All tests (unit + integration)
./pdev test unit          # Unit tests only
./pdev test integration   # Integration tests only
./pdev check              # All checks (format, vet, build, test, lint)
./pdev check go           # Go checks only
./pdev check security     # Gosec security scan
./pdev hooks              # Install git hooks (pre-commit)
./pdev doctor             # Setup dev environment
```

## Unit Tests

```bash
go test ./...
# or
./pdev test unit
```

## Integration Tests

Integration tests launch a real pinchtab server with Chrome and run HTTP-level tests against it.

```bash
go test -tags integration ./tests/integration/ -v -timeout 5m
# or
./pdev test integration
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PINCHTAB_TEST_PORT` | `19867` | Port for the test server |
| `PINCHTAB_TEST_KEEP_DIR` | _(unset)_ | Set to any value to preserve the test dir after tests finish |
| `CHROME_BINARY` | _(auto-detect)_ | Path to Chrome binary (used in CI) |
| `CI` | _(unset)_ | Set to `true` for longer health check timeouts (60s vs 30s) |

### Temp Directory Layout

Each test run creates a single temp directory under `/tmp/pinchtab-test-*/`:

```
/tmp/pinchtab-test-123456789/
├── pinchtab          # Compiled test binary
├── state/            # Dashboard state (profiles, instances)
└── profiles/         # Chrome user-data directories
```

Everything is cleaned up automatically when tests finish. To inspect after a failure:

```bash
PINCHTAB_TEST_KEEP_DIR=1 go test -tags integration ./tests/integration/ -v
```

### Test Utilities (`tests/testutil/`)

The `testutil` package provides reusable helpers:

- **`testutil.StartServer(cfg)`** — Builds, launches, and waits for a pinchtab server. Returns a `*Server` with `Stop()` for cleanup.
- **`testutil.NewClient(url)`** — HTTP client with `Get`, `Post`, `PostRaw`, `Patch`, `Delete`, `PostWithRetry` methods.
- **`testutil.LaunchInstance(url)`** — Creates and waits for a test instance.
- **`testutil.JSONField(t, data, key)`** — Extracts a string field from JSON response body.
- **`testutil.FindRepoRoot()`** — Walks up to find `go.mod`.
- **`testutil.WaitForHealth(url, timeout)`** — Polls `/health` with context-based timeout.
- **`testutil.TerminateProcessGroup(cmd, timeout)`** — SIGTERM to process group, SIGKILL on timeout.

### Writing New Tests

New tests should use `testutil.Client` and `testutil.NewTestServer` (or `StartServer`) instead of the legacy `httpGet`/`httpPost` wrappers. The legacy helpers are kept only for backward compatibility.

```go
//go:build integration

package integration

func TestMyFeature(t *testing.T) {
    navigate(t, "https://example.com")
    
    code, body := client.Get(t, "/snapshot")
    if code != 200 {
        t.Fatalf("expected 200, got %d", code)
    }
    
    value := testutil.JSONField(t, body, "someField")
    // ...
}
```

For tests that need their own isolated server (outside the shared `TestMain` setup):

```go
func TestIsolatedFeature(t *testing.T) {
    srv := testutil.NewTestServer(t, testutil.DefaultConfig())
    c := testutil.NewClient(srv.URL)
    
    code, _ := c.Get(t, "/health")
    // ... srv.Stop() is called automatically via t.Cleanup
}
```

### Process Cleanup

The test server runs in its own process group (`Setpgid`). On shutdown, `SIGTERM` is sent to the entire group, killing Chrome and all child processes. If graceful shutdown fails after 10 seconds, `SIGKILL` is sent to the group.
