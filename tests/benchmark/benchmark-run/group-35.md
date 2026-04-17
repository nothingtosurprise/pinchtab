# Group 35: Long-form article (Medium/Substack style)

### 35.1 Read the article with default `text`
Navigate to `http://fixtures/article.html`. Extract the page's main content using the default `text` (Readability) mode. Default mode is the right choice for article-style pages.

**Verify**: The extracted text contains both `ARTICLE_PUBLISHED_2026_04_15` and `ARTICLE_WORD_COUNT_MARKER_323` (both inside the article body — Readability keeps them).

### 35.2 See the chrome that Readability drops
Re-extract with `text --full` and confirm the footer is included this time.

**Verify**: `--full` output contains `FOOTER_COPYRIGHT_MARKER` (which the default mode trims).

---

