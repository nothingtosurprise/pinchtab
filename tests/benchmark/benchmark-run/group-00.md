# Group 0: Setup & Diagnosis

### 0.1 Server reachable
Check that the PinchTab server is healthy.

**Verify**: Server responds with `status: ok`.

### 0.2 Auth is required
Make a request to the server with a **wrong** token (`PINCHTAB_TOKEN=wrong-token ./scripts/pt health`) and confirm it is rejected. The `pt` wrapper always injects the benchmark token by default, so you must explicitly override it to test auth rejection.

**Verify**: Response is HTTP 401 or contains `unauthorized`.

### 0.3 Auth works with token
Repeat the same request WITH the bearer token and confirm it succeeds.

**Verify**: Response is HTTP 200.

### 0.4 Instance available
Confirm at least one Chrome instance is running. If none exist, start one.

**Verify**: Health response shows `defaultInstance.status == "running"` (or after starting one, the new instance is running).

### 0.5 List existing tabs
Get the current list of open tabs.

**Verify**: A list (possibly empty) is returned without error.

### 0.6 Clean stale tabs
If any tabs from previous runs are open, close them so the benchmark starts from a known state.

**Verify**: After cleanup, the tab list is empty (or only contains a single about:blank tab).

### 0.7 Network reach to target
Navigate to `http://fixtures/` and confirm the fixtures server is reachable from PinchTab.

**Verify**: Navigate returns successfully and the page contains benchmark content.

### 0.8 Capture initial tab ID
Save the tab ID returned by the navigate in 0.7. Use this tab ID for all subsequent tasks to avoid creating new tabs.

**Verify**: A tab ID was captured and matches what `GET /tabs` reports.

---

