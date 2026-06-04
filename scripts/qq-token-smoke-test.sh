#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/openhook-qq-token-smoke-test.XXXXXX")"
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
        if self.path != "/app/getAppAccessToken":
            self.send_response(404)
            self.end_headers()
            return

        length = int(self.headers.get("content-length", "0"))
        raw = self.rfile.read(length)
        with open(os.environ["REQUEST_FILE"], "wb") as f:
            f.write(raw)

        body = json.dumps({"access_token": "mock-token-value", "expires_in": 7200}).encode()
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        return

ThreadingHTTPServer(("127.0.0.1", int(os.environ["PORT"])), Handler).serve_forever()
PY

REQUEST_FILE="${TMP_DIR}/request.json"
REQUEST_FILE="${REQUEST_FILE}" PORT="${PORT}" python3 "${TMP_DIR}/server.py" &
SERVER_PID="$!"

for _ in $(seq 1 80); do
  if curl -fsS -X POST "http://127.0.0.1:${PORT}/app/getAppAccessToken" -d '{}' >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

QQ_APP_ID="test-app" \
QQ_APP_SECRET="test-secret" \
QQ_TOKEN_URL="http://127.0.0.1:${PORT}/app/getAppAccessToken" \
"${ROOT_DIR}/scripts/qq-token-smoke.sh" >"${TMP_DIR}/out"

grep -q "QQ_TOKEN_OK expiresIn=7200" "${TMP_DIR}/out"
if grep -q "mock-token-value" "${TMP_DIR}/out"; then
  echo "qq token smoke leaked access token" >&2
  exit 1
fi

jq -e '.appId == "test-app" and .clientSecret == "test-secret"' "${REQUEST_FILE}" >/dev/null

if QQ_APP_ID="test-app" QQ_TOKEN_URL="http://127.0.0.1:${PORT}/app/getAppAccessToken" "${ROOT_DIR}/scripts/qq-token-smoke.sh" >"${TMP_DIR}/missing.out" 2>"${TMP_DIR}/missing.err"; then
  echo "expected missing QQ_APP_SECRET to fail" >&2
  exit 1
fi
grep -q "QQ_APP_SECRET is required" "${TMP_DIR}/missing.err"

echo "QQ_TOKEN_SMOKE_TEST_OK"
