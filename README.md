# OpenHook

OpenHook is a Go webhook forwarding service with a built-in web management console. It supports message templates, route-based forwarding, token-scoped template updates, filters, dedup rules, delivery logs, and custom JavaScript middleware.

The backend uses SQLite by default and depends only on open-source Go modules. The frontend is a Svelte 5 SPA embedded directly into the Go binary for single-binary deployment.

## Features

- **Web Management Console** — Svelte 5 + Tailwind CSS, embedded into the Go binary.
- SQLite persistence with automatic migrations.
- Template CRUD with `{{data.xxx}}` and `{{global.xxx}}` placeholders.
- **Real-time template preview** — Edit and preview message rendering side-by-side.
- Route configuration for reusable webhook forwarding.
- Generic HTTP webhook forwarding with `envelope` and `raw` modes.
- Token management for external template updates.
- Custom JavaScript middleware powered by `goja`.
- Filter and dedup-rule CRUD storage.
- GitLab and Sentry compatible webhook entrypoints.
- Delivery logs for debugging outbound requests.
- Docker and docker-compose support.

## Quick Start

### Single Binary (Frontend + Backend)

```bash
make build    # builds frontend then Go binary
./bin/openhook
```

Then open `http://localhost:8080/` in your browser for the web console.

### Backend Only

```bash
go run ./cmd/openhook
```

The server listens on `:8080` and creates `openhook.db` in the current directory.

### Environment Variables

```bash
OPENHOOK_ADDR=:8080
OPENHOOK_DB=openhook.db
OPENHOOK_REQUEST_TIMEOUT=10
OPENHOOK_ADMIN_TOKEN=
OPENHOOK_PUBLIC_BASE_URL=http://localhost:8080
OPENHOOK_GITHUB_CLIENT_ID=
OPENHOOK_GITHUB_CLIENT_SECRET=
OPENHOOK_SESSION_TTL=2592000
```

Set `OPENHOOK_ADMIN_TOKEN` to protect write APIs. The web console will prompt for it in Settings, or send it with:

```http
X-OpenHook-Admin-Token: your-token
```

GitHub login is enabled when `OPENHOOK_GITHUB_CLIENT_ID` and `OPENHOOK_GITHUB_CLIENT_SECRET` are set. Use this callback URL in the GitHub OAuth app:

```text
https://your-domain.example/auth/github/callback
```

The web console login page is:

```text
https://your-domain.example/login
```

GitHub OAuth entrypoints:

```text
https://your-domain.example/login/github
https://your-domain.example/register/github
```

Logged-in users get isolated templates and routes. Admin-token requests can manage all templates, routes, tokens, middlewares, filters, dedup rules, and delivery logs.

## Docker

```bash
docker compose up --build
```

## Production Deploy

First-time server bootstrap for a fresh Ubuntu host:

```bash
OPENHOOK_DEPLOY_HOST=openhook \
OPENHOOK_DOMAIN=commute-planner.site \
OPENHOOK_PUBLIC_BASE_URL=https://commute-planner.site \
OPENHOOK_TLS_EMAIL=you@example.com \
OPENHOOK_ENABLE_TLS=1 \
scripts/bootstrap-openhook-server.sh
```

Bootstrap creates:

- `/etc/openhook/openhook.env`
- `/opt/openhook/`
- `/var/lib/openhook/`
- `openhook.service`
- nginx reverse proxy for the domain
- optional Let's Encrypt certificate when `OPENHOOK_ENABLE_TLS=1`

Daily deploy for an initialized host:

```bash
OPENHOOK_DEPLOY_HOST=openhook \
OPENHOOK_PUBLIC_URL=https://commute-planner.site \
scripts/deploy-openhook.sh
```

The script builds the frontend, runs Go tests, runs local e2e, cross-compiles a Linux amd64 binary, uploads it to `/opt/openhook/openhook`, restarts `openhook.service`, and checks `/health`.

