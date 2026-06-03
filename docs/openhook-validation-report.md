# OpenHook 本地验证报告

日期：2026-06-03

## 验证结论

OpenHook 已通过本地验证，可以作为一个独立的 Go webhook 转发服务运行。

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
go test ./...
go build ./cmd/openhook
scripts/local-e2e.sh
rg -n "fishbone|wework|企微|qyapi|inshopline|@yy|wecom|themis|nacos|dubbo" \
  cmd internal examples scripts go.mod README.md Dockerfile docker-compose.yml Makefile .env.example .dockerignore
```

## 验证证据

### 单元测试与 Handler 测试

`go test ./...` 通过。

覆盖包：

- `internal/httpapi`
- `internal/middleware`
- `internal/render`

HTTP handler 测试会创建模板和 route，发送一次 route delivery 请求，并验证本地目标服务收到渲染后的 payload。

### 构建验证

`go build ./cmd/openhook` 通过。

### 本地真实 Webhook 请求

`scripts/local-e2e.sh` 通过，输出如下：

```json
{
  "records": 4,
  "routeContent": "# Checkout down\n- severity: warning\n- service: checkout\n- route: rt_WFb1oT9gseTTxaugeM3EGg",
  "routeHeaders": {
    "x-e2e-route": "configured",
    "x-e2e-middleware": "hit"
  },
  "directContent": "# Direct path\n- severity: critical\n- service: direct\n- route: ",
  "gitlabContent": "# demo Merge Request\n- Title: [Add API](https://gitlab.example/mr/1)\n- Source: feat\n- Target: main\n- Action: open",
  "sentryContent": "# CHECKOUT alert\n- message: TypeError\n- level: error\n- environment: prod\n- url: /checkout\n- event_id: evt1\n- time: 2024-03-10T00:00:00+08:00"
}
E2E_OK app=http://127.0.0.1:18080 receiver=http://127.0.0.1:18081/webhook
```

route delivery 验证点：

- 渲染后的模板内容已到达本地接收器
- route 配置的 headers 已到达本地接收器
- 自定义 middleware 设置的 headers 已到达本地接收器
- 自定义 middleware 已将 `ctx.severity` 补齐为 `warning`
- delivery 日志已记录 request id

### 内网依赖扫描

新 Go 项目文件内的内网与企微相关关键字扫描结果为 0 命中。

扫描关键字：

- `fishbone`
- `wework`
- `企微`
- `qyapi`
- `inshopline`
- `@yy`
- `wecom`
- `themis`
- `nacos`
- `dubbo`

## 与原 sl-webhook 对比

| 模块 | 原 sl-webhook | OpenHook Go 实现 | 状态 |
| --- | --- | --- | --- |
| 运行时 | NestJS / TypeScript | Go HTTP 服务 | 已实现 |
| 持久化 | Fishbone/Mongo 风格模型 | SQLite 自动建表迁移 | 已实现 |
| 模板管理 | `message-template` CRUD 和 preview | `templates` CRUD、preview、render、分页 | 已实现 |
| 模板语法 | Handlebars 和多组内网 helper | `{{data.xxx}}` 与 `{{global.xxx}}` 占位符渲染 | 已按开源范围实现 |
| 模板脚本 | 模板附带 sandbox script | 基于 `goja` 执行模板内联脚本和自定义 middleware | 已实现 |
| Webhook 投递 | 企微机器人投递和通用 HTTP callback | 通用 HTTP webhook 投递 | 已实现 |
| 转发配置 | query 传入 webhookUrls 和 templateId | 持久化 route，包含目标 URL、headers、middlewareIds、mode、enabled | 已实现 |
| Token 能力 | token 范围内编辑模板 | token CRUD 和 `PUT /api/templates/{templateId}/token/{token}` | 已实现 |
| Middleware | 内置 Sentry/Jira/电话/值班/Pulsar/APM middleware | 用户自定义 JS middleware，可修改 `ctx`、`global`、`headers` | 已按开源范围实现 |
| Filters | 预诊断规则 CRUD | 通用 filter rule-set CRUD | 已作为配置面实现 |
| Dedup rules | 告警判重规则 CRUD | 通用 dedup rule-set CRUD 和 active 查询 | 已作为配置面实现 |
| GitLab webhook | GitLab 事件转换为企微 markdown | GitLab 事件转换为通用 webhook markdown | 已实现 |
| Sentry webhook | Sentry 事件转换为企微 text，并带内网增强 | Sentry 事件转换为通用 webhook envelope 内容 | 已实现 |
| 投递可观测性 | 运行日志和内网告警存储 | delivery 表和 `GET /api/deliveries` | 已实现 |
| 文件上传 | 企微机器人 media upload/send-file | 已从 OpenHook 范围移除 | 已按范围裁剪 |
| 企微消息中心 | 内网消息中心 API | 已从 OpenHook 范围移除 | 已按范围裁剪 |
| Jira 自动建单 | 内网 Jira 服务和规则动作 | 由自定义 middleware 扩展点承接 | 已按范围裁剪 |
| 值班表 | 内网值班配置查询 | 由自定义 middleware 或外部目标系统承接 | 已按范围裁剪 |
| AI bot / AI diagnosis | 内网 AI 与 Sage 集成 | 已从 OpenHook 范围移除 | 已按范围裁剪 |
| SSO / 管理身份 | Fishbone SSO | 可选 `OPENHOOK_ADMIN_TOKEN` 写接口保护 | 已实现 |
| 内网配置 | Nacos/Dubbo/Fishbone 配置 | 环境变量和 SQLite 配置行 | 已实现 |

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
- 模板占位符渲染
- JSON object 模板输出
- JS middleware 修改上下文
- JS middleware 拒绝请求

## 需求验收

| 需求 | 证据 | 结果 |
| --- | --- | --- |
| 本地完整自测 | `go test ./...`、`go build ./cmd/openhook`、`scripts/local-e2e.sh` | 通过 |
| 本地实际 webhook 请求 | 本地 receiver 记录到 route、direct、GitLab、Sentry 四类外发请求 | 通过 |
| SQLite 数据库 | E2E 使用临时 SQLite DB；schema 位于 `internal/store/store.go` | 通过 |
| 纯 webhook 转发配置 | route model 存储 target URLs、headers、middleware IDs、mode、enabled | 通过 |
| 保留自定义中间件能力 | E2E 中 middleware 修改 headers 和请求上下文，receiver 已验证 | 通过 |
| 移除内网包和企微行为 | 新 Go 项目文件关键字扫描 0 命中 | 通过 |
| 开源依赖形态 | `go.mod` 只使用开源 Go module | 通过 |

## 建议

将 `scripts/local-e2e.sh` 作为 Go 版 OpenHook 的本地发布门禁，并和 `go test ./...`、内网关键字扫描一起执行。
