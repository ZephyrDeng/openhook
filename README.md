# OpenHook

OpenHook is a Go webhook forwarding service inspired by the original `sl-webhook` backend. It keeps the public webhook configuration workflow: message templates, route-based forwarding, token-scoped template updates, filters, dedup rules, delivery logs, and custom middleware.

The implementation uses SQLite by default and depends only on open-source Go modules.

## Features

- SQLite persistence with automatic migrations.
- Template CRUD with `{{data.xxx}}` and `{{global.xxx}}` placeholders.
- Template preview and render APIs.
- Route configuration for reusable webhook forwarding.
- Generic HTTP webhook forwarding with `envelope` and `raw` modes.
- Token management for external template updates.
- Custom JavaScript middleware powered by `goja`.
- Filter and dedup-rule CRUD storage.
- GitLab and Sentry compatible webhook entrypoints that transform events into generic webhook payloads.
- Delivery logs for debugging outbound requests.
- Docker and docker-compose support.

## Quick Start

```bash
go run ./cmd/openhook
```

The server listens on `:8080` and creates `openhook.db` in the current directory.

Useful environment variables:

```bash
OPENHOOK_ADDR=:8080
OPENHOOK_DB=openhook.db
OPENHOOK_REQUEST_TIMEOUT=10
OPENHOOK_ADMIN_TOKEN=
OPENHOOK_PUBLIC_BASE_URL=http://localhost:8080
```

Set `OPENHOOK_ADMIN_TOKEN` to protect write APIs. Send the token with:

```http
X-OpenHook-Admin-Token: your-token
```

## Docker

```bash
docker compose up --build
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

Deliveries:

- `GET /api/deliveries?limit=50&offset=0`

## Development

```bash
go test ./...
go build ./cmd/openhook
```

Run with a temporary database:

```bash
OPENHOOK_DB=/tmp/openhook.db go run ./cmd/openhook
```

## Project Layout

```text
cmd/openhook        executable entrypoint
internal/httpapi    HTTP routing and handlers
internal/store      SQLite schema and persistence
internal/render     template rendering
internal/middleware JavaScript middleware runtime
internal/forward    outbound webhook sender
internal/model      API and persistence models
examples            template, route, and middleware examples
```