One-command production deploy with WeCom production smoke:

```bash
scripts/deploy-production.sh
```

The same path is available through:

```bash
make deploy-production
```

`scripts/deploy-production.sh` defaults to `OPENHOOK_DEPLOY_HOST=openhook`, `OPENHOOK_PUBLIC_URL=https://commute-planner.site`, `OPENHOOK_REQUIRE_GITHUB=1`, `OPENHOOK_RUN_PRODUCTION_SMOKE=1`, and the current production WeCom route ID. Override any of those environment variables when deploying a different host or route.

To enable GitHub registration/login during the same deploy:

```bash
OPENHOOK_DEPLOY_HOST=openhook \
OPENHOOK_PUBLIC_URL=https://commute-planner.site \
OPENHOOK_PUBLIC_BASE_URL=https://commute-planner.site \
OPENHOOK_GITHUB_CLIENT_ID=github-oauth-client-id \
OPENHOOK_GITHUB_CLIENT_SECRET=github-oauth-client-secret \
OPENHOOK_REQUIRE_GITHUB=1 \
scripts/deploy-openhook.sh
```

`OPENHOOK_REQUIRE_GITHUB=1` makes deployment fail when `/api/auth/me` does not report `githubEnabled:true`.

Production smoke after deploy:

```bash
OPENHOOK_REQUIRE_GITHUB=1 \
OPENHOOK_WECOM_ROUTE_ID=rt_xxx \
QQ_APP_ID=qq-bot-app-id \
QQ_APP_SECRET=qq-bot-app-secret \
QQ_OPENID=qq-c2c-recipient-openid \
scripts/production-smoke.sh
```

The smoke script checks `/health`, `/api/auth/me`, WeCom when `OPENHOOK_WECOM_ROUTE_ID` or `WECOM_WEBHOOK_URL` is present, Telegram when `OPENHOOK_TELEGRAM_ROUTE_ID` or `TELEGRAM_WEBHOOK_URL` plus `TELEGRAM_CHAT_ID` are present, QQ official bot token access when `QQ_APP_ID` and `QQ_APP_SECRET` are present, QQ official C2C delivery when `QQ_OPENID` is also present, and QQ route delivery when `OPENHOOK_QQ_ROUTE_ID` or `QQ_WEBHOOK_URL` is present. Missing provider credentials are reported as skips so the same command can be used while QQ and Telegram delivery targets are still being prepared.

Production environment file example:

```bash
OPENHOOK_ADDR=:8080
OPENHOOK_DB=/var/lib/openhook/openhook.db
OPENHOOK_REQUEST_TIMEOUT=10
OPENHOOK_ADMIN_TOKEN=replace-with-a-long-random-token
OPENHOOK_PUBLIC_BASE_URL=https://commute-planner.site
OPENHOOK_GITHUB_CLIENT_ID=github-oauth-client-id
OPENHOOK_GITHUB_CLIENT_SECRET=github-oauth-client-secret
OPENHOOK_SESSION_TTL=2592000
```

On the server this is currently expected at:

```text
/etc/openhook/openhook.env
```

## Core Workflow

Create a template:

```bash
curl -s http://localhost:8080/api/templates \
  -H 'Content-Type: application/json' \
  -d @examples/template.json
```

Create a route after replacing `templateId` and `targetUrls`:

```bash
curl -s http://localhost:8080/api/routes \
  -H 'Content-Type: application/json' \
  -d @examples/route.json
```

Deliver through the route:

```bash
curl -s http://localhost:8080/api/routes/{routeId}/deliver \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Checkout error",
    "severity": "critical",
    "service": "checkout",
    "traceId": "trace-123"
  }'
```

Send with the compatibility template webhook endpoint:

```bash
curl -s 'http://localhost:8080/webhook/{templateId}?webhookUrls=https://example.com/webhook' \
  -H 'Content-Type: application/json' \
  -d '{"title":"Checkout error","severity":"critical"}'
```

