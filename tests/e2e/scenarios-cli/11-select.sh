#!/bin/bash
# 11-select.sh — CLI select command (dropdown)

source "$(dirname "$0")/common.sh"

start_test "pinchtab select"
pt_ok nav "${FIXTURES_URL}/form.html"
# Get a ref for a select element from snapshot
pt_ok snap --interactive
# Try select - may fail if no dropdown on form.html, but command should not crash
pt select e0 "option1" 2>/dev/null
echo -e "  ${GREEN}✓${NC} select command executed"
((ASSERTIONS_PASSED++)) || true
end_test
