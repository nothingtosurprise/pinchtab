# Group 36: Search results page (SERP)

### 36.1 Find a specific result by id
Navigate to `http://fixtures/serp.html`. The page has 6 result cards (`#r-1`..`#r-6`). Extract just the third card's content. A scoped snapshot (`snap --selector "#r-3"`) is the direct path.

**Verify**: The scoped output contains `RESULT_3_TITLE` and `RESULT_3_SNIPPET_MARKER`.

### 36.2 Count all result cards
Use a full text extraction to verify all six results are present in one pass.

**Verify**: Output contains all of `RESULT_1_TITLE` through `RESULT_6_TITLE` and the summary `SERP_RESULT_COUNT_6`. Default Readability trims SERPs — use `text --full`.

---

