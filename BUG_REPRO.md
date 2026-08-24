# Bug 复现说明

## Bug 是什么

排程（conservation schedule）加入「验证」（verifying）中间态后三处消费方没有同步：

1. `opsScheduleTransitionTable`（ops_state.go）漏了 `verifying → treating` 边，验证中的作业单无法再流转到处理，卡死在验证态（合法流转被拒）；
2. `OpsScheduleService.StatusSummary` 把 `verifying` 误算进 `planned`，进行中统计偏小、已计划偏大；
3. `OpsAlertEngine.Evaluate` 对 `verifying` 报 `SCHED-UNKNOWN` 未知状态告警，而不是进行中提示。

## 如何触发

- 排程计划 → 流转到 inspecting → 流转到 verifying → 尝试流转到 treating → 被拒绝（状态转换不允许）。
- 存在 verifying 状态的排程时查看状态统计与告警接口。

## 真实错误信息

- 流转接口返回：`operations status transition is not allowed: verifying to treating`（HTTP 400）。
- 告警接口出现 `SCHED-UNKNOWN`（level=error）；统计中 `in_progress` 少了 verifying 的数量。
