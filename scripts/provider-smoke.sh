#!/usr/bin/env bash
set -euo pipefail

API_BASE="${OPENHOOK_API_BASE:-https://commute-planner.site}"
PROVIDER="${1:-}"

usage() {
  cat >&2 <<'EOF'
usage: scripts/provider-smoke.sh <wecom|wecom-text|telegram|telegram-text|qq>

Environment:
  OPENHOOK_API_BASE                 default: https://commute-planner.site
  OPENHOOK_ROUTE_ID                 existing route id to deliver through
  OPENHOOK_ADMIN_TOKEN              optional, used when creating a route

Provider target env:
  WECOM_WEBHOOK_URL                 for provider wecom or wecom-text
  TELEGRAM_WEBHOOK_URL              for provider telegram or telegram-text, e.g. https://api.telegram.org/bot<TOKEN>/sendMessage
  TELEGRAM_CHAT_ID                  required for provider telegram or telegram-text
  QQ_WEBHOOK_URL                    for provider qq

Examples:
  WECOM_WEBHOOK_URL=https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=... scripts/provider-smoke.sh wecom
  WECOM_WEBHOOK_URL=https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=... scripts/provider-smoke.sh wecom-text
  TELEGRAM_WEBHOOK_URL=https://api.telegram.org/bot.../sendMessage TELEGRAM_CHAT_ID=123 scripts/provider-smoke.sh telegram
  TELEGRAM_WEBHOOK_URL=https://api.telegram.org/bot.../sendMessage TELEGRAM_CHAT_ID=123 scripts/provider-smoke.sh telegram-text
  QQ_WEBHOOK_URL=https://example.com/qq-webhook scripts/provider-smoke.sh qq
EOF
}

if [[ -z "${PROVIDER}" || "${PROVIDER}" == "-h" || "${PROVIDER}" == "--help" ]]; then
  usage
  exit 1
fi

case "${PROVIDER}" in
  wecom)
    TEMPLATE_FILE="examples/providers/wecom-markdown-template.json"
    TARGET_URL="${WECOM_WEBHOOK_URL:-}"
    PAYLOAD='{"title":"OpenHook WeCom smoke","severity":"info","service":"openhook","environment":"prod","time":"2026-06-04 00:00:00","description":"provider smoke"}'
    ;;
  wecom-text)
    TEMPLATE_FILE="examples/providers/wecom-text-template.json"
    TARGET_URL="${WECOM_WEBHOOK_URL:-}"
    PAYLOAD='{"title":"OpenHook WeCom text smoke","severity":"info","service":"openhook","environment":"prod","description":"provider smoke","mentionedList":[],"mentionedMobileList":[]}'
    ;;
  telegram)
    TEMPLATE_FILE="examples/providers/telegram-send-message-template.json"
    TARGET_URL="${TELEGRAM_WEBHOOK_URL:-}"
    if [[ -z "${TELEGRAM_CHAT_ID:-}" ]]; then
      echo "TELEGRAM_CHAT_ID is required" >&2
      exit 1
    fi
    PAYLOAD="{\"chatId\":\"${TELEGRAM_CHAT_ID}\",\"title\":\"OpenHook Telegram smoke\",\"severity\":\"info\",\"service\":\"openhook\",\"environment\":\"prod\",\"description\":\"provider smoke\"}"
    ;;
  telegram-text)
    TEMPLATE_FILE="examples/providers/telegram-text-template.json"
    TARGET_URL="${TELEGRAM_WEBHOOK_URL:-}"
    if [[ -z "${TELEGRAM_CHAT_ID:-}" ]]; then
      echo "TELEGRAM_CHAT_ID is required" >&2
      exit 1
    fi
    PAYLOAD="{\"chatId\":\"${TELEGRAM_CHAT_ID}\",\"title\":\"OpenHook Telegram text smoke\",\"severity\":\"info\",\"service\":\"openhook\",\"environment\":\"prod\",\"description\":\"provider smoke\"}"
    ;;
  qq)
    TEMPLATE_FILE="examples/providers/qq-webhook-text-template.json"
    TARGET_URL="${QQ_WEBHOOK_URL:-}"
    PAYLOAD='{"title":"OpenHook QQ smoke","severity":"info","service":"openhook","environment":"prod","description":"provider smoke"}'
    ;;
  *)
    usage
    exit 1
    ;;
esac

if [[ ! -f "${TEMPLATE_FILE}" ]]; then
  echo "template file missing: ${TEMPLATE_FILE}" >&2
  exit 1
fi

api() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local args=(-fsS -X "${method}" "${API_BASE}${path}" -H "Content-Type: application/json")
  if [[ -n "${OPENHOOK_ADMIN_TOKEN:-}" ]]; then
    args+=(-H "X-OpenHook-Admin-Token: ${OPENHOOK_ADMIN_TOKEN}")
  fi
  if [[ -n "${body}" ]]; then
    args+=(-d "${body}")
  fi
  curl "${args[@]}"
}

json_field() {
  jq -r "$1"
}

ROUTE_ID="${OPENHOOK_ROUTE_ID:-}"
if [[ -z "${ROUTE_ID}" ]]; then
  if [[ -z "${TARGET_URL}" ]]; then
    echo "OPENHOOK_ROUTE_ID or provider webhook URL is required" >&2
    exit 1
  fi
  template_resp="$(api POST /api/templates "$(cat "${TEMPLATE_FILE}")")"
  template_id="$(printf '%s' "${template_resp}" | json_field '.data.templateId')"
  route_body="$(jq -n --arg name "${PROVIDER}-smoke-route" --arg templateId "${template_id}" --arg targetUrl "${TARGET_URL}" '{name:$name, templateId:$templateId, targetUrls:[$targetUrl], headers:{}, middlewareIds:[], mode:"raw", enabled:true}')"
  route_resp="$(api POST /api/routes "${route_body}")"
  ROUTE_ID="$(printf '%s' "${route_resp}" | json_field '.data.routeId')"
fi

deliver_resp="$(api POST "/api/routes/${ROUTE_ID}/deliver" "${PAYLOAD}")"
printf '%s\n' "${deliver_resp}" | jq '{
  code,
  statusCode: .data[0].statusCode,
  message: .data[0].message,
  targetUrl: .data[0].targetUrl,
  response: .data[0].response
}'
