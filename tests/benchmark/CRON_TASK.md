# PinchTab Optimization Cron Task

**Goal: improve the agent lane against a known-good benchmark environment.**

Baseline is not part of the recurring agent task. It is a direct shell
verification that should be run after infrastructure, fixture, or benchmark
changes to confirm the environment still behaves as expected.

---

## Baseline Gate

Run baseline manually when any of these change:

- PinchTab server or CLI behavior
- benchmark fixtures
- `tests/benchmark/scripts/baseline.sh`
- benchmark Docker setup

Command:

```bash
cd ~/dev/pinchtab/tests/benchmark
./scripts/run-optimization.sh
./scripts/baseline.sh
./scripts/finalize-report.sh "$(cat ./results/current_baseline_report.txt)"
```

Only continue with optimization work if baseline is green.

---

## Recurring Optimization Task

### Step 1 — Setup
```bash
cd ~/dev/pinchtab/tests/benchmark
git checkout feat/benchmark && git pull --rebase origin feat/benchmark
./scripts/run-optimization.sh
# Note the TIMESTAMP from output
```

### Step 2 — Run Agent Benchmark
Execute `AGENT_TASKS.md` using `skills/pinchtab/SKILL.md` as the guide.

- Use the benchmark wrapper `./scripts/pt`
- Record results with `record-step.sh --type agent`
- Verify answers with `verify-step.sh`
- Log every command executed
- Do not use baseline as an execution guide

### Step 3 — Gap Analysis
Compare agent results against the latest green baseline report and classify failures:

| Failure Type | Cause | Fix |
|---|---|---|
| **Wrong endpoint** | Agent used `/text` when should use `/snapshot` | Improve `SKILL.md` |
| **Wrong selector** | Agent guessed selector incorrectly | Improve `SKILL.md` or fixture |
| **Missing step** | Agent skipped a required action | Clarify `AGENT_TASKS.md` |
| **Wrong URL** | Agent used wrong fixture path | Clarify `AGENT_TASKS.md` |
| **API bug** | Endpoint behaves unexpectedly | Fix PinchTab code |
| **Test ambiguity** | Verification string hard to find | Fix fixture or `scripts/baseline.sh` |

### Step 4 — Make Exactly 1 Change

Priority order:
1. **API Bug** → Fix PinchTab Go code, commit as `fix: <description>`
2. **Skill Gap** → Improve `SKILL.md`, commit as `docs(skill): <description>`
3. **Test Ambiguity** → Fix fixture HTML or `scripts/baseline.sh`, commit as `test: <description>`
4. **Agent Task Clarity** → Improve `AGENT_TASKS.md`, commit as `test(agent): <description>`
5. **No Gaps Found** → Add new benchmark coverage, with matching baseline + agent cases

**If agent is already at 100%**: add harder cases.

### Step 5 — Verify the Fix Makes Sense
Before committing, ask:

- will this change help the agent pick the right tool next time?
- if this is a skill/docs fix, does it directly address the observed failure?
- if this is a benchmark fix, is the expected signal reachable via the real tool flow?

### Step 6 — Commit & Push
```bash
cd ~/dev/pinchtab
git add -A
git commit -m "<type>: <clear description of what changed and why>"
git push origin feat/benchmark
```

### Step 7 — Log the Run
Append to `results/optimization_log.md`:

```markdown
## Run #N — YYYY-MM-DD HH:MM

**Results:**
- Baseline reference: X/Y (Z%)
- Agent: X/Y (Z%)
- Gap: N steps

**Agent Failures:**
- Step X.Y: [failure type] — [what went wrong]
- Step X.Y: [failure type] — [what went wrong]

**Root Cause:**
[One clear sentence explaining the pattern]

**Change Made:**
- Type: api|skill|test|agent-task
- Description: [What changed]
- Expected impact: [Why this should close the gap]
- Commit: [hash]

**Next Focus:**
[What to watch in the next run]
```

### Step 8 — Report to User
Send a concise update:

```text
Run #N complete
Baseline reference: X% | Agent: Y% | Gap: N steps
Failure: [brief description]
Fix: [what changed] ([commit])
```

---

## Success Criteria

- **Short term**: Agent pass rate ≥ 95%
- **Long term**: Agent matches the latest green baseline on all existing tests
- **Ongoing**: When the gap closes, increase test complexity

## What NOT to do

- Do not run baseline as an agent task
- Do not make multiple changes per run
- Do not skip root-cause analysis
- Do not weaken tests just to improve the score
- Do not commit a change that does not directly address an observed failure
