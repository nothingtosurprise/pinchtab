# PinchTab Setup

Lane-specific setup and operating guidance for running the benchmark with
PinchTab.

## Read Order

1. Read `../../skills/pinchtab/SKILL.md`
2. Read `./benchmark-run/index.md`

`benchmark-run/index.md` contains the shared task map. This file only covers
setup, wrapper usage, and PinchTab-specific expectations.

## Recording

Record each completed step as factual `answer`, `fail`, or `skip`, then verify:

```bash
./scripts/record-step.sh --type pinchtab <group> <step> answer "<what you saw>" "notes"
./scripts/verify-step.sh --type pinchtab <group> <step> <pass|fail|skip> "verification notes"
```

Do not self-grade inside the answer payload. Keep the answer factual.

## Environment

- PinchTab: `http://localhost:9867`, token: `benchmark-token`
- Fixtures: `http://fixtures/` (running in Docker as `fixtures` hostname)
- Pages: `/`, `/wiki.html`, `/wiki-go.html`, `/articles.html`, `/search.html`,
  `/form.html`, `/dashboard.html`, `/ecommerce.html`, `/spa.html`, `/login.html`

## Wrapper

Use only:

```bash
./scripts/pt ...
```

Do not call `pinchtab` directly for benchmark runs.

The wrapper:

- executes the CLI inside the benchmark Docker container
- forwards `PINCHTAB_TOKEN`, `PINCHTAB_SERVER`, and `PINCHTAB_TAB`
- preserves one shared tab across commands when `PINCHTAB_TAB` is set

## Recommended Flow

```bash
./scripts/pt health
./scripts/pt tab
export PINCHTAB_TAB=$(./scripts/pt nav http://fixtures/)
./scripts/pt snap -i -c
```

After that:

- use `./scripts/pt snap -i -c` for actionable refs
- use `./scripts/pt text` or `./scripts/pt text --full` for reading
- re-snapshot after navigation or DOM changes
- record and verify each step as you go

## PinchTab-Specific Notes

- A navigation-triggering click can return `Error 409: unexpected page navigation ...` and still have succeeded. Verify with a fresh snapshot or text read.
- Do not assume every command returns JSON. Parse only when the output is actually JSON.
- The download endpoint returns JSON with base64 content in `data`, not a local file path.
- `PINCHTAB_TAB` should be reused for the whole lane unless a task explicitly needs a new tab.

## Group 0

For the PinchTab lane, run Group 0 from `benchmark-run/group-00.md`.
