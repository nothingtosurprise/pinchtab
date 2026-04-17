# Group 8: Error Handling

### 8.1 Handle 404 gracefully
Try to navigate to a page that doesn't exist: `http://fixtures/missing-page-abc.html`.

**Verify**: Got a response (404 or error), no crash, server still responsive after.

### 8.2 Handle missing element gracefully
On any page, try to click an element with ID `#fake-button-that-does-not-exist`.

**Verify**: Got a clear error message, not a crash or hang.

---

