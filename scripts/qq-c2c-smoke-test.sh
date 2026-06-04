#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/openhook-qq-c2c-smoke-test.XXXXXX")"
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
    def do_POST(self):
        length = int(self.headers.get("content-length", "0"))
        raw = self.rfile.read(length)
        if self.path == "/app/getAppAccessToken":
            self.send_json({"access_token": "mock-token-value", "expires_in": "7200"})
            return
        if self.path == "/v2/users/test-openid/messages":
            with open(os.environ["REQUEST_FILE"], "wb") as f:
                f.write(raw)
            with open(os.environ["AUTH_FILE"], "w") as f:
                f.write(self.headers.get("authorization", ""))
            with open(os.environ["APPID_FILE"], "w") as f:
                f.write(self.headers.get("x-union-appid", ""))
            self.send_json({"id": "msg-123", "content": "OpenHook QQ C2C smoke"})
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

REQUEST_FILE="${TMP_DIR}/request.json"
AUTH_FILE="${TMP_DIR}/authorization.txt"
APPID_FILE="${TMP_DIR}/appid.txt"
REQUEST_FILE="${REQUEST_FILE}" AUTH_FILE="${AUTH_FILE}" APPID_FILE="${APPID_FILE}" PORT="${PORT}" python3 "${TMP_DIR}/server.py" &
SERVER_PID="$!"

for _ in $(seq 1 80); do
  if curl -fsS -X POST "http://127.0.0.1:${PORT}/app/getAppAccessToken" -d '{}' >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

QQ_APP_ID="test-app" \
QQ_APP_SECRET="test-secret" \
QQ_OPENID="test-openid" \
QQ_TOKEN_URL="http://127.0.0.1:${PORT}/app/getAppAccessToken" \
QQ_API_BASE="http://127.0.0.1:${PORT}" \
QQ_MESSAGE="OpenHook QQ C2C smoke" \
QQ_MSG_SEQ="42" \
"${ROOT_DIR}/scripts/qq-c2c-smoke.sh" >"${TMP_DIR}/out"

grep -q "QQ_C2C_OK statusCode=200 messageId=msg-123" "${TMP_DIR}/out"
if grep -q "mock-token-value" "${TMP_DIR}/out"; then
  echo "qq c2c smoke leaked access token" >&2
  exit 1
fi
grep -q "QQBot mock-token-value" "${AUTH_FILE}"
grep -q "test-app" "${APPID_FILE}"
jq -e '.content == "OpenHook QQ C2C smoke" and .msg_type == 0 and .msg_seq == 42' "${REQUEST_FILE}" >/dev/null

if QQ_APP_ID="test-app" QQ_APP_SECRET="test-secret" "${ROOT_DIR}/scripts/qq-c2c-smoke.sh" >"${TMP_DIR}/missing.out" 2>"${TMP_DIR}/missing.err"; then
  echo "expected missing QQ_OPENID to fail" >&2
  exit 1
fi
grep -q "QQ_OPENID is required" "${TMP_DIR}/missing.err"

echo "QQ_C2C_SMOKE_TEST_OK"
