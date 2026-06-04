#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_SITE_PATTERN="commute-planner[.]site"

require_match() {
  local pattern="$1"
  local file="$2"
  if ! grep -Eq "$pattern" "${ROOT_DIR}/${file}"; then
    echo "missing pattern ${pattern} in ${file}" >&2
    exit 1
  fi
}

reject_match() {
  local pattern="$1"
  local file="$2"
  if grep -Eq "$pattern" "${ROOT_DIR}/${file}"; then
    echo "unexpected pattern ${pattern} in ${file}" >&2
    exit 1
  fi
}

require_match "from './pages/Guide.svelte'" frontend/src/App.svelte
require_match "id: 'guide'.*label: '指南'" frontend/src/App.svelte
require_match "id: 'guide'.*label: '使用指南'" frontend/src/components/Sidebar.svelte
require_match "currentPage === 'guide'" frontend/src/App.svelte
require_match "快速上手" frontend/src/pages/Guide.svelte
require_match "创建消息模板" frontend/src/pages/Guide.svelte
require_match "创建路由" frontend/src/pages/Guide.svelte
require_match "POST /webhook/routes/\\{routeId\\}" frontend/src/pages/Guide.svelte
require_match "对外 Webhook 地址" frontend/src/pages/Guide.svelte
require_match "Content-Type: application/json" frontend/src/pages/Guide.svelte
require_match "目标 Webhook 地址.*下游" frontend/src/pages/Guide.svelte
require_match "\\{\\{data.title\\}\\}" frontend/src/pages/Guide.svelte
require_match "envelope" frontend/src/pages/Guide.svelte
require_match "raw" frontend/src/pages/Guide.svelte
require_match "对外 Webhook 地址" frontend/src/pages/RouteEditor.svelte
require_match "复制地址" frontend/src/pages/RouteEditor.svelte
require_match "/webhook/routes/\\$\\{route.routeId\\}" frontend/src/pages/RouteEditor.svelte
reject_match "https://${TEST_SITE_PATTERN}/api/routes/\\{routeId\\}/deliver" frontend/src/pages/Guide.svelte
reject_match "window.location.origin\\}/api/routes" frontend/src/pages/RouteEditor.svelte
require_match "login-hero" frontend/src/App.svelte
require_match "login-panel" frontend/src/App.svelte
require_match "\\.login-hero" frontend/src/app.css
require_match "\\.login-panel" frontend/src/app.css
require_match "meta\\.get" frontend/src/pages/Settings.svelte
reject_match ">[[:space:]]*/login/github[[:space:]]*<" frontend/src/App.svelte
reject_match "v0\\.1\\.0" frontend/src/pages/Settings.svelte

if grep -RE "${TEST_SITE_PATTERN}" "${ROOT_DIR}/README.md" "${ROOT_DIR}/scripts" "${ROOT_DIR}/frontend/src" >/tmp/openhook-domain-hits.txt; then
  cat /tmp/openhook-domain-hits.txt >&2
  rm -f /tmp/openhook-domain-hits.txt
  echo "test site domain must stay out of source and docs" >&2
  exit 1
fi
rm -f /tmp/openhook-domain-hits.txt

echo "FRONTEND_CONTENT_TEST_OK"
