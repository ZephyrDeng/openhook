#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="${ROOT_DIR}/scripts/deploy-openhook.sh"

grep -q "OPENHOOK_DEPLOY_DRY_RUN" "${SCRIPT}"
grep -q "OPENHOOK_RUN_PRODUCTION_SMOKE" "${SCRIPT}"
grep -q "scripts/production-smoke.sh" "${SCRIPT}"

echo "DEPLOY_OPENHOOK_TEST_OK"
