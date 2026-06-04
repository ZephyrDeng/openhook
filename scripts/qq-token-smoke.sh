#!/usr/bin/env bash
set -euo pipefail

TOKEN_URL="${QQ_TOKEN_URL:-https://bots.qq.com/app/getAppAccessToken}"

usage() {
  cat >&2 <<'EOF'
usage: scripts/qq-token-smoke.sh

Environment:
  QQ_APP_ID                         QQ bot AppID
  QQ_APP_SECRET                     QQ bot AppSecret
  QQ_TOKEN_URL                      default: https://bots.qq.com/app/getAppAccessToken
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

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "${name} is required" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd jq
require_env QQ_APP_ID
require_env QQ_APP_SECRET

TMP_BODY="$(mktemp "${TMPDIR:-/tmp}/openhook-qq-token.XXXXXX")"
cleanup() {
  rm -f "${TMP_BODY}"
}
trap cleanup EXIT

request_body="$(jq -n --arg appId "${QQ_APP_ID}" --arg clientSecret "${QQ_APP_SECRET}" '{appId:$appId, clientSecret:$clientSecret}')"
http_status="$(curl -sS -o "${TMP_BODY}" -w '%{http_code}' \
  -X POST "${TOKEN_URL}" \
  -H "Content-Type: application/json" \
  -d "${request_body}")"

response_body="$(cat "${TMP_BODY}")"
access_token="$(printf '%s' "${response_body}" | jq -r '.access_token // empty' 2>/dev/null || true)"
expires_in="$(printf '%s' "${response_body}" | jq -r '.expires_in // empty' 2>/dev/null || true)"

if [[ "${http_status}" == "200" && -n "${access_token}" && -n "${expires_in}" ]]; then
  printf 'QQ_TOKEN_OK expiresIn=%s\n' "${expires_in}"
  exit 0
fi

error_code="$(printf '%s' "${response_body}" | jq -r '.code // .errcode // .error // empty' 2>/dev/null || true)"
error_message="$(printf '%s' "${response_body}" | jq -r '.message // .errmsg // .error_description // empty' 2>/dev/null || true)"
printf 'QQ_TOKEN_FAIL httpStatus=%s' "${http_status}" >&2
if [[ -n "${error_code}" ]]; then
  printf ' code=%s' "${error_code}" >&2
fi
if [[ -n "${error_message}" ]]; then
  printf ' message=%s' "${error_message:0:200}" >&2
fi
printf '\n' >&2
exit 1
