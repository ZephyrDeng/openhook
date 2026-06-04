#!/usr/bin/env bash
set -euo pipefail

TOKEN_URL="${QQ_TOKEN_URL:-https://bots.qq.com/app/getAppAccessToken}"
if [[ -n "${QQ_API_BASE:-}" ]]; then
  API_BASE="${QQ_API_BASE}"
elif [[ "${QQ_SANDBOX:-0}" == "1" ]]; then
  API_BASE="https://sandbox.api.sgroup.qq.com"
else
  API_BASE="https://api.sgroup.qq.com"
fi
MESSAGE="${QQ_MESSAGE:-OpenHook QQ C2C smoke}"
MSG_SEQ="${QQ_MSG_SEQ:-$(date +%s)}"

usage() {
  cat >&2 <<'EOF'
usage: scripts/qq-c2c-smoke.sh

Environment:
  QQ_APP_ID                         QQ bot AppID
  QQ_APP_SECRET                     QQ bot AppSecret
  QQ_OPENID                         recipient C2C openid from QQ bot events
  QQ_MESSAGE                        default: OpenHook QQ C2C smoke
  QQ_MSG_SEQ                        default: current unix timestamp
  QQ_TOKEN_URL                      default: https://bots.qq.com/app/getAppAccessToken
  QQ_API_BASE                       default: https://api.sgroup.qq.com
  QQ_SANDBOX=1                      use https://sandbox.api.sgroup.qq.com when QQ_API_BASE is unset
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

json_summary() {
  local body="$1"
  local code message trace_id
  code="$(printf '%s' "${body}" | jq -r '.err_code // .code // .errcode // .error // empty' 2>/dev/null || true)"
  message="$(printf '%s' "${body}" | jq -r '.message // .errmsg // .error_description // empty' 2>/dev/null || true)"
  trace_id="$(printf '%s' "${body}" | jq -r '.trace_id // empty' 2>/dev/null || true)"
  if [[ -n "${code}" ]]; then
    printf ' code=%s' "${code}" >&2
  fi
  if [[ -n "${message}" ]]; then
    printf ' message=%s' "${message:0:200}" >&2
  fi
  if [[ -n "${trace_id}" ]]; then
    printf ' traceId=%s' "${trace_id}" >&2
  fi
}

require_cmd curl
require_cmd jq
require_env QQ_APP_ID
require_env QQ_APP_SECRET
require_env QQ_OPENID

if [[ ! "${MSG_SEQ}" =~ ^[0-9]+$ ]]; then
  echo "QQ_MSG_SEQ must be a non-negative integer" >&2
  exit 1
fi

TOKEN_BODY="$(mktemp "${TMPDIR:-/tmp}/openhook-qq-c2c-token.XXXXXX")"
DELIVERY_BODY="$(mktemp "${TMPDIR:-/tmp}/openhook-qq-c2c-delivery.XXXXXX")"
cleanup() {
  rm -f "${TOKEN_BODY}" "${DELIVERY_BODY}"
}
trap cleanup EXIT

token_request="$(jq -n --arg appId "${QQ_APP_ID}" --arg clientSecret "${QQ_APP_SECRET}" '{appId:$appId, clientSecret:$clientSecret}')"
token_status="$(curl -sS -o "${TOKEN_BODY}" -w '%{http_code}' \
  -X POST "${TOKEN_URL}" \
  -H "Content-Type: application/json" \
  -d "${token_request}")"
token_response="$(cat "${TOKEN_BODY}")"
access_token="$(printf '%s' "${token_response}" | jq -r '.access_token // empty' 2>/dev/null || true)"
if [[ "${token_status}" != "200" || -z "${access_token}" ]]; then
  printf 'QQ_TOKEN_FAIL httpStatus=%s' "${token_status}" >&2
  json_summary "${token_response}"
  printf '\n' >&2
  exit 1
fi

delivery_body="$(jq -n --arg content "${MESSAGE}" --argjson msgSeq "${MSG_SEQ}" '{content:$content, msg_type:0, msg_seq:$msgSeq}')"
delivery_status="$(curl -sS -o "${DELIVERY_BODY}" -w '%{http_code}' \
  -X POST "${API_BASE%/}/v2/users/${QQ_OPENID}/messages" \
  -H "Content-Type: application/json" \
  -H "Authorization: QQBot ${access_token}" \
  -H "X-Union-Appid: ${QQ_APP_ID}" \
  -d "${delivery_body}")"
delivery_response="$(cat "${DELIVERY_BODY}")"

if [[ "${delivery_status}" =~ ^2[0-9][0-9]$ ]]; then
  message_id="$(printf '%s' "${delivery_response}" | jq -r '.id // .message.id // .data.id // empty' 2>/dev/null || true)"
  if [[ -z "${message_id}" ]]; then
    message_id="unknown"
  fi
  printf 'QQ_C2C_OK statusCode=%s messageId=%s\n' "${delivery_status}" "${message_id}"
  exit 0
fi

printf 'QQ_C2C_FAIL httpStatus=%s' "${delivery_status}" >&2
json_summary "${delivery_response}"
printf '\n' >&2
exit 1
