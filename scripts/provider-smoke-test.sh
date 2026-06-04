#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="${ROOT_DIR}/scripts/provider-smoke.sh"

if "${SCRIPT}" >/tmp/openhook-provider-smoke.out 2>/tmp/openhook-provider-smoke.err; then
  echo "expected provider-smoke.sh without provider to fail" >&2
  exit 1
fi

grep -q "usage:" /tmp/openhook-provider-smoke.err
grep -q "wecom" /tmp/openhook-provider-smoke.err
grep -q "telegram" /tmp/openhook-provider-smoke.err
grep -q "qq" /tmp/openhook-provider-smoke.err

echo "PROVIDER_SMOKE_USAGE_OK"
