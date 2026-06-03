#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/openhook-e2e.XXXXXX")"
APP_PORT="${OPENHOOK_E2E_APP_PORT:-18080}"
RECEIVER_PORT="${OPENHOOK_E2E_RECEIVER_PORT:-18081}"
APP_URL="http://127.0.0.1:${APP_PORT}"
RECEIVER_URL="http://127.0.0.1:${RECEIVER_PORT}/webhook"
DB_PATH="${TMP_DIR}/openhook.db"
APP_LOG="${TMP_DIR}/openhook.log"
RECEIVER_LOG="${TMP_DIR}/receiver.jsonl"
RECEIVER_PID=""
APP_PID=""

cleanup() {
  if [[ -n "${APP_PID}" ]]; then
    kill "${APP_PID}" >/dev/null 2>&1 || true
    wait "${APP_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${RECEIVER_PID}" ]]; then
    kill "${RECEIVER_PID}" >/dev/null 2>&1 || true
    wait "${RECEIVER_PID}" >/dev/null 2>&1 || true
  fi
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
  echo "FAIL: $*" >&2
  echo "--- openhook log ---" >&2
  test -f "${APP_LOG}" && tail -80 "${APP_LOG}" >&2 || true
  echo "--- receiver log ---" >&2
  test -f "${RECEIVER_LOG}" && cat "${RECEIVER_LOG}" >&2 || true
  exit 1
}

json_get() {
  local expr="$1"
  python3 -c 'import json,sys; data=json.load(sys.stdin); cur=data; 
for part in sys.argv[1].split("."):
    if part:
        cur = cur[int(part)] if isinstance(cur, list) else cur[part]
print(cur)' "${expr}"
}

wait_for() {
  local url="$1"
  for _ in $(seq 1 80); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

cat >"${TMP_DIR}/receiver.py" <<'PY'
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

log_path = os.environ["RECEIVER_LOG"]

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
            return
        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        length = int(self.headers.get("content-length", "0"))
        raw = self.rfile.read(length)
        try:
            body = json.loads(raw or b"{}")
        except Exception:
            body = raw.decode("utf-8", "replace")
        record = {
            "path": self.path,
            "headers": {k.lower(): v for k, v in self.headers.items()},
            "body": body,
        }
        with open(log_path, "a", encoding="utf-8") as f:
            f.write(json.dumps(record, ensure_ascii=False, sort_keys=True) + "\n")
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.end_headers()
        self.wfile.write(b'{"receiver":"ok"}')

    def log_message(self, fmt, *args):
        return

ThreadingHTTPServer(("127.0.0.1", int(os.environ["RECEIVER_PORT"])), Handler).serve_forever()
PY

RECEIVER_LOG="${RECEIVER_LOG}" RECEIVER_PORT="${RECEIVER_PORT}" python3 "${TMP_DIR}/receiver.py" &
RECEIVER_PID="$!"
wait_for "http://127.0.0.1:${RECEIVER_PORT}/health" || fail "receiver did not start"

(
  cd "${ROOT_DIR}"
  OPENHOOK_DB="${DB_PATH}" OPENHOOK_ADDR=":${APP_PORT}" OPENHOOK_REQUEST_TIMEOUT=5 go run ./cmd/openhook
) >"${APP_LOG}" 2>&1 &
APP_PID="$!"
wait_for "${APP_URL}/health" || fail "openhook did not start"

template_resp="$(curl -fsS "${APP_URL}/api/templates" \
  -H 'Content-Type: application/json' \
  -d '{"templateName":"e2e-alert","msgType":"markdown","content":"# {{data.title}}\n- severity: {{data.severity}}\n- route: {{global.routeId}}","script":"ctx.title = ctx.title || \"untitled\"; return true;","simulation":{"title":"sample","severity":"warning"}}')"
template_id="$(printf '%s' "${template_resp}" | json_get data.templateId)"

preview_resp="$(curl -fsS "${APP_URL}/api/templates/preview" \
  -H 'Content-Type: application/json' \
  -d '{"content":"hello {{data.name}}","simulation":{"name":"openhook"}}')"
[[ "$(printf '%s' "${preview_resp}" | json_get data)" == "hello openhook" ]] || fail "template preview mismatch"

render_resp="$(curl -fsS "${APP_URL}/api/templates/${template_id}/render" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Render title","severity":"info"}')"
printf '%s' "${render_resp}" | grep -q 'Render title' || fail "template render mismatch"

