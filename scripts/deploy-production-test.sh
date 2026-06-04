#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_FILE="$(mktemp "${TMPDIR:-/tmp}/openhook-deploy-production-test.XXXXXX")"
trap 'rm -f "${OUT_FILE}"' EXIT

if OPENHOOK_DEPLOY_DRY_RUN=1 "${ROOT_DIR}/scripts/deploy-production.sh" >"${OUT_FILE}" 2>&1; then
  echo "expected deploy-production to require OPENHOOK_PUBLIC_URL" >&2
  exit 1
fi
grep -q "OPENHOOK_PUBLIC_URL is required" "${OUT_FILE}"

OPENHOOK_DEPLOY_DRY_RUN=1 \
OPENHOOK_PUBLIC_URL=https://your-domain.example \
"${ROOT_DIR}/scripts/deploy-production.sh" >"${OUT_FILE}"

grep -q "OPENHOOK_DEPLOY_HOST=openhook" "${OUT_FILE}"
grep -q "OPENHOOK_PUBLIC_URL=https://your-domain.example" "${OUT_FILE}"
grep -q "OPENHOOK_REQUIRE_GITHUB=1" "${OUT_FILE}"
grep -q "OPENHOOK_RUN_PRODUCTION_SMOKE=1" "${OUT_FILE}"
grep -q "OPENHOOK_RUN_PRODUCTION_READINESS=1" "${OUT_FILE}"
grep -q "OPENHOOK_WECOM_ROUTE_ID=rt_bnU7i41mLNCStFbYYsy4bg" "${OUT_FILE}"
grep -q "would run production smoke against https://your-domain.example" "${OUT_FILE}"
grep -q "would run production readiness against https://your-domain.example" "${OUT_FILE}"

OPENHOOK_DEPLOY_DRY_RUN=1 \
OPENHOOK_PUBLIC_URL=https://your-domain.example \
OPENHOOK_WECOM_ROUTE_ID=rt_override \
"${ROOT_DIR}/scripts/deploy-production.sh" >"${OUT_FILE}"

grep -q "OPENHOOK_WECOM_ROUTE_ID=rt_override" "${OUT_FILE}"

echo "DEPLOY_PRODUCTION_TEST_OK"
