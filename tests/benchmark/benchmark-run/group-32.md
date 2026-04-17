# Group 32: Dynamic iframe (inserted after load)

### 32.1 Wait for a late iframe, then interact with it
Navigate to `http://fixtures/iframe-dynamic.html`. The iframe is inserted ~1.2 s after load (use `wait --text "IFRAME_DYNAMIC_ATTACHED"`). Then scope into `#late-frame`, fill `#iframe-input` with "Late World", click `#iframe-submit`, and verify the inner result marker.

**Verify**: Scoped snapshot contains `IFRAME_INPUT_RECEIVED_LATE_WORLD`.

---

