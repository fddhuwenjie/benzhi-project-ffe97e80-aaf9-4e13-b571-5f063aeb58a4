# BENZHI_README

## 项目说明
- 项目：benzhi-project-ffe97e80-aaf9-4e13-b571-5f063aeb58a4
- 项目用途：实现了面向濒危方言田野资料的发布治理 HTTP 服务，以 SQLite 事务串联授权、录音材料、敏感片段处置、社区退回复核、发布核准、确定性清单封存和完整审计查询。
- Go 工具链：`golang:1.23.0`
- 前端工具链：无

## 项目描述
- 项目名称：语守方言语料发布治理服务
- 项目介绍：面向濒危方言田野项目的语料发布治理 HTTP 服务，将采集对象的知情同意范围、录音材料、敏感片段脱敏、社区代表复核与最终发布清单串成一个可追溯状态闭环。项目按 standard 档规划，目标约 2300 行真实生产 Go 代码和至少 22 个生产 Go 文件，不引入前端或命令行操作界面。
- 项目概述：面向濒危方言田野项目的语料发布治理 HTTP 服务，将采集对象的知情同意范围、录音材料、敏感片段脱敏、社区代表复核与最终发布清单串成一个可追溯状态闭环。项目按 standard 档规划，目标约 2300 行真实生产 Go 代码和至少 22 个生产 Go 文件，不引入前端或命令行操作界面。
- 核心工作流：资料管理员创建语料发布个案并登记参与者授权条款，补齐录音材料及摘要后逐段登记敏感性发现与脱敏处置；个案提交社区代表复核，退回时回到脱敏处理中补正并重新提交，通过后由发布审核人员核准，系统生成带摘要的只读发布清单并将个案从 DRAFT、CONSENT_READY、MATERIAL_REGISTERED、REDACTION_REVIEW、STEWARD_REVIEW、APPROVED 推进到 SEALED。
- 对外接口：提供版本化 JSON HTTP API：以 /api/v1/cases 为公开入口完成建档、授权登记、材料登记、敏感片段处置、社区复核、发布核准、封存清单及审计查询；所有写请求携带 request_id 和 expected_revision。服务支持 -addr=127.0.0.1:<port>，默认监听 127.0.0.1:19081，也可读取 PORT 并绑定 127.0.0.1:<PORT>，不得默认绑定 0.0.0.0 或常见低位端口。

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...

cd /app && GOTOOLCHAIN=local go run ./cmd/server -addr=127.0.0.1:19081 -self-check

cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh

./build_benzhi_docker.sh benzhi-project-ffe97e80-aaf9-4e13-b571-5f063aeb58a4-amd64 linux/amd64

./build_benzhi_docker.sh benzhi-project-ffe97e80-aaf9-4e13-b571-5f063aeb58a4-arm64 linux/arm64

docker run -it benzhi-project-ffe97e80-aaf9-4e13-b571-5f063aeb58a4-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -addr=127.0.0.1:19081 -self-check`
