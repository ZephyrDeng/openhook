#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_HOST="${OPENHOOK_DEPLOY_HOST:-openhook}"
REMOTE_BIN="${OPENHOOK_REMOTE_BIN:-/opt/openhook/openhook}"
REMOTE_SERVICE="${OPENHOOK_REMOTE_SERVICE:-openhook}"
REMOTE_ENV_FILE="${OPENHOOK_REMOTE_ENV_FILE:-/etc/openhook/openhook.env}"
PUBLIC_URL="${OPENHOOK_PUBLIC_URL:?OPENHOOK_PUBLIC_URL is required}"
BUILD_PATH="${ROOT_DIR}/bin/openhook-linux-amd64"
DRY_RUN="${OPENHOOK_DEPLOY_DRY_RUN:-0}"

log() {
  printf '[openhook deploy] %s\n' "$*"
}

run_frontend_build() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    log "would build frontend"
    return
  fi
  if [[ ! -d "${ROOT_DIR}/frontend/node_modules" ]]; then
    log "install frontend dependencies"
    (cd "${ROOT_DIR}/frontend" && npm ci)
  fi
  log "build frontend"
  (cd "${ROOT_DIR}/frontend" && npm run build)
}

run_checks() {
  if [[ "${OPENHOOK_SKIP_CHECKS:-0}" == "1" ]]; then
    log "skip local checks"
    return
  fi
  if [[ "${DRY_RUN}" == "1" ]]; then
    log "would run go tests"
    log "would run local e2e"
    return
  fi
  log "run go tests"
  (cd "${ROOT_DIR}" && go test ./...)
  log "run local e2e"
  (cd "${ROOT_DIR}" && scripts/local-e2e.sh)
}

build_binary() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    log "would build linux amd64 binary"
    return
  fi
  log "build linux amd64 binary"
  mkdir -p "${ROOT_DIR}/bin"
  (cd "${ROOT_DIR}" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o "${BUILD_PATH}" ./cmd/openhook)
}

remote_set_env() {
  local key="$1"
  local value="$2"
  local encoded
  encoded="$(printf '%s' "${value}" | base64 | tr -d '\n')"
  ssh "${DEPLOY_HOST}" "sudo bash -s -- '${REMOTE_ENV_FILE}' '${key}'" <<EOF
set -euo pipefail
file="\$1"
key="\$2"
value="\$(printf '%s' '${encoded}' | base64 -d)"
dir="\$(dirname "\${file}")"
tmp="\$(mktemp)"
sudo mkdir -p "\${dir}"
sudo touch "\${file}"
sudo awk -v key="\${key}" 'index(\$0, key "=") != 1 { print }' "\${file}" > "\${tmp}"
escaped="\$(printf '%s' "\${value}" | sed 's/\\\\/\\\\\\\\/g; s/"/\\\\"/g')"
printf '%s="%s"\n' "\${key}" "\${escaped}" >> "\${tmp}"
sudo install -m 0600 "\${tmp}" "\${file}"
rm -f "\${tmp}"
EOF
}

sync_remote_env() {
  local changed=0
  if [[ "${DRY_RUN}" == "1" ]]; then
    if [[ -n "${OPENHOOK_PUBLIC_BASE_URL:-}" || -n "${OPENHOOK_GITHUB_CLIENT_ID:-}" || -n "${OPENHOOK_GITHUB_CLIENT_SECRET:-}" || -n "${OPENHOOK_SESSION_TTL:-}" ]]; then
      log "would update remote env"
    fi
    return
  fi
  if [[ -n "${OPENHOOK_PUBLIC_BASE_URL:-}" ]]; then
    remote_set_env "OPENHOOK_PUBLIC_BASE_URL" "${OPENHOOK_PUBLIC_BASE_URL}"
    changed=1
  fi
  if [[ -n "${OPENHOOK_GITHUB_CLIENT_ID:-}" ]]; then
    remote_set_env "OPENHOOK_GITHUB_CLIENT_ID" "${OPENHOOK_GITHUB_CLIENT_ID}"
    changed=1
  fi
  if [[ -n "${OPENHOOK_GITHUB_CLIENT_SECRET:-}" ]]; then
    remote_set_env "OPENHOOK_GITHUB_CLIENT_SECRET" "${OPENHOOK_GITHUB_CLIENT_SECRET}"
    changed=1
  fi
  if [[ -n "${OPENHOOK_SESSION_TTL:-}" ]]; then
    remote_set_env "OPENHOOK_SESSION_TTL" "${OPENHOOK_SESSION_TTL}"
    changed=1
  fi
  if [[ "${changed}" == "1" ]]; then
    log "remote env updated"
  fi
}

deploy_binary() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    log "would upload binary to ${DEPLOY_HOST} and restart ${REMOTE_SERVICE}"
    return
  fi
  log "upload binary to ${DEPLOY_HOST}"
  remote_tmp="$(ssh "${DEPLOY_HOST}" 'mktemp /tmp/openhook.XXXXXX')"
  scp -q "${BUILD_PATH}" "${DEPLOY_HOST}:${remote_tmp}"
  ssh "${DEPLOY_HOST}" \
    "sudo install -m 0755 '${remote_tmp}' '${REMOTE_BIN}' && sudo rm -f '${remote_tmp}' && sudo systemctl restart '${REMOTE_SERVICE}' && sudo systemctl is-active --quiet '${REMOTE_SERVICE}'"
}

verify_remote() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    log "would verify ${PUBLIC_URL}/health"
    log "would verify ${PUBLIC_URL}/api/auth/me"
    return
  fi
  log "verify ${PUBLIC_URL}/health"
  curl -fsS "${PUBLIC_URL}/health" >/dev/null
  auth_payload="$(curl -fsS "${PUBLIC_URL}/api/auth/me")"
  github_enabled="$(printf '%s' "${auth_payload}" | jq -r '.data.githubEnabled')"
  auth_required="$(printf '%s' "${auth_payload}" | jq -r '.data.authRequired')"
  log "auth status authRequired=${auth_required} githubEnabled=${github_enabled}"
  if [[ "${OPENHOOK_REQUIRE_GITHUB:-0}" == "1" && "${github_enabled}" != "true" ]]; then
    log "github oauth is required but not enabled"
    exit 1
  fi
  log "deploy ok"
}

run_production_smoke() {
  if [[ "${OPENHOOK_RUN_PRODUCTION_SMOKE:-0}" != "1" ]]; then
    return
  fi
  if [[ "${DRY_RUN}" == "1" ]]; then
    log "would run production smoke against ${PUBLIC_URL}"
    return
  fi
  log "run production smoke"
  (cd "${ROOT_DIR}" && OPENHOOK_API_BASE="${PUBLIC_URL}" scripts/production-smoke.sh)
}

run_production_readiness() {
  if [[ "${OPENHOOK_RUN_PRODUCTION_READINESS:-0}" != "1" ]]; then
    return
  fi
  if [[ "${DRY_RUN}" == "1" ]]; then
    log "would run production readiness against ${PUBLIC_URL}"
    return
  fi
  log "run production readiness"
  (cd "${ROOT_DIR}" && OPENHOOK_API_BASE="${PUBLIC_URL}" OPENHOOK_DEPLOY_HOST="${DEPLOY_HOST}" OPENHOOK_REMOTE_SERVICE="${REMOTE_SERVICE}" OPENHOOK_REMOTE_ENV_FILE="${REMOTE_ENV_FILE}" scripts/production-readiness.sh)
}

run_frontend_build
run_checks
build_binary
sync_remote_env
deploy_binary
verify_remote
run_production_smoke
run_production_readiness