token_resp="$(curl -fsS "${APP_URL}/api/tokens/create" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"e2e-token\",\"templateIds\":[\"${template_id}\"],\"userIds\":[\"api-user\"],\"isCoverAll\":false}")"
token="$(printf '%s' "${token_resp}" | json_get data.token)"

curl -fsS -X PUT "${APP_URL}/api/templates/${template_id}/token/${token}" \
  -H 'Content-Type: application/json' \
  -d '{"content":"# {{data.title}}\n- severity: {{data.severity}}\n- service: {{data.service}}\n- route: {{global.routeId}}","msgType":"markdown"}' >/dev/null

middleware_resp="$(curl -fsS "${APP_URL}/api/middlewares" \
  -H 'Content-Type: application/json' \
  -d '{"name":"e2e-header-and-severity","enabled":true,"code":"headers[\"X-E2E-Middleware\"] = \"hit\"; ctx.severity = ctx.severity || \"warning\"; return true;"}')"
middleware_id="$(printf '%s' "${middleware_resp}" | json_get data.middlewareId)"

route_resp="$(curl -fsS "${APP_URL}/api/routes" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"e2e-route\",\"templateId\":\"${template_id}\",\"targetUrls\":[\"${RECEIVER_URL}\"],\"headers\":{\"X-E2E-Route\":\"configured\"},\"middlewareIds\":[\"${middleware_id}\"],\"mode\":\"envelope\",\"enabled\":true}")"
route_id="$(printf '%s' "${route_resp}" | json_get data.routeId)"

route_deliver_resp="$(curl -fsS "${APP_URL}/api/routes/${route_id}/deliver" \
  -H 'Content-Type: application/json' \
  -H 'X-Request-ID: e2e-route-request' \
  -d '{"title":"Checkout down","service":"checkout"}')"
printf '%s' "${route_deliver_resp}" | grep -q '"code":0' || fail "route deliver failed"

direct_resp="$(curl -fsS "${APP_URL}/webhook/${template_id}?webhookUrls=${RECEIVER_URL}" \
  -H 'Content-Type: application/json' \
  -H 'X-Request-ID: e2e-direct-request' \
  -d '{"title":"Direct path","severity":"critical","service":"direct"}')"
printf '%s' "${direct_resp}" | grep -q '"code":0' || fail "direct webhook failed"

gitlab_resp="$(curl -fsS "${APP_URL}/webhook/gitlab?webhookUrls=${RECEIVER_URL}" \
  -H 'Content-Type: application/json' \
  -H 'X-Request-ID: e2e-gitlab-request' \
  -d '{"object_kind":"merge_request","project":{"name":"demo"},"object_attributes":{"title":"Add API","url":"https://gitlab.example/mr/1","source_branch":"feat","target_branch":"main","action":"open"}}')"
printf '%s' "${gitlab_resp}" | grep -q '"code":0' || fail "gitlab compatibility webhook failed"

sentry_resp="$(curl -fsS "${APP_URL}/webhook/sentry?webhookUrls=${RECEIVER_URL}" \
  -H 'Content-Type: application/json' \
  -H 'X-Request-ID: e2e-sentry-request' \
  -d '{"project_name":"checkout","message":"TypeError","web_url":"https://sentry.example/issues/1/","event":{"level":"error","environment":"prod","event_id":"evt1","timestamp":1710000000,"tags":[["url","/checkout"]]}}')"
printf '%s' "${sentry_resp}" | grep -q '"code":0' || fail "sentry compatibility webhook failed"

