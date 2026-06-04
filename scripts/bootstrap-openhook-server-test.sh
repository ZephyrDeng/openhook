#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="${ROOT_DIR}/scripts/bootstrap-openhook-server.sh"

if ! OPENHOOK_BOOTSTRAP_DRY_RUN=1 OPENHOOK_DOMAIN=your-domain.example "${SCRIPT}" >/tmp/openhook-bootstrap.out 2>/tmp/openhook-bootstrap.err; then
  cat /tmp/openhook-bootstrap.err >&2
  exit 1
fi

grep -q "systemd service" /tmp/openhook-bootstrap.out
grep -q "nginx reverse proxy" /tmp/openhook-bootstrap.out
grep -q "your-domain.example" /tmp/openhook-bootstrap.out

echo "BOOTSTRAP_DRY_RUN_OK"
