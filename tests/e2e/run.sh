#!/bin/bash
# run.sh - Run grouped E2E scenarios for a suite directory.

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
SUITE="${1:-api}"
shift || true

require_commands() {
  local missing=0
  for cmd in "$@"; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      echo "missing required command: $cmd" >&2
      missing=1
    fi
  done
  if [ "$missing" -ne 0 ]; then
    echo "one or more required commands are unavailable in this test environment" >&2
    exit 127
  fi
}

# Parse arguments
RUN_EXTENDED=false
SCENARIO_FILTER="${E2E_SCENARIO_FILTER:-}"
EXTRA_SCENARIOS=""

for arg in "$@"; do
  case "$arg" in
    extended=true|all=true)
      RUN_EXTENDED=true
      ;;
    extended=false|all=false)
      RUN_EXTENDED=false
      ;;
    filter=*)
      SCENARIO_FILTER="${arg#filter=}"
      ;;
    extra=*)
      EXTRA_SCENARIOS="${arg#extra=}"
      ;;
    *)
      echo "unknown argument: $arg" >&2
      echo "usage: /bin/bash tests/e2e/run.sh api|cli|infra [extended=true|all=true] [filter=<substring>] [extra=<files>]" >&2
      exit 1
      ;;
  esac
done

# Map suite to directory and configuration
case "$SUITE" in
  api|api-extended)
    source "${ROOT_DIR}/helpers/api.sh"
    GROUP_DIR="${ROOT_DIR}/scenarios/api"
    SUITE_KIND="api"
    [ "$SUITE" = "api-extended" ] && RUN_EXTENDED=true
    SUITE_TITLE_BASIC="PinchTab E2E API Suite"
    SUITE_TITLE_EXTENDED="PinchTab E2E API Extended Suite"
    SUMMARY_FILE_BASIC="summary-api.txt"
    SUMMARY_FILE_EXTENDED="summary-api-extended.txt"
    REPORT_FILE_BASIC="report-api.md"
    REPORT_FILE_EXTENDED="report-api-extended.md"
    PROGRESS_FILE_BASIC="progress-api.log"
    PROGRESS_FILE_EXTENDED="progress-api-extended.log"
    REQUIRED_COMMANDS=(curl jq grep sed awk seq)
    ;;
  cli|cli-extended)
    source "${ROOT_DIR}/helpers/cli.sh"
    GROUP_DIR="${ROOT_DIR}/scenarios/cli"
    SUITE_KIND="cli"
    [ "$SUITE" = "cli-extended" ] && RUN_EXTENDED=true
    SUITE_TITLE_BASIC="PinchTab E2E CLI Suite"
    SUITE_TITLE_EXTENDED="PinchTab E2E CLI Extended Suite"
    SUMMARY_FILE_BASIC="summary-cli.txt"
    SUMMARY_FILE_EXTENDED="summary-cli-extended.txt"
    REPORT_FILE_BASIC="report-cli.md"
    REPORT_FILE_EXTENDED="report-cli-extended.md"
    PROGRESS_FILE_BASIC="progress-cli.log"
    PROGRESS_FILE_EXTENDED="progress-cli-extended.log"
    REQUIRED_COMMANDS=(pinchtab curl jq grep sed awk seq mktemp)
    ;;
  infra|infra-extended)
    source "${ROOT_DIR}/helpers/api.sh"
    GROUP_DIR="${ROOT_DIR}/scenarios/infra"
    SUITE_KIND="api"
    [ "$SUITE" = "infra-extended" ] && RUN_EXTENDED=true
    SUITE_TITLE_BASIC="PinchTab E2E Infra Suite"
    SUITE_TITLE_EXTENDED="PinchTab E2E Infra Extended Suite"
    SUMMARY_FILE_BASIC="summary-infra.txt"
    SUMMARY_FILE_EXTENDED="summary-infra-extended.txt"
    REPORT_FILE_BASIC="report-infra.md"
    REPORT_FILE_EXTENDED="report-infra-extended.md"
    PROGRESS_FILE_BASIC="progress-infra.log"
    PROGRESS_FILE_EXTENDED="progress-infra-extended.log"
    REQUIRED_COMMANDS=(curl jq grep sed awk seq)
    ;;
  plugin)
    source "${ROOT_DIR}/helpers/api.sh"
    GROUP_DIR="${ROOT_DIR}/scenarios/plugin"
    SUITE_KIND="api"
    RUN_EXTENDED=false
    SUITE_TITLE_BASIC="PinchTab E2E Plugin Suite"
    SUITE_TITLE_EXTENDED="PinchTab E2E Plugin Suite"
    SUMMARY_FILE_BASIC="summary-plugin.txt"
    SUMMARY_FILE_EXTENDED="summary-plugin.txt"
    REPORT_FILE_BASIC="report-plugin.md"
    REPORT_FILE_EXTENDED="report-plugin.md"
    PROGRESS_FILE_BASIC="progress-plugin.log"
    PROGRESS_FILE_EXTENDED="progress-plugin.log"
    REQUIRED_COMMANDS=(curl jq grep sed awk seq)
    ;;
  *)
    echo "unknown suite: $SUITE" >&2
    echo "usage: /bin/bash tests/e2e/run.sh api|cli|infra|plugin [extended=true|all=true] [filter=<substring>] [extra=<files>]" >&2
    exit 1
    ;;
