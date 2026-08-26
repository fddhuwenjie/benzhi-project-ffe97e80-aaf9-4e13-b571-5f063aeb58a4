# 语守方言语料发布治理服务

本项目是面向濒危方言田野资料的发布治理 HTTP 服务。它将参与者知情同意、录音材料登记、敏感片段脱敏、社区代表复核、发布核准和只读清单封存组织为一条可追溯流程。

服务仅提供版本化 JSON API，不包含浏览器页面或命令行业务界面。业务数据、跨重启幂等响应、审计事件和封存清单均保存在本地 SQLite 数据库中。

## 构建与测试

```bash
go build ./cmd/server
go test ./...
```

## 运行

默认监听高位回环地址 `127.0.0.1:19081`，默认数据库文件为当前目录下的 `dialect-release.db`：

```bash
go run ./cmd/server
```

可通过 `-addr` 指定其他回环地址，通过 `-db` 指定 SQLite 文件：

```bash
go run ./cmd/server -addr=127.0.0.1:19181 -db=./data/release.db
```

未显式提供 `-addr` 时，也可以设置 `PORT`。例如 `PORT=19182` 会绑定 `127.0.0.1:19182`。服务拒绝非回环监听地址，避免默认对外暴露。

## 有界自检

以下命令会创建临时数据库、启动真实 HTTP 服务，依次完成建档、授权、材料、脱敏、社区退回、补正重审、复核通过、核准封存、时间线查询、摘要校验和封存后拒写，然后主动关闭并退出：

```bash
go run ./cmd/server -addr=127.0.0.1:19081 -self-check
```

## API 流程

所有写请求使用 `application/json`，并携带唯一 `request_id`、当前 `expected_revision` 和操作人 `actor`。相同 `request_id` 会返回首次成功提交的原始状态码与响应体；错误的 `expected_revision` 会返回冲突。

主要端点如下：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `POST` | `/api/v1/cases` | 创建发布个案 |
| `GET` | `/api/v1/cases` | 按可选 `status` 查询个案 |
| `GET` | `/api/v1/cases/{caseID}` | 查询个案详情与当前修订 |
| `POST` | `/api/v1/cases/{caseID}/consents` | 登记参与者授权 |
| `PATCH` | `/api/v1/cases/{caseID}/consents/{consentID}/withdraw` | 撤回尚未进入材料阶段的授权 |
| `POST` | `/api/v1/cases/{caseID}/assets` | 登记录音材料及内容摘要 |
| `POST` | `/api/v1/cases/{caseID}/assets/{assetID}/findings` | 登记敏感片段及处置 |
| `PATCH` | `/api/v1/cases/{caseID}/findings/{findingID}` | 关闭敏感性发现 |
| `POST` | `/api/v1/cases/{caseID}/submit` | 提交社区代表复核 |
| `POST` | `/api/v1/cases/{caseID}/reviews` | 记录通过或结构化退回意见 |
| `POST` | `/api/v1/cases/{caseID}/approve` | 发布核准并永久封存 |
| `GET` | `/api/v1/cases/{caseID}/timeline` | 查询完整审计时间线 |
| `GET` | `/api/v1/cases/{caseID}/manifest` | 查询并重新校验封存清单 |

健康检查和就绪探测分别为 `GET /healthz` 与 `GET /readyz`。请求体上限为 1 MiB，未知 JSON 字段会被拒绝。
