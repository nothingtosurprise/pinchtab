# Agent Browser Setup

Lane-specific setup and operating guidance for running the benchmark with
`agent-browser`.

## Read Order

1. Read `./benchmark-run/index.md`
2. Load the live skill through the wrapper:

```bash
./scripts/ab skills get agent-browser --full
```

This file covers setup, wrapper usage, and Group 0 differences. Shared task
groups live in `benchmark-run/`.

## Recording

Record each completed step as factual `answer`, `fail`, or `skip`, then verify:

```bash
./scripts/record-step.sh --type agent-browser <group> <step> answer "<what you saw>" "notes"
./scripts/verify-step.sh --type agent-browser <group> <step> <pass|fail|skip> "verification notes"
```

Do not self-grade inside the answer payload. Keep the answer factual.

## Wrapper

For benchmark runs in this repo, do not call `agent-browser` directly. Use:

```bash
./scripts/ab ...
```

That wrapper:

- executes `agent-browser` inside the benchmark Docker service
- preserves the shared browser session across commands
- logs tool calls to `results/agent_browser_commands.ndjson`

## Benchmark Workflow

1. Start the lane:

```bash
cd tests/benchmark
./scripts/run-agent-browser-benchmark.sh
```

2. Load the live CLI skill:

```bash
./scripts/ab skills get agent-browser --full
```

3. Run browser actions through the wrapper:

```bash
./scripts/ab open http://fixtures/
./scripts/ab snapshot -i -c
./scripts/ab click @e2
./scripts/ab fill @e3 "agent@benchmark.test"
```

4. Record execution results:

```bash
./scripts/record-step.sh --type agent-browser 1 1 answer \
  "opened fixtures home and got refs e1-e13" "completed"
```

5. Verify later in a separate pass:

```bash
./scripts/verify-step.sh --type agent-browser 1 1 pass \
  "matched expected homepage state"
```

## Environment

- Fixtures: `http://fixtures/`
- Session name: `benchmark` by default (`AGENT_BROWSER_SESSION` overrides)
- Browser driver: Docker service `agent-browser`
- Pages: `/`, `/wiki.html`, `/wiki-go.html`, `/articles.html`, `/search.html`,
  `/form.html`, `/dashboard.html`, `/ecommerce.html`, `/spa.html`, `/login.html`

## Operating Guidance

- Treat `./scripts/ab skills get agent-browser --full` as the source of truth for
  current command syntax and workflows
- Prefer refs from `snapshot -i -c` over brittle selectors
- Re-snapshot after navigation or any DOM-changing action
- Keep one session for the whole benchmark lane unless a task explicitly needs a reset

## Group 0 Override

For the `agent-browser` lane, Group 0 differs from `benchmark-run/group-00.md`:

- 0.1 `./scripts/ab open http://fixtures/` succeeds
- 0.2 `./scripts/ab snapshot -i -c` returns interactive refs
- 0.3 session state persists across multiple `./scripts/ab ...` commands

From Group 1 onward, use `benchmark-run/group-01.md` through `benchmark-run/group-38.md`.
