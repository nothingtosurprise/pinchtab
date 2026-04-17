# Group 11: State Persistence & Page Reload

### 11.1 Add an item and verify after page reload
Navigate to `http://fixtures/spa.html?reset=1` (starts with clean state). Add a task titled exactly "Persistent Task Test". Then navigate away to `http://fixtures/` and back to `http://fixtures/spa.html` (WITHOUT the reset param) to verify localStorage persistence.

**Verify**: After reload, the task still appears in the list (`TASK_PERSISTENT_TEST_FOUND_AFTER_RELOAD`).

### 11.2 Logout and log back in
From the logged-in dashboard, click Sign Out to log out. Then log in again with username "benchmark" / password "test456".

**Verify**: Successfully logged back in and dashboard shows `SESSION_RENEWED`.

---

