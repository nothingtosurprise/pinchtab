#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BENCH_DIR="${ROOT_DIR}/tests/benchmark"
RESULTS_DIR="${BENCH_DIR}/results"

resolve_current_report() {
  local ptr="$1"
  if [[ -f "${ptr}" ]]; then
    tr -d '[:space:]' < "${ptr}"
    return 0
  fi
  return 1
}

usage() {
  cat <<'EOF'
Usage:
  ./dev benchmark baseline
  OPENAI_API_KEY=... ./dev benchmark agent [extra runner args...]
  OPENAI_API_KEY=... ./dev benchmark agent-browser [extra runner args...]
  ANTHROPIC_API_KEY=... ./dev benchmark agent [extra runner args...]
  ANTHROPIC_API_KEY=... ./dev benchmark agent-browser [extra runner args...]

Examples:
  ./dev benchmark baseline
  OPENAI_API_KEY=... ./dev benchmark agent --model gpt-5
  OPENAI_API_KEY=... ./dev benchmark agent-browser --model gpt-5
  ANTHROPIC_API_KEY=... ./dev benchmark agent --model claude-haiku-4-5-20251001
  ANTHROPIC_API_KEY=... ./dev benchmark agent-browser --model claude-haiku-4-5-20251001
  ANTHROPIC_API_KEY=... ./dev benchmark agent --provider anthropic --turn-delay-ms 3000
  ANTHROPIC_API_KEY=... ./dev benchmark agent --provider anthropic --profile common10
  ANTHROPIC_API_KEY=... ./dev benchmark agent-browser --provider anthropic --groups 0,1,2,3
EOF
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

mode="$1"
shift

cd "${BENCH_DIR}"

case "${mode}" in
  baseline)
    ./scripts/run-optimization.sh
    ./scripts/baseline.sh
    BASELINE_REPORT="$(resolve_current_report "${RESULTS_DIR}/current_baseline_report.txt")"
    ./scripts/finalize-report.sh "${BASELINE_REPORT}"
    ;;
  agent|agent-browser)
    if [[ -z "${OPENAI_API_KEY:-}" && -z "${ANTHROPIC_API_KEY:-}" ]]; then
      echo "ERROR: OPENAI_API_KEY or ANTHROPIC_API_KEY is required for benchmark ${mode}" >&2
      exit 1
    fi
    exec node --experimental-strip-types ./scripts/run-api-benchmark.ts --lane "${mode}" --finalize "$@"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    echo "ERROR: unknown benchmark mode: ${mode}" >&2
    usage
    exit 1
    ;;
esac