When authentication is enabled, this direct template webhook requires an admin token or a logged-in user session that owns the template. Production integrations should use route delivery:

```text
POST /api/routes/{routeId}/deliver
```

## Provider Templates

Provider-specific templates use `mode: raw`, so OpenHook sends the rendered JSON body directly to the target webhook.

Built-in examples:

```text
examples/providers/wecom-markdown-template.json
examples/providers/telegram-send-message-template.json
examples/providers/qq-webhook-text-template.json
examples/providers/raw-route.json
```

WeCom group robot:

```bash
curl -s http://localhost:8080/api/templates \
  -H 'Content-Type: application/json' \
  -d @examples/providers/wecom-markdown-template.json
```

Use route `mode: raw` and target URL:

```text
https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...
```

Telegram Bot API:

```bash
curl -s http://localhost:8080/api/templates \
  -H 'Content-Type: application/json' \
  -d @examples/providers/telegram-send-message-template.json
```

Use route `mode: raw` and target URL:

```text
https://api.telegram.org/bot<token>/sendMessage
```

Telegram payload requires `chatId` in the delivery data. Telegram `sendMessage` accepts `chat_id`, `text`, and optional `parse_mode`.

QQ webhook bridge:

```bash
curl -s http://localhost:8080/api/templates \
  -H 'Content-Type: application/json' \
  -d @examples/providers/qq-webhook-text-template.json
```

QQ official bot integrations differ by bridge or SDK. Keep route `mode: raw`, set the target URL to the QQ bridge webhook, then adjust the JSON body to that bridge's required shape.

QQ official bot token smoke:

```bash
QQ_APP_ID='qq-bot-app-id' \
QQ_APP_SECRET='qq-bot-app-secret' \
scripts/qq-token-smoke.sh
```

`scripts/qq-token-smoke.sh` calls `https://bots.qq.com/app/getAppAccessToken` with `appId` and `clientSecret`, then prints `QQ_TOKEN_OK expiresIn=...` without printing the access token. Official C2C delivery still requires a recipient openid from QQ bot events or a QQ bridge/SDK endpoint that accepts OpenHook's rendered JSON.

QQ official C2C smoke:

```bash
QQ_APP_ID='qq-bot-app-id' \
QQ_APP_SECRET='qq-bot-app-secret' \
QQ_OPENID='qq-c2c-recipient-openid' \
scripts/qq-c2c-smoke.sh
```

`scripts/qq-c2c-smoke.sh` sends a text payload to `https://api.sgroup.qq.com/v2/users/{openid}/messages` with `Authorization: QQBot ...` and `X-Union-Appid`. Use `QQ_SANDBOX=1` for sandbox API or `QQ_API_BASE` to override the API host.

Provider smoke helper:

```bash
WECOM_WEBHOOK_URL='https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...' \
scripts/provider-smoke.sh wecom

TELEGRAM_WEBHOOK_URL='https://api.telegram.org/bot<TOKEN>/sendMessage' \
TELEGRAM_CHAT_ID='123456789' \
scripts/provider-smoke.sh telegram

QQ_WEBHOOK_URL='https://example.com/qq-webhook' \
scripts/provider-smoke.sh qq
```

For existing production routes, pass `OPENHOOK_ROUTE_ID` and omit the provider webhook URL:

```bash
OPENHOOK_ROUTE_ID=rt_xxx scripts/provider-smoke.sh wecom
```

## Payload Modes

`envelope` mode sends:

```json
{
  "msgType": "markdown",
  "content": "# Checkout error",
  "messageContent": {
    "title": "Checkout error"
  },
  "timestamp": 1710000000000,
  "requestId": "req_xxx"
}
```

`raw` mode sends the rendered `content` as the request body.

For raw JSON templates, use `{{json data.xxx}}` for string fields. It renders a JSON-safe literal, so quotes and newlines in user input cannot break the payload:

```json
{
  "text": {{json data.text}}
}
```

