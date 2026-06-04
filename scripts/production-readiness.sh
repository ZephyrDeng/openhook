#!/usr/bin/env bash
set -euo pipefail

API_BASE="${OPENHOOK_API_BASE:?OPENHOOK_API_BASE is required}"
DEPLOY_HOST="${OPENHOOK_DEPLOY_HOST:-openhook}"
REMOTE_SERVICE="${OPENHOOK_REMOTE_SERVICE:-openhook}"
REMOTE_ENV_FILE="${OPENHOOK_REMOTE_ENV_FILE:-/etc/openhook/openhook.env}"
SSH_BIN="${OPENHOOK_SSH_BIN:-ssh}"
EXPECT_WECOM_ROUTE_ID="${OPENHOOK_WECOM_ROUTE_ID:-rt_bnU7i41mLNCStFbYYsy4bg}"
MIN_GITHUB_USERS="${OPENHOOK_MIN_GITHUB_USERS:-1}"
MIN_USER_TEMPLATES="${OPENHOOK_MIN_USER_TEMPLATES:-3}"
MIN_STARTER_TEMPLATES="${OPENHOOK_MIN_STARTER_TEMPLATES:-3}"

usage() {
  cat >&2 <<'EOF'
usage: scripts/production-readiness.sh

Environment:
  OPENHOOK_API_BASE                 production OpenHook base URL
  OPENHOOK_DEPLOY_HOST              default: openhook
  OPENHOOK_REMOTE_SERVICE           default: openhook
  OPENHOOK_REMOTE_ENV_FILE          default: /etc/openhook/openhook.env
  OPENHOOK_WECOM_ROUTE_ID           expected production WeCom route
  OPENHOOK_MIN_GITHUB_USERS         default: 1
  OPENHOOK_MIN_USER_TEMPLATES       default: 3
  OPENHOOK_MIN_STARTER_TEMPLATES    default: 3
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

log() {
  printf '[openhook readiness] %s\n' "$*"
}

require_cmd curl
require_cmd jq
require_cmd "${SSH_BIN}"

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
if [[ "${github_enabled}" != "true" || "${auth_required}" != "true" ]]; then
  echo "github auth readiness failed: githubEnabled=${github_enabled} authRequired=${auth_required}" >&2
  exit 1
fi
log "AUTH_OK githubEnabled=${github_enabled} authRequired=${auth_required}"

service_state="$("${SSH_BIN}" "${DEPLOY_HOST}" "systemctl is-active '${REMOTE_SERVICE}'" | tr -d '\r' | tail -n 1)"
if [[ "${service_state}" != "active" ]]; then
  echo "service readiness failed: ${REMOTE_SERVICE} state=${service_state}" >&2
  exit 1
fi
log "SERVICE_OK service=${REMOTE_SERVICE}"

"${SSH_BIN}" "${DEPLOY_HOST}" "sudo bash -s -- '${REMOTE_ENV_FILE}' '${MIN_GITHUB_USERS}' '${MIN_USER_TEMPLATES}' '${MIN_STARTER_TEMPLATES}' '${EXPECT_WECOM_ROUTE_ID}'" <<'REMOTE'
set -euo pipefail

env_file="$1"
min_github_users="$2"
min_user_templates="$3"
min_starter_templates="$4"
expect_wecom_route_id="$5"

log() {
  printf '[openhook readiness] %s\n' "$*"
}

if ! command -v sqlite3 >/dev/null 2>&1 && ! command -v python3 >/dev/null 2>&1; then
  echo "sqlite3 or python3 is required on remote host" >&2
  exit 1
fi

db_path="$(
  sudo awk -F= '
    $1 == "OPENHOOK_DB" {
      value = substr($0, index($0, "=") + 1)
      gsub(/^"/, "", value)
      gsub(/"$/, "", value)
      gsub(/^'\''/, "", value)
      gsub(/'\''$/, "", value)
      print value
    }
  ' "${env_file}"
)"
if [[ -z "${db_path}" ]]; then
  db_path="/var/lib/openhook/openhook.db"
fi

sudo test -r "${db_path}"
log "DB_OK path=${db_path}"

sql_scalar() {
  local sql="$1"
  if command -v sqlite3 >/dev/null 2>&1; then
    sudo sqlite3 "${db_path}" "${sql}"
    return
  fi
  sudo DB_PATH="${db_path}" SQL_QUERY="${sql}" python3 - <<'PY'
import os
import sqlite3

conn = sqlite3.connect(os.environ["DB_PATH"])
try:
    row = conn.execute(os.environ["SQL_QUERY"]).fetchone()
    print(row[0] if row else "")
finally:
    conn.close()
PY
}

github_users="$(sql_scalar "SELECT COUNT(1) FROM users WHERE provider = 'github';")"
if (( github_users < min_github_users )); then
  echo "github user readiness failed: count=${github_users} min=${min_github_users}" >&2
  exit 1
fi
log "GITHUB_USERS_OK count=${github_users}"

user_templates="$(sql_scalar "SELECT COUNT(1) FROM templates WHERE current_owner IN (SELECT user_id FROM users WHERE provider = 'github');")"
if (( user_templates < min_user_templates )); then
  echo "user template readiness failed: count=${user_templates} min=${min_user_templates}" >&2
  exit 1
fi
log "USER_TEMPLATES_OK count=${user_templates}"

starter_templates="$(sql_scalar "SELECT COUNT(DISTINCT template_name) FROM templates WHERE current_owner IN (SELECT user_id FROM users WHERE provider = 'github') AND template_name IN ('企微-机器人 Markdown', 'Telegram-sendMessage', 'QQ-Webhook 文本');")"
if (( starter_templates < min_starter_templates )); then
  echo "starter template readiness failed: count=${starter_templates} min=${min_starter_templates}" >&2
  exit 1
fi
log "STARTER_TEMPLATES_OK count=${starter_templates}"

wecom_route="$(sql_scalar "SELECT COUNT(1) FROM routes WHERE route_id = '${expect_wecom_route_id}' AND enabled = 1;")"
if (( wecom_route < 1 )); then
  echo "wecom route readiness failed: routeId=${expect_wecom_route_id}" >&2
  exit 1
fi
log "WECOM_ROUTE_OK routeId=${expect_wecom_route_id}"
REMOTE

log "PRODUCTION_READINESS_OK"
