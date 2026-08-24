# Bug 复现说明

## Bug 是什么

批量完成现场作业会话接口（POST /api/ops/sessions/complete）存在死锁：`OpsSessionManager.BulkComplete` 把 `wg.Add(1)` 写在 goroutine 内部，且 `results` 是无缓冲 channel、消费端在 `wg.Wait()` 之后才读。请求上下文一旦被取消（客户端断连/网关超时），worker 走 `ctx.Done()` 分支执行 `results <- ctx.Err()`，因无人读取而永久阻塞，`wg.Wait()` 也永久等待，handler 永不返回；每个被取消的请求都会留下阻塞 goroutine，越积越多。此外未缓冲 channel 在部分失败提前返回时还会触发对已关闭 channel 的 send panic。

## 如何触发

- 打开若干会话后调用批量完成接口，在请求处理过程中取消请求（客户端断开 / 上游超时）。
- 批量中混入不存在的会话 id 且其余会话状态异常时，提前返回路径会留下未关闭的 channel。

## 真实错误信息

- goroutine dump：多个 worker 阻塞在 `ops_sessions.go` 的 `results <- ctx.Err()`（`chan send`），handler 阻塞在 `sync.WaitGroup.Wait`，请求永不返回。
- 部分失败提前返回时：`send on closed channel` panic。
