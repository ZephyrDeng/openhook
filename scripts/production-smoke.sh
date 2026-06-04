#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_BASE="${OPENHOOK_API_BASE:-https://commute-planner.site}"

usage() {
  cat >&2 <<'EOF'
usage: scripts/production-smoke.sh

Environment:
  OPENHOOK_API_BASE                 default: https://commute-planner.site
  OPENHOOK_REQUIRE_GITHUB=1         require /api/auth/me githubEnabled=true

Existing route IDs:
  OPENHOOK_WECOM_ROUTE_ID
  OPENHOOK_TELEGRAM_ROUTE_ID
  OPENHOOK_QQ_ROUTE_ID

Provider target env for creating temporary routes:
  WECOM_WEBHOOK_URL
  TELEGRAM_WEBHOOK_URL
  TELEGRAM_CHAT_ID
  QQ_WEBHOOK_URL

QQ official bot token smoke:
  QQ_APP_ID
  QQ_APP_SECRET
  QQ_OPENID                         optional, enables official C2C message smoke
  QQ_MESSAGE                        optional C2C message text
  QQ_MSG_SEQ                        optional C2C message sequence
  QQ_TOKEN_URL                      default: https://bots.qq.com/app/getAppAccessToken
  QQ_API_BASE                       default: https://api.sgroup.qq.com
  QQ_SANDBOX=1                      use sandbox API when QQ_API_BASE is unset

Optional:
  OPENHOOK_ADMIN_TOKEN              used by provider-smoke when creating routes
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "$1 is required" >&2
    exit 1
  }
}

require_cmd curl
require_cmd jq

log() {
  printf '[openhook production smoke] %s\n' "$*"
}

health_payload="$(curl -fsS "${API_BASE}/health")"
health_status="$(printf '%s' "${health_payload}" | jq -r '.data.status')"
if [[ "${health_status}" != "ok" ]]; then
  echo "health check failed: ${health_payload}" >&2
  exit 1
fi
log "HEALTH_OK"

auth_payload="$(curl -fsS "${API_BASE}/api/auth/me")"
github_enabled="$(printf '%s' "${auth_payload}" | jq -r '.data.githubEnabled')"
auth_required="$(printf '%s' "${auth_payload}" | jq -r '.data.authRequired')"
if [[ "${OPENHOOK_REQUIRE_GITHUB:-0}" == "1" && "${github_enabled}" != "true" ]]; then
  echo "github oauth required but githubEnabled=${github_enabled}" >&2
  exit 1
fi
log "AUTH_OK githubEnabled=${github_enabled} authRequired=${auth_required}"

run_provider() {
  local provider="$1"
  local route_id="$2"
  if [[ -n "${route_id}" ]]; then
    OPENHOOK_API_BASE="${API_BASE}" OPENHOOK_ROUTE_ID="${route_id}" "${ROOT_DIR}/scripts/provider-smoke.sh" "${provider}"
  else
    OPENHOOK_API_BASE="${API_BASE}" "${ROOT_DIR}/scripts/provider-smoke.sh" "${provider}"
  fi
}

if [[ -n "${OPENHOOK_WECOM_ROUTE_ID:-}" || -n "${WECOM_WEBHOOK_URL:-}" ]]; then
  run_provider "wecom" "${OPENHOOK_WECOM_ROUTE_ID:-}"
  log "WECOM_OK"
else
  log "WECOM_SKIP missing OPENHOOK_WECOM_ROUTE_ID or WECOM_WEBHOOK_URL"
fi

if [[ -n "${OPENHOOK_TELEGRAM_ROUTE_ID:-}" || -n "${TELEGRAM_WEBHOOK_URL:-}" ]]; then
  if [[ -z "${TELEGRAM_CHAT_ID:-}" ]]; then
    log "TELEGRAM_SKIP missing TELEGRAM_CHAT_ID"
  else
    run_provider "telegram" "${OPENHOOK_TELEGRAM_ROUTE_ID:-}"
    log "TELEGRAM_OK"
  fi
else
  log "TELEGRAM_SKIP missing OPENHOOK_TELEGRAM_ROUTE_ID or TELEGRAM_WEBHOOK_URL"
fi

if [[ -n "${QQ_APP_ID:-}" || -n "${QQ_APP_SECRET:-}" ]]; then
  if [[ -z "${QQ_APP_ID:-}" || -z "${QQ_APP_SECRET:-}" ]]; then
    echo "QQ_APP_ID and QQ_APP_SECRET must be provided together" >&2
    exit 1
  fi
  qq_token_output="$("${ROOT_DIR}/scripts/qq-token-smoke.sh")"
  while IFS= read -r line; do
    log "${line}"
  done <<<"${qq_token_output}"
  if [[ -n "${QQ_OPENID:-}" ]]; then
    qq_c2c_output="$("${ROOT_DIR}/scripts/qq-c2c-smoke.sh")"
    while IFS= read -r line; do
      log "${line}"
    done <<<"${qq_c2c_output}"
  else
    log "QQ_C2C_SKIP missing QQ_OPENID"
  fi
else
  log "QQ_TOKEN_SKIP missing QQ_APP_ID or QQ_APP_SECRET"
fi

if [[ -n "${OPENHOOK_QQ_ROUTE_ID:-}" || -n "${QQ_WEBHOOK_URL:-}" ]]; then
  run_provider "qq" "${OPENHOOK_QQ_ROUTE_ID:-}"
  log "QQ_DELIVERY_OK"
else
  log "QQ_DELIVERY_SKIP missing OPENHOOK_QQ_ROUTE_ID or QQ_WEBHOOK_URL"
fi

log "PRODUCTION_SMOKE_OK"
