#!/usr/bin/env bash
set -euo pipefail

DEPLOY_HOST="${OPENHOOK_DEPLOY_HOST:-openhook}"
REMOTE_SERVICE="${OPENHOOK_REMOTE_SERVICE:-openhook}"
REMOTE_USER="${OPENHOOK_REMOTE_USER:-ubuntu}"
REMOTE_GROUP="${OPENHOOK_REMOTE_GROUP:-${REMOTE_USER}}"
REMOTE_BIN="${OPENHOOK_REMOTE_BIN:-/opt/openhook/openhook}"
REMOTE_ENV_FILE="${OPENHOOK_REMOTE_ENV_FILE:-/etc/openhook/openhook.env}"
REMOTE_DATA_DIR="${OPENHOOK_REMOTE_DATA_DIR:-/var/lib/openhook}"
OPENHOOK_ADDR_VALUE="${OPENHOOK_ADDR:-0.0.0.0:8080}"
OPENHOOK_DB_VALUE="${OPENHOOK_DB:-${REMOTE_DATA_DIR}/openhook.db}"
OPENHOOK_REQUEST_TIMEOUT_VALUE="${OPENHOOK_REQUEST_TIMEOUT:-10}"
OPENHOOK_DOMAIN_VALUE="${OPENHOOK_DOMAIN:-}"
OPENHOOK_PUBLIC_BASE_URL_VALUE="${OPENHOOK_PUBLIC_BASE_URL:-}"
OPENHOOK_TLS_EMAIL_VALUE="${OPENHOOK_TLS_EMAIL:-}"
OPENHOOK_ENABLE_TLS_VALUE="${OPENHOOK_ENABLE_TLS:-0}"

if [[ -z "${OPENHOOK_PUBLIC_BASE_URL_VALUE}" && -n "${OPENHOOK_DOMAIN_VALUE}" ]]; then
  OPENHOOK_PUBLIC_BASE_URL_VALUE="https://${OPENHOOK_DOMAIN_VALUE}"
fi

log() {
  printf '[openhook bootstrap] %s\n' "$*"
}

usage() {
  cat >&2 <<'EOF'
usage: scripts/bootstrap-openhook-server.sh

Environment:
  OPENHOOK_DEPLOY_HOST=openhook
  OPENHOOK_DOMAIN=commute-planner.site
  OPENHOOK_PUBLIC_BASE_URL=https://commute-planner.site
  OPENHOOK_ADMIN_TOKEN=optional-existing-token
  OPENHOOK_TLS_EMAIL=you@example.com
  OPENHOOK_ENABLE_TLS=1
  OPENHOOK_BOOTSTRAP_DRY_RUN=1
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ "${OPENHOOK_BOOTSTRAP_DRY_RUN:-0}" == "1" ]]; then
  log "would configure systemd service ${REMOTE_SERVICE} on ${DEPLOY_HOST}"
  log "would create ${REMOTE_DATA_DIR}, ${REMOTE_BIN%/*}, ${REMOTE_ENV_FILE}"
  if [[ -n "${OPENHOOK_DOMAIN_VALUE}" ]]; then
    log "would configure nginx reverse proxy for ${OPENHOOK_DOMAIN_VALUE}"
  fi
  if [[ "${OPENHOOK_ENABLE_TLS_VALUE}" == "1" ]]; then
    log "would request TLS certificate for ${OPENHOOK_DOMAIN_VALUE}"
  fi
  exit 0
fi

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "$1 is required" >&2
    exit 1
  }
}

require_cmd ssh

encoded_admin_token="$(printf '%s' "${OPENHOOK_ADMIN_TOKEN:-}" | base64 | tr -d '\n')"
encoded_public_base_url="$(printf '%s' "${OPENHOOK_PUBLIC_BASE_URL_VALUE}" | base64 | tr -d '\n')"
encoded_github_client_id="$(printf '%s' "${OPENHOOK_GITHUB_CLIENT_ID:-}" | base64 | tr -d '\n')"
encoded_github_client_secret="$(printf '%s' "${OPENHOOK_GITHUB_CLIENT_SECRET:-}" | base64 | tr -d '\n')"

