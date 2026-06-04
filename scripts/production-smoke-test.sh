#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/openhook-production-smoke-test.XXXXXX")"
SERVER_PID=""

cleanup() {
  if [[ -n "${SERVER_PID}" ]]; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

PORT="$(python3 - <<'PY'
import socket
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
)"

cat >"${TMP_DIR}/server.py" <<'PY'
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_json({"code": 200, "message": "OK", "data": {"status": "ok"}})
            return
        if self.path == "/api/auth/me":
            self.send_json({"code": 200, "message": "OK", "data": {"authRequired": True, "authenticated": False, "githubEnabled": True}})
            return
        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        if self.path == "/app/getAppAccessToken":
            self.send_json({"access_token": "mock-production-token", "expires_in": 7200})
            return
        if self.path == "/v2/users/test-openid/messages":
            self.send_json({"id": "msg-456"})
            return
        self.send_response(404)
        self.end_headers()

    def send_json(self, body):
        raw = json.dumps(body).encode()
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def log_message(self, fmt, *args):
        return

ThreadingHTTPServer(("127.0.0.1", int(os.environ["PORT"])), Handler).serve_forever()
PY

PORT="${PORT}" python3 "${TMP_DIR}/server.py" &
SERVER_PID="$!"

for _ in $(seq 1 80); do
  if curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

OPENHOOK_API_BASE="http://127.0.0.1:${PORT}" \
OPENHOOK_REQUIRE_GITHUB=1 \
"${ROOT_DIR}/scripts/production-smoke.sh" >"${TMP_DIR}/out"

grep -q "HEALTH_OK" "${TMP_DIR}/out"
grep -q "AUTH_OK githubEnabled=true" "${TMP_DIR}/out"
grep -q "WECOM_SKIP" "${TMP_DIR}/out"
grep -q "TELEGRAM_SKIP" "${TMP_DIR}/out"
grep -q "QQ_TOKEN_SKIP" "${TMP_DIR}/out"
grep -q "QQ_DELIVERY_SKIP" "${TMP_DIR}/out"

OPENHOOK_API_BASE="http://127.0.0.1:${PORT}" \
OPENHOOK_REQUIRE_GITHUB=1 \
QQ_APP_ID="test-app" \
QQ_APP_SECRET="test-secret" \
QQ_OPENID="test-openid" \
QQ_TOKEN_URL="http://127.0.0.1:${PORT}/app/getAppAccessToken" \
QQ_API_BASE="http://127.0.0.1:${PORT}" \
"${ROOT_DIR}/scripts/production-smoke.sh" >"${TMP_DIR}/out-qq"

grep -q "QQ_TOKEN_OK expiresIn=7200" "${TMP_DIR}/out-qq"
grep -q "QQ_C2C_OK statusCode=200 messageId=msg-456" "${TMP_DIR}/out-qq"
grep -q "QQ_DELIVERY_SKIP" "${TMP_DIR}/out-qq"
if grep -q "mock-production-token" "${TMP_DIR}/out-qq"; then
  echo "production smoke leaked QQ access token" >&2
  exit 1
fi

echo "PRODUCTION_SMOKE_TEST_OK"
