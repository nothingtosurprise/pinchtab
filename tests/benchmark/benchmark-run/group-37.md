# Group 37: Q&A thread (Stack-Overflow style)

### 37.1 Find the accepted answer id
Navigate to `http://fixtures/qa.html`. The accepted answer carries `data-accepted="true"` on its `<div>` wrapper. Use `eval` to return that element's `id`.

**Verify**: The `eval` result equals `"a-2"`.

### 37.2 Extract the accepted answer's body
Scope a snapshot to `#a-2` and verify the body content.

**Verify**: Scoped snapshot contains `ANSWER_2_BODY_MARKER` and `ACCEPTED_ANSWER_ID_A2`.

---

