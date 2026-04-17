# Group 34: Sandboxed iframe

### 34.1 Click inside a sandboxed iframe
Navigate to `http://fixtures/iframe-sandbox.html`. The iframe has `sandbox="allow-scripts allow-same-origin"`. Scope into `#sandboxed`, click `#sandbox-button`, and verify.

**Verify**: Scoped snapshot contains `SANDBOX_CLICKED=YES`.

---

