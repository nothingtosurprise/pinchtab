# Group 19: iFrame

### 19.1 Read iframe content
Navigate to `http://fixtures/iframe.html` and extract content from inside the embedded same-origin iframe.

**Verify**: The iframe's inner content includes `IFRAME_INNER_CONTENT_LOADED`.

### 19.2 Type into iframe input (native frame scope)
Use `pinchtab frame '#content-frame'` to scope into the iframe, then `fill` the input with "Hello World" and `click` the Save button. Verify via a scoped `snap` — `text --full` doesn't pierce iframes. Reset with `pinchtab frame main` afterwards.

**Verify**: Scoped snapshot contains `IFRAME_INPUT_RECEIVED_HELLO_WORLD`.

---

