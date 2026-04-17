# Group 27: Contenteditable editor

### 27.1 Type into the rich-text editor
Navigate to `http://fixtures/editor.html`. Type the text `Hello rich text` into the `#editor` div. `contenteditable` elements don't have `.value`, so use `type` (keyboard events), not `fill`.

**Verify**: Page text contains `EDITOR_CHARS=15` and the mirror shows `Hello rich text`.

### 27.2 Commit by pressing Enter
Press Enter (the editor intercepts Enter to commit the current buffer to a separate marker).

**Verify**: Page text contains `EDITOR_COMMITTED=Hello rich text`.

---

