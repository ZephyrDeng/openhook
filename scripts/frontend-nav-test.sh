#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if rg -n "label: '过滤器'|label: '去重规则'|currentPage === 'filters'|currentPage === 'dedup'|from './pages/Filters.svelte'|from './pages/DedupRules.svelte'" \
  "${ROOT_DIR}/frontend/src/App.svelte" \
  "${ROOT_DIR}/frontend/src/components/Sidebar.svelte"; then
  echo "hidden rule-storage pages are still reachable from frontend navigation" >&2
  exit 1
fi

echo "FRONTEND_NAV_TEST_OK"