esac

require_commands "${REQUIRED_COMMANDS[@]}"

# Build scenario list
SCENARIO_GROUPS=()

# Always include basic scenarios
for basic_path in "${GROUP_DIR}"/*-basic.sh; do
  if [ -f "${basic_path}" ]; then
    SCENARIO_GROUPS+=("$(basename "${basic_path}")")
  fi
done

# Include extended if requested
if [ "$RUN_EXTENDED" = "true" ]; then
  for extended_path in "${GROUP_DIR}"/*-extended.sh; do
    if [ -f "${extended_path}" ]; then
      extended_script=$(basename "${extended_path}")
      case " ${SCENARIO_GROUPS[*]} " in
        *" ${extended_script} "*) ;;
        *) SCENARIO_GROUPS+=("${extended_script}") ;;
      esac
    fi
  done
  # Include standalone scripts (no -basic or -extended suffix)
  for standalone in "${GROUP_DIR}"/*.sh; do
    if [ -f "$standalone" ]; then
      name=$(basename "$standalone")
      if [[ "$name" != *-basic.sh && "$name" != *-extended.sh ]]; then
        case " ${SCENARIO_GROUPS[*]} " in
          *" ${name} "*) ;;
          *) SCENARIO_GROUPS+=("$name") ;;
        esac
      fi
    fi
  done
fi

# Add extra touched scenarios
if [ -n "$EXTRA_SCENARIOS" ]; then
  for extra in $EXTRA_SCENARIOS; do
    name=$(basename "$extra")
    # Only add if not already in list and file exists
    if [ -f "${GROUP_DIR}/${name}" ]; then
      case " ${SCENARIO_GROUPS[*]} " in
        *" ${name} "*) ;;
        *) SCENARIO_GROUPS+=("${name}") ;;
      esac
    fi
  done
fi

# Apply filter if specified
if [ -n "$SCENARIO_FILTER" ]; then
  FILTERED_GROUPS=()
  for script_name in "${SCENARIO_GROUPS[@]}"; do
    if [[ "${script_name}" == *"${SCENARIO_FILTER}"* ]]; then
      FILTERED_GROUPS+=("${script_name}")
    fi
  done
  SCENARIO_GROUPS=("${FILTERED_GROUPS[@]}")
  if [ "${#SCENARIO_GROUPS[@]}" -eq 0 ]; then
    echo "no scenario files matched filter: ${SCENARIO_FILTER}" >&2
    exit 1
  fi
fi

# Check we have scenarios to run
if [ "${#SCENARIO_GROUPS[@]}" -eq 0 ]; then
  echo "no scenario files found in: ${GROUP_DIR}" >&2
  exit 1
fi

# Set output file names based on mode
if [ "$RUN_EXTENDED" = "true" ]; then
  SUITE_TITLE="$SUITE_TITLE_EXTENDED"
  SUMMARY_FILE="$SUMMARY_FILE_EXTENDED"
  REPORT_FILE="$REPORT_FILE_EXTENDED"
  PROGRESS_FILE="$PROGRESS_FILE_EXTENDED"
else
  SUITE_TITLE="$SUITE_TITLE_BASIC"
  SUMMARY_FILE="$SUMMARY_FILE_BASIC"
  REPORT_FILE="$REPORT_FILE_BASIC"
  PROGRESS_FILE="$PROGRESS_FILE_BASIC"
fi

export E2E_SUMMARY_TITLE="$SUITE_TITLE"
export E2E_SUMMARY_FILE="$SUMMARY_FILE"
export E2E_REPORT_FILE="$REPORT_FILE"
export E2E_PROGRESS_FILE="$PROGRESS_FILE"
export E2E_GENERATE_MARKDOWN_REPORT=1

# Print header and wait for instances
if [ "$SUITE_KIND" = "api" ]; then
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo -e "${BLUE}${SUITE_TITLE}${NC}"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "E2E_SERVER: ${E2E_SERVER}"
  echo "FIXTURES_URL: ${FIXTURES_URL}"
  if [ -n "$SCENARIO_FILTER" ]; then
    echo "FILTER: ${SCENARIO_FILTER}"
  fi
  if [ -n "${E2E_TEST_FILTER:-}" ]; then
    echo "TEST:   ${E2E_TEST_FILTER}"
  fi
  if [ -n "$EXTRA_SCENARIOS" ]; then
    echo "EXTRA: ${EXTRA_SCENARIOS}"
  fi
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""

  echo "Waiting for instances to become ready..."
  wait_for_instance_ready "${E2E_SERVER}"
  if [ "$RUN_EXTENDED" = "true" ]; then
    wait_for_instance_ready "${E2E_SECURE_SERVER}"
    if [ -n "${E2E_AUTOCLOSE_SERVER:-}" ]; then
      wait_for_instance_ready "${E2E_AUTOCLOSE_SERVER}"
    fi
    if [ -n "${E2E_MEDIUM_SERVER:-}" ]; then
      wait_for_instance_ready "${E2E_MEDIUM_SERVER}"
    fi
    if [ -n "${E2E_FULL_SERVER:-}" ]; then
      wait_for_instance_ready "${E2E_FULL_SERVER}"
    fi
    if [ -n "${E2E_LITE_SERVER:-}" ]; then
      wait_for_instance_ready "${E2E_LITE_SERVER}"
    fi
    if [ -n "${E2E_BRIDGE_URL:-}" ]; then
      wait_for_instance_ready "${E2E_BRIDGE_URL}" 60 "${E2E_BRIDGE_TOKEN:-}"
    fi
  fi
  echo ""
else
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "${SUITE_TITLE}"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo "  Server: $E2E_SERVER"
  echo "  Fixtures: $FIXTURES_URL"
  if [ -n "$SCENARIO_FILTER" ]; then
    echo "  Filter: $SCENARIO_FILTER"
  fi
  if [ -n "${E2E_TEST_FILTER:-}" ]; then
    echo "  Test:   ${E2E_TEST_FILTER}"
  fi
  if [ -n "$EXTRA_SCENARIOS" ]; then
    echo "  Extra: $EXTRA_SCENARIOS"
  fi
  echo ""

  wait_for_instance_ready "$E2E_SERVER"

  if ! command -v pinchtab &> /dev/null; then
    echo "ERROR: pinchtab CLI not found in PATH"
    exit 1
  fi

  echo ""
  if [ "$RUN_EXTENDED" = "true" ]; then
    echo "Running CLI extended tests..."
  else
    echo "Running CLI tests..."
  fi
  echo ""
fi

# When E2E_TEST_FILTER is set, source only scenario preamble + matching
# start_test...end_test blocks. Lets a single test run end-to-end with the
# scenario's setup intact, no per-helper guards needed.
TEST_FILTER="${E2E_TEST_FILTER:-}"

source_filtered_scenario() {
  local script_path="$1"
  local pattern="$2"
  local script_dir
  script_dir="$(dirname "${script_path}")"
  # The scenarios dir may be read-only (runner mounts ./:/e2e:ro), so we
  # write the filtered tempfile to /tmp. Scenarios resolve helper paths
  # via `GROUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)`, which
  # would point at /tmp from the tempfile — so we strip that line and
  # pre-set GROUP_DIR to the original scenario directory instead.
  # BusyBox mktemp (Alpine) needs the XXXXXX at the end of TEMPLATE — no
  # trailing extension. Use an explicit /tmp path so behaviour is consistent
  # across GNU and BusyBox.
  local tmp
  tmp=$(mktemp /tmp/e2e-scenario.XXXXXX)
  printf 'GROUP_DIR=%q\n' "${script_dir}" > "${tmp}"
  awk -v want="${pattern}" '
    BEGIN { preamble=1; in_test=0; capture=0; matched=0 }
    /^[[:space:]]*GROUP_DIR=.*BASH_SOURCE/ { next }
    /^[[:space:]]*start_test[[:space:]]/ {
      preamble=0
      in_test=1
      name=$0
      sub(/^[[:space:]]*start_test[[:space:]]+/, "", name)
      gsub(/^["'\'']|["'\'']$/, "", name)
      if (index(name, want) > 0) { capture=1; matched=1 } else { capture=0 }
    }
    preamble || (in_test && capture) { print }
    /^[[:space:]]*end_test[[:space:]]*$/ && in_test { in_test=0; capture=0 }
    END { exit matched ? 0 : 2 }
  ' "${script_path}" >> "${tmp}"
  local awk_status=$?
  if [ "${awk_status}" -eq 2 ]; then
    rm -f "${tmp}"
    return 2
  fi
  # shellcheck disable=SC1090
  source "${tmp}"
  rm -f "${tmp}"
  return 0
}

# Run scenarios
for script_name in "${SCENARIO_GROUPS[@]}"; do
  script_path="${GROUP_DIR}/${script_name}"
  if [ ! -f "${script_path}" ]; then
    echo "group entry not found: ${script_path}" >&2
    exit 1
  fi

  if [ -d "${RESULTS_DIR:-}" ] && [ -n "${E2E_PROGRESS_FILE:-}" ]; then
    printf '%s RUNNING %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "${script_name}" >> "${RESULTS_DIR}/${E2E_PROGRESS_FILE}"
  fi

  echo -e "${YELLOW}Running: ${script_name}${NC}"
  echo ""
  CURRENT_SCENARIO_FILE="${script_name%.sh}"
  if [ -n "${TEST_FILTER}" ]; then
    if ! source_filtered_scenario "${script_path}" "${TEST_FILTER}"; then
      echo -e "${MUTED}  no matching test in ${script_name}${NC}"
    fi
  else
    source "${script_path}"
  fi
  echo ""

  if [ -d "${RESULTS_DIR:-}" ] && [ -n "${E2E_PROGRESS_FILE:-}" ]; then
    printf '%s DONE %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "${script_name}" >> "${RESULTS_DIR}/${E2E_PROGRESS_FILE}"
  fi
done

print_summary
