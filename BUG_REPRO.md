# Bug 复现说明

## Bug 是什么

1. `OpsStore` 把首个请求的 `ctx` 存进结构体字段 `lastCtx`，后续请求先检查上一个请求的 ctx；上一个请求若带短超时且已过期，后续普通请求会被旧 deadline 污染，报 `context deadline exceeded`，服务需重启才恢复。
2. 导出接口（GET /api/ops/export）用 `context.Background()` 调 store，且循环内不检查取消；归档接口（POST /api/ops/archive/{id}）同样用 `context.Background()` 调 store，取消请求后操作仍会跑完。

## 如何触发

- 先发一个带 30ms 超时的导出/查询请求，等其超时后，再发任何普通请求 → 立刻报超时错误。
- 导出或归档过程中取消请求 → 操作继续执行完。

## 真实错误信息

- 后续请求报：`context deadline exceeded`。
- `-race` 运行真实服务时 worker 与 heartbeat 对 `ops_store.go` 的 `lastCtx` 读写存在 DATA RACE。
