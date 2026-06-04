#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/openhook-production-readiness-test.XXXXXX")"
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

cat >"${TMP_DIR}/ssh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${SSH_ARGS_FILE}"
cat >/dev/null || true
if [[ "$*" == *"systemctl is-active"* ]]; then
  echo "active"
  exit 0
fi
cat <<'EOF'
[openhook readiness] DB_OK path=/var/lib/openhook/openhook.db
[openhook readiness] GITHUB_USERS_OK count=1
[openhook readiness] USER_TEMPLATES_OK count=3
[openhook readiness] STARTER_TEMPLATES_OK count=3
[openhook readiness] WECOM_ROUTE_OK routeId=rt_bnU7i41mLNCStFbYYsy4bg
EOF
SH
chmod +x "${TMP_DIR}/ssh"

PORT="${PORT}" python3 "${TMP_DIR}/server.py" &
SERVER_PID="$!"

for _ in $(seq 1 80); do
  if curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

SSH_ARGS_FILE="${TMP_DIR}/ssh-args" \
OPENHOOK_API_BASE="http://127.0.0.1:${PORT}" \
OPENHOOK_SSH_BIN="${TMP_DIR}/ssh" \
"${ROOT_DIR}/scripts/production-readiness.sh" >"${TMP_DIR}/out"

grep -q "HEALTH_OK" "${TMP_DIR}/out"
grep -q "AUTH_OK githubEnabled=true authRequired=true" "${TMP_DIR}/out"
grep -q "SERVICE_OK service=openhook" "${TMP_DIR}/out"
grep -q "GITHUB_USERS_OK count=1" "${TMP_DIR}/out"
grep -q "USER_TEMPLATES_OK count=3" "${TMP_DIR}/out"
grep -q "STARTER_TEMPLATES_OK count=3" "${TMP_DIR}/out"
grep -q "WECOM_ROUTE_OK routeId=rt_bnU7i41mLNCStFbYYsy4bg" "${TMP_DIR}/out"
grep -q "PRODUCTION_READINESS_OK" "${TMP_DIR}/out"
grep -q "openhook" "${TMP_DIR}/ssh-args"
grep -q "rt_bnU7i41mLNCStFbYYsy4bg" "${TMP_DIR}/ssh-args"

echo "PRODUCTION_READINESS_TEST_OK"
