# PinchTab Benchmark

Structured benchmarks to measure AI agent performance with PinchTab and other
browser-control surfaces against the same fixture suite.

## Quick Start

```bash
cd tests/benchmark

# initialize PinchTab benchmark reports + Docker
./scripts/run-optimization.sh

# deterministic baseline verification after infrastructure/product changes
./scripts/baseline.sh
./scripts/finalize-report.sh "$(cat ./results/current_baseline_report.txt)"

# agent-browser lane
./scripts/run-agent-browser-benchmark.sh
# Then read setup-agent-browser.md and benchmark-run/index.md and run tasks with ./scripts/ab
./scripts/finalize-report.sh

# repo entrypoints (recommended)
./dev benchmark baseline
./dev benchmark pinchtab --dry-run
OPENAI_API_KEY=... ./dev benchmark pinchtab
ANTHROPIC_API_KEY=... ./dev benchmark pinchtab --profile common10
ANTHROPIC_API_KEY=... ./dev benchmark pinchtab --max-input-tokens 50000
ANTHROPIC_API_KEY=... ./dev benchmark agent-browser --groups 0,1,2,3

# direct Go runner
ANTHROPIC_API_KEY=... go run ./cmd/api-runner --lane pinchtab --finalize
ANTHROPIC_API_KEY=... go run ./cmd/api-runner --lane agent-browser --finalize
```

## MANDATORY: Docker Environment

**The benchmark MUST run against Docker.** Do not use a local pinchtab server.

Reasons:
- Reproducible: Same environment every run
- Clean state: No leftover profiles, instances, or sessions
- Latest build: Builds from current source
- Isolated: No interference from local config

If Docker build fails or is skipped, the benchmark is **INVALID**.

## Files

| File | Purpose |
|------|---------|
| `../../skills/pinchtab/SKILL.md` | PinchTab skill (same as shipped product) |
| `setup-pinchtab.md` | PinchTab lane setup and wrapper guidance |
| `setup-agent-browser.md` | agent-browser lane setup and wrapper guidance |
| `benchmark-run/index.md` | Shared benchmark task index for the agent-driven lanes |
| `benchmark-run/group-XX.md` | One markdown file per benchmark group |
| `scripts/baseline.sh` | Executable baseline lane source of truth |
| `scripts/run-optimization.sh` | Initialize PinchTab benchmark reports |
| `scripts/run-agent-browser-benchmark.sh` | Start fixtures + `agent-browser` and initialize a fresh report |
| `scripts/ab` | Docker-backed `agent-browser` wrapper with tool-call logging |
| `scripts/record-step.sh` | Record step results and tool-call counts |
| `scripts/record-run-usage.sh` | Stamp exact run-level usage into a report |
| `scripts/run-api-benchmark.ts` | Optional OpenAI/Anthropic API runner with exact run usage capture |
| `scripts/finalize-report.sh` | Generate final summary report |
| `config/pinchtab.json` | PinchTab configuration |
| `agent-browser/Dockerfile` | `agent-browser` benchmark image |
| `docker-compose.yml` | Docker environment definition |
| `results/` | Output directory for reports |

## Execution

The benchmark is designed to run in a fresh agent context:

1. Initialize the relevant benchmark lane
2. Run baseline as a direct shell verification when you need to confirm the benchmark environment still works after a change
3. Execute the natural-language tasks only for the agent and agent-browser lanes
4. Record each step's raw answer/result
5. Verify observed steps in a separate pass
6. Let the harness count browser/tool calls where possible

This measures the **real cost** of using a browser tool with an agent, including:
- Context loading overhead
- Browser/tool-call count
- Total benchmark cost

Baseline is not an agent lane. It is a deterministic shell script verification:

- executable source of truth: `scripts/baseline.sh`
- typical use: run it after PinchTab, fixtures, or benchmark-harness changes
- purpose: confirm the benchmark environment still behaves as expected before spending agent budget

For the agent lanes, the recommended flow is two-phase:

1. execution records each step as `answer`
2. verification stamps each answered step with `verify-step.sh`

Baseline runs do both phases inline inside `scripts/baseline.sh`. Agent lanes keep
execution and verification separate by default.

## Token Usage

The benchmark distinguishes between:

- **step data**: execution answers, tool calls, pass/fail/skip
- **run usage**: exact model usage reported by the provider for the full run

Step-level `--tokens` on `record-step.sh` remains only as backward-compatible
schema baggage. The preferred path for precise token accounting is run-level
usage recorded after the run completes.

When the optional provider runner is used, the report stores exact usage from
the backing API:

- `input_tokens` (uncached input after the cache breakpoint)
- `cache_creation_input_tokens`
- `cache_read_input_tokens`
- `output_tokens`
- `total_input_tokens`
- `total_tokens`

Those values live under `run_usage` in the report JSON and are summarized by
`finalize-report.sh`.

## Environment

The benchmark runs PinchTab in Docker with:

- **Port**: 9867
- **Token**: `benchmark-token`
- **Stealth**: Full (for protected sites)
- **Headless**: Yes
- **Multi-instance**: Enabled (2 instances)

## Step Recording

Every step should record the observed answer/result:

```bash
./scripts/record-step.sh --type pinchtab <group> <step> <answer|fail|skip> "answer" "notes"
```

Example:
```bash
./scripts/record-step.sh --type pinchtab 1 1 answer "Navigation completed in 1.2s" "observed output"
./scripts/record-step.sh --type pinchtab 2 3 fail "Element not found"
```

Deferred-verification example:

```bash
./scripts/record-step.sh --type pinchtab 1 1 answer \
  "Found categories Programming Languages: 12, Databases: 8" \
  "raw answer"
./scripts/verify-step.sh --type pinchtab 1 1 pass \
  "Answer satisfies the benchmark oracle"
```

Baseline automation uses the same report shape, but verifies inline:

```bash
./scripts/record-step.sh --type baseline 1 1 answer "ok" "Health endpoint returned ok"
./scripts/verify-step.sh --type baseline 1 1 pass "Health endpoint matched oracle"
```

## Reports

Reports are generated in `results/`:

- `benchmark_YYYYMMDD_HHMMSS.json` - Raw JSON data
- `benchmark_YYYYMMDD_HHMMSS_summary.md` - Human-readable summary
- `agent_browser_commands.ndjson` - `agent-browser` command log for tool-call attribution

### Example Summary

```
# PinchTab Benchmark Results

## Results
| Metric | Value |
|--------|-------|
| Steps Passed | 30 |
| Steps Failed | 2 |
| Pass Rate | 93.7% |

## Tooling
| Metric | Value |
|--------|-------|
| Tool Calls | 128 |
```

## Running Programmatically

For automated benchmarks, you can:

1. Run `scripts/baseline.sh` for deterministic shell verification
2. Treat `scripts/baseline.sh` as the executable baseline contract
3. Use the API runner only for `agent` and `agent-browser`
4. Call `scripts/record-step.sh` with results
5. Call `scripts/verify-step.sh` for answered steps
6. Run `scripts/finalize-report.sh`

### Optional Provider Runner

The API runner executes the benchmark lane through the OpenAI Responses API or
Anthropic Messages API with a persistent local shell tool. It is useful when
you want exact provider-reported usage at the end of the run.

Examples:

```bash
cd tests/benchmark

# PinchTab lane
OPENAI_API_KEY=... node --experimental-strip-types ./scripts/run-api-benchmark.ts \
  --lane agent \
  --model gpt-5 \
  --finalize

# PinchTab lane via Anthropic
ANTHROPIC_API_KEY=... node --experimental-strip-types ./scripts/run-api-benchmark.ts \
  --lane agent \
  --model claude-haiku-4-5-20251001 \
  --finalize

# agent-browser lane via Anthropic
ANTHROPIC_API_KEY=... node --experimental-strip-types ./scripts/run-api-benchmark.ts \
  --lane agent-browser \
  --model claude-haiku-4-5-20251001 \
  --finalize
```

By default the runner enables provider prompt caching when supported, so the
final report can include cached and uncached input token counts.

If you hit provider rate limits, add a delay between API turns:

```bash
ANTHROPIC_API_KEY=... ./dev benchmark agent --provider anthropic --turn-delay-ms 3000
```

The runner also honors provider `retry-after` headers automatically on `429`
responses and prints the resolved provider/model/report path before execution.
It also writes a tool-command trace to `results/agent_commands.ndjson` and
stops early if too many turns pass without recording any benchmark progress.

You can also run only a subset of benchmark groups:

```bash
# Representative ~10% subset of groups
ANTHROPIC_API_KEY=... ./dev benchmark agent --provider anthropic --profile common10

# Exact group selection
ANTHROPIC_API_KEY=... ./dev benchmark agent-browser --provider anthropic --groups 0,1,2,3
```

Current subset preset:

- `common10` -> groups `0,1,2,3` (setup, reading/extraction, search, form)

## Reproducibility

For consistent results:

1. Always start with a fresh Docker-backed benchmark lane
2. Use the same model/temperature for comparisons
3. Run benchmarks at similar times (site load varies)
4. Record exact PinchTab version from `/version` endpoint
5. Clear browser state between full benchmark runs
