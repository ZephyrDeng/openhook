#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

require_match() {
  local pattern="$1"
  local file="$2"
  if ! grep -Eq "$pattern" "${ROOT_DIR}/${file}"; then
    echo "missing pattern ${pattern} in ${file}" >&2
    exit 1
  fi
}

require_match 'class="app-shell"' frontend/src/App.svelte
require_match 'class="desktop-sidebar"' frontend/src/App.svelte
require_match 'class="mobile-bottom-nav"' frontend/src/App.svelte
require_match '\.page-shell' frontend/src/app.css
require_match 'class="page-header' frontend/src/pages/Templates.svelte
require_match 'class="mobile-card-list' frontend/src/pages/Templates.svelte
require_match 'class="mobile-card-list' frontend/src/pages/Routes.svelte
require_match 'class="mobile-card-list' frontend/src/pages/Deliveries.svelte
require_match 'class="editor-split' frontend/src/pages/TemplateEditor.svelte
require_match 'class="mobile-sticky-actions' frontend/src/pages/TemplateEditor.svelte
require_match 'class="mobile-sticky-actions' frontend/src/pages/RouteEditor.svelte
require_match '@media[[:space:]]*\(max-width:[[:space:]]*767px\)' frontend/src/app.css
require_match 'min-height:[[:space:]]*44px' frontend/src/app.css

echo "MOBILE_LAYOUT_TEST_OK"
