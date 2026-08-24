# Bug 复现说明

## Bug 是什么

服务长时间运行内存与后台协程持续增长（重启即恢复），三处泄漏来源：

1. `OpsTelemetry.Record` 只 append 不按 `limit` 截断，50ms 心跳持续喂入导致采样无限增长；`Series` 还返回内部切片引用；
2. `OpsAudit.Prune` 是空函数体，worker 每次调用等于没调，审计事件永久累积；
3. `OpsHeartbeat.Stop` 只置 running=false 不发停止信号，`loop` 用 `context.Background` 永不退出，每次 Start→Stop 泄漏一个协程并持续写入 telemetry。

## 如何触发

- 服务跑一段时间（或多次 Start/Stop Heartbeat），观察内存与协程数增长。
- 持续产生遥测采样与审计事件后，查看 `Count()` 超过上限仍继续增长。

## 真实错误信息

- 无报错，表现为 `runtime.NumGoroutine` 与内存随运行时间线性增长；多次 Start→Stop 后协程数不回落到基线；`Prune(keep)` 后事件数不减少。
