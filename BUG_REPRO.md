# Bug 复现说明

## Bug 是什么

批量流转作业单状态接口（POST /api/ops/batch/apply）存在三处问题：

1. work unit 在循环体内用 `defer` 释放，堆积到函数末尾才归还；批量条数超过单位上限（默认 8）时，第 9 条在 `<-p.units` 处永久阻塞，接口卡死无响应；
2. 非法状态流转（如 queued→paused）被静默跳过并计为 skipped，错误不向调用方暴露；
3. `OpsManifestRegistry.Release` 只查存在性不删除，批次结束后 manifest 一直残留；HTTP 响应只回 `{"ok":true}`，丢失 batch_id / applied 等结果。

## 如何触发

- 批量提交超过 8 条记录 → 请求卡死（goroutine 阻塞在 `<-p.units`）。
- 批量中带非法流转目标 → 接口返回成功但 skipped 计数，无错误信息。
- 任意一次批量后查询批次清单 → active 数不清零。

## 真实错误信息

- 卡死时 goroutine dump：`panic: test timed out after 3s`，主 goroutine 阻塞在 `OpsBatchProcessor.Apply` 的 select（等待 work unit）。