filter_resp="$(curl -fsS "${APP_URL}/api/filters" \
  -H 'Content-Type: application/json' \
  -d '{"name":"e2e-filter","status":true,"domain":["checkout"],"platform":"generic","payload":{"filters":[{"rules":[{"attribute":"severity","match":"eq","value":"critical"}]}]}}')"
filter_id="$(printf '%s' "${filter_resp}" | json_get data.id)"
curl -fsS "${APP_URL}/api/filters?domain=checkout" | grep -q 'e2e-filter' || fail "filter list failed"
curl -fsS -X PUT "${APP_URL}/api/filters/${filter_id}" -H 'Content-Type: application/json' -d '{"name":"e2e-filter","status":false,"domain":["checkout"],"platform":"generic","payload":{"updated":true}}' >/dev/null
curl -fsS -X DELETE "${APP_URL}/api/filters/${filter_id}" >/dev/null

dedup_resp="$(curl -fsS "${APP_URL}/api/dedup-rule" \
  -H 'Content-Type: application/json' \
  -d '{"name":"e2e-dedup","status":true,"domain":["checkout"],"platform":"generic","payload":{"alarmRules":["checkout-error"],"dedupRules":[{"sourceField":"traceId"}]}}')"
dedup_id="$(printf '%s' "${dedup_resp}" | json_get data.id)"
curl -fsS "${APP_URL}/api/dedup-rule/one?domain=checkout" | grep -q 'e2e-dedup' || fail "dedup active lookup failed"
curl -fsS -X PUT "${APP_URL}/api/dedup-rule/${dedup_id}" -H 'Content-Type: application/json' -d '{"name":"e2e-dedup","status":false,"domain":["checkout"],"platform":"generic","payload":{"updated":true}}' >/dev/null
curl -fsS -X DELETE "${APP_URL}/api/dedup-rule/${dedup_id}" >/dev/null

curl -fsS "${APP_URL}/api/deliveries?limit=10" | grep -q 'e2e-route-request' || fail "delivery log missing route request"

python3 - "${RECEIVER_LOG}" <<'PY'
import json
import sys

records = [json.loads(line) for line in open(sys.argv[1], encoding="utf-8")]
if len(records) < 4:
    raise SystemExit(f"expected at least 4 webhook records, got {len(records)}")

by_id = {r["headers"].get("x-openhook-request-id"): r for r in records}
route = by_id.get("e2e-route-request")
direct = by_id.get("e2e-direct-request")
gitlab = by_id.get("e2e-gitlab-request")
sentry = by_id.get("e2e-sentry-request")

if not route:
    raise SystemExit("route webhook record missing")
if route["headers"].get("x-e2e-middleware") != "hit":
    raise SystemExit("middleware header missing")
if route["headers"].get("x-e2e-route") != "configured":
    raise SystemExit("route header missing")
if route["body"].get("content", "").find("Checkout down") < 0:
    raise SystemExit("route content missing title")
if route["body"].get("messageContent", {}).get("severity") != "warning":
    raise SystemExit("middleware context mutation missing")

if not direct or direct["body"].get("content", "").find("Direct path") < 0:
    raise SystemExit("direct webhook content missing")
if not gitlab or gitlab["body"].get("content", "").find("Merge Request") < 0:
    raise SystemExit("gitlab compatibility content missing")
if not sentry or sentry["body"].get("content", "").find("CHECKOUT") < 0:
    raise SystemExit("sentry compatibility content missing")

print(json.dumps({
    "records": len(records),
    "routeContent": route["body"]["content"],
    "routeHeaders": {
        "x-e2e-route": route["headers"].get("x-e2e-route"),
        "x-e2e-middleware": route["headers"].get("x-e2e-middleware"),
    },
    "directContent": direct["body"]["content"],
    "gitlabContent": gitlab["body"]["content"],
    "sentryContent": sentry["body"]["content"],
}, ensure_ascii=False, indent=2))
PY

echo "E2E_OK app=${APP_URL} receiver=${RECEIVER_URL}"
