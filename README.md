# Museum Conservation Service

museum-conservation-service HTTP 服务，提供文物保养作业记录（work order）管理、状态流转、
保养备注、现场作业会话、巡检排程、统计快照、导出与归档等能力，并附带健康检查和静态页面。

## Quick Start

从项目根目录执行：

    cp .env.example .env
    cd backend
    go test ./...
    go build ./...
    go run .

默认监听 PORT 指定的端口，未设置时为 8080。健康检查：GET /health。

## API

### 基础接口

- GET /health 健康检查
- GET /api/artifacts 文物列表
- POST /api/artifacts/status 更新文物状态（stable、watch、treatment）

### 作业记录（ops records）

- GET /api/ops/records 查询作业记录（subject / status / priority / owner / page / page_size）
- POST /api/ops/records 新建作业记录（id、subject、owner、priority、status、labels）
- GET /api/ops/records/{id} 查询单条记录
- POST /api/ops/records/{id}/transition 状态流转（expected_revision、target_status、actor）
- GET /api/ops/records/{id}/audit 审计轨迹
- GET /api/ops/records/{id}/notes 备注列表
- POST /api/ops/records/{id}/notes 添加备注（author、text）

### 统计与监控

- GET /api/ops/snapshot 统计快照（缓存优先）
- GET /api/ops/dashboard 仪表盘（快照 + 热门记录 + 搜索命中数）
- GET /api/ops/metrics 滚动延迟指标
- GET /api/ops/rules 规则列表（severity / terminal 过滤）

### 作业会话与批量

- POST /api/ops/sessions 打开现场作业会话（record_id、owner）
- POST /api/ops/sessions/complete 批量完成会话（ids）
- GET /api/ops/sessions 会话列表
- POST /api/ops/batch/apply 批量状态流转（owner、from_status、to_status、ids）
- GET /api/ops/manifests 活跃批次清单数

### 排程与告警

- GET /api/ops/schedule 排程列表
- POST /api/ops/schedule 新建排程（record_id、priority）
- POST /api/ops/schedule/{id}/move 排程流转（target：inspecting / treating / done / cancelled）
- GET /api/ops/alerts 告警列表

### 导出与归档

- GET /api/ops/export 导出作业记录文本（status / limit）
- POST /api/ops/archive/{id} 归档已关闭记录

## Layout

    .
    ├── backend/               Go 模块、源码、静态资源
    │   ├── go.mod
    │   ├── *.go
    │   ├── web/               页面资源
    │   └── Dockerfile
    ├── database/README.md     持久化说明
    ├── output/verification.md 验证记录
    └── runtime_smoke.json     启动契约

backend/web.go 使用 go:embed 嵌入 backend/web/index.html 和 backend/web/app.js。

## Verification

验证命令和真实启动结果记录在 output/verification.md。
