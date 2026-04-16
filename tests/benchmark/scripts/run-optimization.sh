#!/bin/bash
#
# PinchTab Benchmark Optimization Loop
# Runs both benchmarks, analyzes differences, proposes improvements
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="${SCRIPT_DIR}/.."
RESULTS_DIR="${SCRIPT_DIR}/../results"
mkdir -p "${RESULTS_DIR}"
LOG_FILE="${RESULTS_DIR}/optimization_log.md"
CURRENT_BASELINE_PTR="${RESULTS_DIR}/current_baseline_report.txt"
CURRENT_AGENT_PTR="${RESULTS_DIR}/current_agent_report.txt"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RUN_NUMBER=$(grep -c "^## Run #" "${LOG_FILE}" 2>/dev/null || echo 0)
RUN_NUMBER=$((RUN_NUMBER + 1))

echo "=== PinchTab Optimization Run #${RUN_NUMBER} ==="
echo "Timestamp: ${TIMESTAMP}"

# Ensure Docker is running
cd "${BENCH_DIR}"
if ! docker compose ps 2>/dev/null | grep -q "running"; then
    echo "Starting Docker..."
    docker compose up -d --build
    sleep 15
fi

# Verify PinchTab is healthy
if ! curl -sf -H "Authorization: Bearer benchmark-token" http://localhost:9867/health > /dev/null; then
    echo "ERROR: PinchTab not responding, restarting..."
    docker compose down
    docker compose up -d --build
    sleep 15
fi

# Initialize reports
BASELINE_REPORT="${RESULTS_DIR}/baseline_${TIMESTAMP}.json"
AGENT_REPORT="${RESULTS_DIR}/agent_benchmark_${TIMESTAMP}.json"

cat > "${BASELINE_REPORT}" << EOF
{
  "benchmark": {
    "type": "baseline",
    "run_number": ${RUN_NUMBER},
    "timestamp": "${TIMESTAMP}",
    "model": "${BENCHMARK_MODEL:-baseline}",
    "runner": "${BENCHMARK_RUNNER:-manual}"
  },
  "totals": {
    "input_tokens": 0,
    "output_tokens": 0,
    "total_tokens": 0,
    "estimated_cost_usd": 0,
    "tool_calls": 0,
    "steps_passed": 0,
    "steps_failed": 0,
    "steps_skipped": 0,
    "steps_answered": 0,
    "steps_verified_passed": 0,
    "steps_verified_failed": 0,
    "steps_verified_skipped": 0,
    "steps_pending_verification": 0
  },
  "run_usage": {
    "source": "none",
    "provider": "",
    "request_count": 0,
    "input_tokens": 0,
    "output_tokens": 0,
    "cache_creation_input_tokens": 0,
    "cache_read_input_tokens": 0,
    "total_input_tokens": 0,
    "total_tokens": 0
  },
  "steps": []
}
EOF

cat > "${AGENT_REPORT}" << EOF
{
  "benchmark": {
    "type": "agent",
    "run_number": ${RUN_NUMBER},
    "timestamp": "${TIMESTAMP}",
    "model": "${BENCHMARK_MODEL:-unknown}",
    "runner": "${BENCHMARK_RUNNER:-manual}"
  },
  "totals": {
    "input_tokens": 0,
    "output_tokens": 0,
    "total_tokens": 0,
    "estimated_cost_usd": 0,
    "tool_calls": 0,
    "steps_passed": 0,
    "steps_failed": 0,
    "steps_skipped": 0,
    "steps_answered": 0,
    "steps_verified_passed": 0,
    "steps_verified_failed": 0,
    "steps_verified_skipped": 0,
    "steps_pending_verification": 0
  },
  "run_usage": {
    "source": "none",
    "provider": "",
    "request_count": 0,
    "input_tokens": 0,
    "output_tokens": 0,
    "cache_creation_input_tokens": 0,
    "cache_read_input_tokens": 0,
    "total_input_tokens": 0,
    "total_tokens": 0
  },
  "steps": []
}
EOF

printf '%s\n' "${BASELINE_REPORT}" > "${CURRENT_BASELINE_PTR}"
printf '%s\n' "${AGENT_REPORT}" > "${CURRENT_AGENT_PTR}"

# Clear previous agent command trace
rm -f "${RESULTS_DIR}/agent_commands.ndjson"

echo "Reports initialized:"
echo "  Baseline: ${BASELINE_REPORT}"
echo "  Agent: ${AGENT_REPORT}"
echo ""
echo "Ready for benchmark execution."
echo "Timestamp for this run: ${TIMESTAMP}"
echo "Run number: ${RUN_NUMBER}"
