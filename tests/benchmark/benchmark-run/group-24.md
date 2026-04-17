# Group 24: Keyboard events

### 24.1 Press Escape
Navigate to `http://fixtures/keyboard.html` (it auto-focuses an input so keyboard events land). Press the Escape key.

**Verify**: Page text contains `KEYBOARD_ESCAPE_PRESSED`.

### 24.2 Press 'a' then Enter
Without reloading, press the `a` key, then the `Enter` key.

**Verify**: Page text now contains all three markers in order: `KEYBOARD_ESCAPE_PRESSED`, `KEYBOARD_KEY_A_PRESSED`, `KEYBOARD_ENTER_PRESSED`.

---

