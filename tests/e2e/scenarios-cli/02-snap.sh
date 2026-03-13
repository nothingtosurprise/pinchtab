#!/bin/bash
# 02-snap.sh — CLI snapshot command

source "$(dirname "$0")/common.sh"

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab snap"

pt_ok nav "${FIXTURES_URL}/index.html"
pt_ok snap
assert_output_json
assert_output_contains "nodes" "returns snapshot nodes"

end_test
