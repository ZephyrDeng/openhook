# OpenHook 本地验证报告

日期：2026-06-03

## 验证结论

OpenHook 已通过本地验证，可以作为独立的 Go webhook 转发服务运行。

本地 E2E 验证启动了：

- OpenHook 服务：`http://127.0.0.1:18080`
- 本地 webhook 接收器：`http://127.0.0.1:18081/webhook`
- 临时 SQLite 数据库

本地 webhook 接收器实际记录到 4 条外发 webhook 请求：

- route 配置转发：`POST /api/routes/{routeId}/deliver`
- 直接模板转发：`POST /webhook/{templateId}`
- GitLab 兼容入口转发：`POST /webhook/gitlab`
- Sentry 兼容入口转发：`POST /webhook/sentry`

## 验证命令

```bash
make lint
make test
make build
make e2e
pattern="$(printf 'sl-%s' 'webhook')" && ! rg -n "$pattern"
```

## 验证证据

### 单元测试与 Handler 测试

`go test ./...` 通过。

覆盖包：

- `internal/httpapi`
- `internal/middleware`
- `internal/render`
- `internal/store`

HTTP handler 测试会创建模板和 route，发送 route delivery 请求，并验证本地目标服务收到渲染后的 payload。补充测试覆盖了 admin token 保护、route 更新、token 编辑权限、rule domain 过滤、delivery 查询。

### 构建验证

`go build ./cmd/openhook` 通过。

### 本地真实 Webhook 请求

`scripts/local-e2e.sh` 通过。验证范围：

- 模板创建、preview、render
- token 范围内模板更新
- middleware 修改 headers 和请求上下文
- route delivery
- 直接模板 webhook 转发
- GitLab 兼容入口转发
- Sentry 兼容入口转发
- filter CRUD 和 domain 查询
- dedup rule CRUD 和 active 查询
- delivery 日志查询

### 关键字扫描

仓库内旧名称扫描结果为 0 命中。

## API 覆盖

E2E 直接验证的接口：

- `GET /health`
- `POST /api/templates`
- `POST /api/templates/preview`
- `POST /api/templates/{templateId}/render`
- `POST /api/tokens/create`
- `PUT /api/templates/{templateId}/token/{token}`
- `POST /api/middlewares`
- `POST /api/routes`
- `POST /api/routes/{routeId}/deliver`
- `POST /webhook/{templateId}?webhookUrls=...`
- `POST /webhook/gitlab?webhookUrls=...`
- `POST /webhook/sentry?webhookUrls=...`
- `POST /api/filters`
- `GET /api/filters?domain=...`
- `PUT /api/filters/{id}`
- `DELETE /api/filters/{id}`
- `POST /api/dedup-rule`
- `GET /api/dedup-rule/one?domain=...`
- `PUT /api/dedup-rule/{id}`
- `DELETE /api/dedup-rule/{id}`
- `GET /api/deliveries?limit=10`

单元测试覆盖：

- HTTP route delivery 到达本地接收器
- Admin token 写接口保护
- 模板占位符渲染
- JSON object 模板输出
- JS middleware 修改上下文
- JS middleware 拒绝请求
- Store 层模板、route、token、rule set、delivery 行为

## 需求验收

| 需求 | 证据 | 结果 |
| --- | --- | --- |
| 本地完整自测 | `make ci` | 通过 |
| 本地实际 webhook 请求 | 本地 receiver 记录到 route、direct、GitLab、Sentry 四类外发请求 | 通过 |
| SQLite 数据库 | E2E 使用临时 SQLite DB；schema 位于 `internal/store/store.go` | 通过 |
| 纯 webhook 转发配置 | route model 存储 target URLs、headers、middleware IDs、mode、enabled | 通过 |
| 保留自定义中间件能力 | E2E 中 middleware 修改 headers 和请求上下文，receiver 已验证 | 通过 |
| 开源依赖形态 | `go.mod` 只使用开源 Go module | 通过 |

## 建议

将 `make ci` 作为 OpenHook 的本地发布门禁。GitHub Actions 已配置 push、pull request 和 tag release 验证。
