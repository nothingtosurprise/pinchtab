# Group 1: Reading & Extracting Real Content

### 1.1 Get the full list of categories from the wiki index
Navigate to `http://fixtures/wiki.html` and extract all category names with their article counts.

**Verify**: Can name at least 2 categories and their article counts (e.g. "Programming Languages: 12 articles").

### 1.2 Navigate by clicking a link
From the wiki index, click the "Go (programming language)" link to navigate to the Go article.

**Verify**: You are now on the Go article page (not the wiki index).

### 1.3 Extract structured data from a table
On the Go article, read the infobox table and answer: Who designed Go, and what year did it first appear?

**Verify**: Answer contains "Robert Griesemer" (or "Rob Pike" or "Ken Thompson") and "2009".

### 1.4 Count list items
On the Go article, count how many key features are listed.

**Verify**: Answer is 6 (verify against `FEATURE_COUNT_6`).

### 1.5 Read all article headlines from articles page
Navigate to `http://fixtures/articles.html` and list all article titles.

**Verify**: Found at least 3 articles including "The Future of Artificial Intelligence".

### 1.6 Read dashboard metrics
Navigate to `http://fixtures/dashboard.html` and extract: Total Users, Revenue, and Conversion Rate.

**Verify**: Found `24,582` users AND `$1,284,930` revenue.

---

