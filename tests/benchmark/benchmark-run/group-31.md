# Group 31: Nested iframes (3 levels deep)

### 31.1 Click a button in the deepest frame
Navigate to `http://fixtures/iframe-nested.html`. The outer page embeds `#level-2`, which in turn embeds `#level-3`, which contains `#deep-button`. Drill through the two frame hops and click the button. Verify via a scoped `snap` before resetting to `main`.

**Verify**: Scoped snapshot contains `DEEP_CLICKED=YES_LEVEL_3`.

---

