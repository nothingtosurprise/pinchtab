#!/bin/bash
# 13-pdf.sh — CLI PDF export command

source "$(dirname "$0")/common.sh"

start_test "pinchtab pdf"
pt_ok nav "${FIXTURES_URL}/index.html"
pt_ok pdf -o /tmp/e2e-pdf-test.pdf
if [ -f /tmp/e2e-pdf-test.pdf ]; then
  echo -e "  ${GREEN}✓${NC} PDF file created"
  ((ASSERTIONS_PASSED++)) || true
  rm -f /tmp/e2e-pdf-test.pdf
else
  echo -e "  ${RED}✗${NC} PDF file not created"
  ((ASSERTIONS_FAILED++)) || true
fi
end_test