log "configure remote host ${DEPLOY_HOST}"
ssh "${DEPLOY_HOST}" "sudo bash -s -- \
  '${REMOTE_SERVICE}' \
  '${REMOTE_USER}' \
  '${REMOTE_GROUP}' \
  '${REMOTE_BIN}' \
  '${REMOTE_ENV_FILE}' \
  '${REMOTE_DATA_DIR}' \
  '${OPENHOOK_ADDR_VALUE}' \
  '${OPENHOOK_DB_VALUE}' \
  '${OPENHOOK_REQUEST_TIMEOUT_VALUE}' \
  '${OPENHOOK_DOMAIN_VALUE}' \
  '${OPENHOOK_TLS_EMAIL_VALUE}' \
  '${OPENHOOK_ENABLE_TLS_VALUE}' \
  '${encoded_admin_token}' \
  '${encoded_public_base_url}' \
  '${encoded_github_client_id}' \
  '${encoded_github_client_secret}'" <<'REMOTE'
set -euo pipefail

service="$1"
user="$2"
group="$3"
remote_bin="$4"
env_file="$5"
data_dir="$6"
listen_addr="$7"
db_path="$8"
request_timeout="$9"
domain="${10}"
tls_email="${11}"
enable_tls="${12}"
admin_token_b64="${13}"
public_base_url_b64="${14}"
github_client_id_b64="${15}"
github_client_secret_b64="${16}"

decode() {
  printf '%s' "$1" | base64 -d
}

admin_token="$(decode "${admin_token_b64}")"
public_base_url="$(decode "${public_base_url_b64}")"
github_client_id="$(decode "${github_client_id_b64}")"
github_client_secret="$(decode "${github_client_secret_b64}")"

if [[ -z "${admin_token}" ]]; then
  admin_token="$(openssl rand -hex 32)"
fi

apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get install -y nginx curl ca-certificates openssl

install -d -o "${user}" -g "${group}" "${data_dir}"
install -d -o "${user}" -g "${group}" "$(dirname "${remote_bin}")"
install -d -m 0755 "$(dirname "${env_file}")"

tmp_env="$(mktemp)"
{
  printf 'OPENHOOK_ADDR=%q\n' "${listen_addr}"
  printf 'OPENHOOK_DB=%q\n' "${db_path}"
  printf 'OPENHOOK_REQUEST_TIMEOUT=%q\n' "${request_timeout}"
  printf 'OPENHOOK_PUBLIC_BASE_URL=%q\n' "${public_base_url}"
  printf 'OPENHOOK_ADMIN_TOKEN=%q\n' "${admin_token}"
  if [[ -n "${github_client_id}" ]]; then
    printf 'OPENHOOK_GITHUB_CLIENT_ID=%q\n' "${github_client_id}"
  fi
  if [[ -n "${github_client_secret}" ]]; then
    printf 'OPENHOOK_GITHUB_CLIENT_SECRET=%q\n' "${github_client_secret}"
  fi
} > "${tmp_env}"
install -m 0600 "${tmp_env}" "${env_file}"
rm -f "${tmp_env}"

cat >"/etc/systemd/system/${service}.service" <<EOF
[Unit]
Description=OpenHook webhook forwarding service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${user}
Group=${group}
WorkingDirectory=${data_dir}
EnvironmentFile=${env_file}
ExecStart=${remote_bin}
Restart=always
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ReadWritePaths=${data_dir}

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "${service}"

if [[ -n "${domain}" ]]; then
  cat >"/etc/nginx/sites-available/${service}.conf" <<EOF
server {
    listen 80;
    server_name ${domain} www.${domain};

    client_max_body_size 10m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
  ln -sfn "/etc/nginx/sites-available/${service}.conf" "/etc/nginx/sites-enabled/${service}.conf"
  nginx -t
  systemctl reload nginx || systemctl restart nginx

  if [[ "${enable_tls}" == "1" ]]; then
    if [[ -z "${tls_email}" ]]; then
      echo "OPENHOOK_TLS_EMAIL is required when OPENHOOK_ENABLE_TLS=1" >&2
      exit 1
    fi
    DEBIAN_FRONTEND=noninteractive apt-get install -y certbot python3-certbot-nginx
    certbot --nginx --non-interactive --agree-tos --redirect -m "${tls_email}" -d "${domain}" -d "www.${domain}"
  fi
fi

echo "BOOTSTRAP_OK service=${service} env=${env_file} data=${data_dir}"
REMOTE

log "bootstrap ok"
