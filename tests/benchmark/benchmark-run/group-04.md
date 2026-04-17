# Group 4: SPA State Management

### 4.1 Read initial app state
Navigate to `http://fixtures/spa.html?reset=1` (the `?reset=1` query param clears any previous localStorage state so the SPA starts with its default 3 tasks). Read the current task list — how many tasks exist, how many are active vs done?

**Verify**: Found 3 total, 2 active, 1 done (verify `TASK_STATS_TOTAL_3_ACTIVE_2_DONE_1`).

### 4.2 Add a new high-priority task
Add a task called "Automate deployment" with high priority.

**Verify**: Task appeared in the list (`TASK_ADDED_AUTOMATE_DEPLOYMENT_PRIORITY_HIGH`).

### 4.3 Delete a task
Delete the task titled "Write benchmark tests".

**Verify**: Task count changed (went from 4 to 3).

---

