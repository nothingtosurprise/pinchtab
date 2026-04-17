# Group 26: Accordion

### 26.1 Open section A
Navigate to `http://fixtures/accordion.html`. Click the header for Section A to expand it.

**Verify**: Page text includes `ACCORDION_SECTION_A_OPEN`.

### 26.2 Open section B
Click the header for Section B. Because the accordion is exclusive-expand, Section A should close.

**Verify**: Page text includes `ACCORDION_SECTION_B_OPEN`, and Section A's `aria-expanded` attribute is now `"false"` (use `eval` to check the attribute if needed).

---

