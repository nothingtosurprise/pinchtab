# Group 21: Async / awaitPromise

### 21.1 Await a promise-returning function
Navigate to `http://fixtures/async.html`. The page exposes `window.fetchPayload()`, which returns a `Promise` that resolves after a short delay. Use `eval` to call it and retrieve the **resolved** value, not a Promise wrapper.

**Verify**: The resolved value contains `ASYNC_PAYLOAD_READY_42`.

### 21.2 Await a promise resolving to an object
On the same page, call `window.fetchUser()` and retrieve the resolved object so you can read a field from it.

**Verify**: The resolved object's `name` field equals `ASYNC_USER_NAME_ADA`.

---

