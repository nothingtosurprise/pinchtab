#!/bin/bash
# 07-screenshot.sh — CLI screenshot command

source "$(dirname "$0")/common.sh"

start_test "pinchtab screenshot"
pt_ok nav "${FIXTURES_URL}/index.html"
pt_ok screenshot -o /tmp/e2e-screenshot-test.jpg
if [ -f /tmp/e2e-screenshot-test.jpg ]; then
  echo -e "  ${GREEN}✓${NC} screenshot file created"
  ((ASSERTIONS_PASSED++)) || true
  rm -f /tmp/e2e-screenshot-test.jpg
else
  echo -e "  ${RED}✗${NC} screenshot file not created"
  ((ASSERTIONS_FAILED++)) || true
fi
end_test