## Custom Middleware

Middleware receives three mutable globals:

- `ctx`: incoming request JSON.
- `global`: route, query, and request metadata.
- `headers`: outbound HTTP headers.

Return values:

- `true` or empty return continues forwarding.
- `false` rejects forwarding.
- `{ reject: true, message: "reason" }` rejects forwarding with a reason.

Example:

```js
if (ctx.environment === "test") {
  return { reject: true, message: "test environment ignored" };
}

headers["X-OpenHook-Source"] = "custom-middleware";
ctx.severity = ctx.severity || "warning";
return true;
```

Create middleware:

```bash
curl -s http://localhost:8080/api/middlewares \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "drop-test",
    "enabled": true,
    "code": "if (ctx.environment === \"test\") return { reject: true, message: \"ignored\" }; return true;"
  }'
```

Attach returned `middlewareId` to a route through `middlewareIds`.

## API Reference

Templates:

- `GET /api/templates`
- `GET /api/templates/paginated?page=1&size=20&search=keyword`
- `POST /api/templates`
- `GET /api/templates/{templateId}`
- `PUT /api/templates/{templateId}`
- `DELETE /api/templates/{templateId}`
- `POST /api/templates/{templateId}/render`
- `POST /api/templates/preview`
- `PUT /api/templates/{templateId}/token/{token}`

Tokens:

- `GET /api/tokens`
- `POST /api/tokens/create`
- `GET /api/tokens/{token}`
- `POST /api/tokens/{token}`
- `DELETE /api/tokens/{token}`

Routes:

- `GET /api/routes`
- `POST /api/routes`
- `GET /api/routes/{routeId}`
- `PUT /api/routes/{routeId}`
- `DELETE /api/routes/{routeId}`
- `POST /api/routes/{routeId}/deliver`

Middleware:

- `GET /api/middlewares`
- `POST /api/middlewares`
- `GET /api/middlewares/{middlewareId}`
- `PUT /api/middlewares/{middlewareId}`
- `DELETE /api/middlewares/{middlewareId}`

Filters:

- `GET /api/filters`
- `POST /api/filters`
- `PUT /api/filters/{id}`
- `DELETE /api/filters/{id}`

Dedup rules:

- `GET /api/dedup-rule`
- `GET /api/dedup-rule/one`
- `POST /api/dedup-rule`
- `PUT /api/dedup-rule/{id}`
- `DELETE /api/dedup-rule/{id}`

Webhook compatibility:

- `POST /webhook/{templateId}?webhookUrls=url1,url2`
- `POST /webhook/gitlab?webhookUrls=url1,url2`
- `POST /webhook/sentry?webhookUrls=url1,url2`

When authentication is enabled, direct `webhookUrls` compatibility calls require an admin token. Compatibility calls using `routeId` stay public for external webhook integrations and respect the route `enabled` flag.

Deliveries:

- `GET /api/deliveries?limit=50&offset=0`

## Development

### Backend

```bash
go test ./...
go build ./cmd/openhook
```

Run with a temporary database:

```bash
OPENHOOK_DB=/tmp/openhook.db go run ./cmd/openhook
```

### Frontend

The web console lives in `frontend/` and is built with Svelte 5 + Tailwind CSS + Vite.

```bash
cd frontend
npm install

# Dev mode with hot reload (proxies API to localhost:8080)
npm run dev

# Production build (outputs to internal/static/dist/)
npm run build
```

### Building Everything

```bash
make build-all   # frontend + Go binary
make run         # build then run the binary
```

## Project Layout

```text
cmd/openhook          executable entrypoint
internal/httpapi      HTTP routing and handlers
internal/store        SQLite schema and persistence
internal/render       template rendering
internal/middleware   JavaScript middleware runtime
internal/forward      outbound webhook sender
internal/model        API and persistence models
internal/static       embedded frontend assets (dist/)
frontend/             Svelte 5 web console source
examples              template, route, and middleware examples
```
