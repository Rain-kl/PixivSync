---
name: "new-async-task"
description: "项目专用：当新增或修改 Asynq 异步任务、后台任务、定时任务、任务元数据、TaskHandler、TaskParam、PayloadValidator、AppendLog、任务重试、任务执行记录或 Admin 任务 API 时必须使用。本技能按项目约束指导常量定义、处理器实现、统一注册、Worker 路由、Cron 配置、Swagger、测试和 code-check 验证。"
---

# 异步任务开发

本技能只覆盖  Asynq 任务工作流。开始前先读仓库根目录 `AGENTS.md`，并遵守其中的项目级规则：HTTP 路由只在 `internal/router/router.go` 注册、API 变更后运行 `make swagger`、提交前运行 `make code-check`、不要删除 `frontend/node_modules`、`internal/util/` 不引入框架依赖。

## 先定位真实链路

新增或修改任务前，先快速查看这些文件，确认当前实现没有漂移：

- `internal/task/handler.go`: `TaskHandler`、`TaskResult`、可选 `PayloadValidator`。
- `internal/task/constants.go`: Asynq 任务类型常量、Admin 任务类型常量、`TaskMeta`、`TaskParam`、`DispatchableTasks`。
- `internal/task/executor.go`: `RegisterHandler`、`ValidateAndNormalizePayload`、`DispatchTask`、`RetryTask`、`ProcessTask`、`AppendLog`。
- `internal/task/handlers/register.go`: 内置 handler 的统一注册点，Admin API 和 Worker 都依赖它。
- `internal/task/worker/worker.go`: Asynq mux 路由和队列配置。
- `internal/task/scheduler/scheduler.go`: Cron 调度。
- `internal/apps/admin/task/routers.go`: Admin 下发、查询、详情、重试 API。
- 现有参考：`internal/apps/upload/tasks.go`（无参数任务）、`internal/apps/user/tasks.go`（带参数任务）。

当前任务执行链路：

```text
Admin dispatch -> ValidateAndNormalizePayload -> DispatchTask
  -> Asynq Redis queue -> worker mux -> ProcessTask
  -> registered TaskHandler.Execute -> TaskExecution status/log/result
```

## 修改检查清单

按任务影响面选择对应步骤。不要只改其中一条链路。

> 需要可复制的代码模板时，阅读 [references/CODE-EXAMPLES.md](references/CODE-EXAMPLES.md)。那里包含任务常量、无参数 handler、带参数 `PayloadValidator`、统一注册、Worker 路由、Cron 配置和测试示例。

1. 定义任务元数据。
   - 在 `internal/task/constants.go` 添加 Asynq task type 常量，例如 `upload:cleanup_unused`。
   - 添加 Admin 可下发 task type 常量，例如 `cleanup_unused_uploads`。
   - 在 `DispatchableTasks` 添加 `TaskMeta`，设置 `AsynqTask`、`Name`、`Description`、`MaxRetry`、`Queue`、`Retryable`。
   - 有参数任务在 `Params` 中描述前端表单字段。`TaskParam.Name` 必须与 payload JSON tag 对齐。

2. 实现 handler。
   - 优先放在对应业务模块的 `internal/apps/<module>/tasks.go`。
   - handler 必须实现 `task.TaskHandler`。
   - 带参数任务定义 payload struct，并实现 `task.PayloadValidator` 做服务端校验和标准化。
   - `Execute` 中仍要解析 payload，因为 Scheduler、重试或其他入口不一定经过 Admin 校验。
   - 不要在 handler 中写复杂 SQL；复杂查询放到 `internal/model/` 或 `internal/service/`。
   - 新增 Go 文件后检查 license header；必要时运行 `make license`。

3. 统一注册 handler。
   - 在 `internal/task/handlers/register.go` 导入业务模块并调用 `task.RegisterHandler(asynqTaskType, handler)`。
   - 这里是 Admin payload 校验和 Worker 执行共同依赖的注册点。不要只在 worker 包里注册。

4. 注册 Worker 路由。
   - 在 `internal/task/worker/worker.go` 的 Asynq mux 中添加 `mux.HandleFunc(task.YourAsynqTask, task.ProcessTask)`。
   - 所有业务任务都应交给 `task.ProcessTask`，由 executor 根据 task type 分发到 handler。

5. 如需 Cron 调度，补齐配置链路。
   - 在 `internal/task/scheduler/scheduler.go` 注册 cron。
   - 在 `internal/config/model.go` 添加 scheduler config 字段。
   - 在 `config.example.yaml` 添加对应配置项。
   - runtime 代码从 `config.Config` 读取配置，不直接读环境变量。
   - Scheduler 直接入队的任务可能没有 Admin 创建的 `TaskExecution` 记录；需要可见执行记录时，优先通过 Admin dispatch 触发。

6. 如改动 Admin API。
   - handler 放在 `internal/apps/admin/<module>/` 或现有 Admin task 模块内。
   - 路由只在 `internal/router/router.go` 注册。
   - 响应保持 `{ "error_msg": "", "data": ... }`，分页保持 `{ "total": 0, "results": [] }`。
   - 补完整 Swagger 注释并运行 `make swagger`。

## Handler 模式

约定：

- `ValidatePayload` 是 Admin 下发时的服务端校验入口，返回值会作为标准化 payload 存库和入队。
- `TaskParam` 只是前端表单元数据，不代替服务端校验。
- 成功返回 `&task.TaskResult{Message: "...", Detail: "..."}`；失败返回 `nil, fmt.Errorf("...")`，由 `ProcessTask` 标记失败并交给 Asynq 重试。
- 错误要返回给框架，不在 handler 内吞掉；可继续的单条失败可用 `AppendLog` 记录后继续处理。

> 在新增无参数任务、带参数任务或 `PayloadValidator` 时，阅读 [references/CODE-EXAMPLES.md](references/CODE-EXAMPLES.md) 的 Handler 示例。

## AppendLog 规则

在 `TaskHandler.Execute` 内用 `task.AppendLog(ctx, format, args...)` 写任务日志。

- 任务开始、参数摘要、批次进度、关键状态、可继续错误、完成摘要适合记录。
- 避免对大循环中的每条记录都写日志；每次 `AppendLog` 都可能触发一次数据库更新。
- 如果上下文中没有 taskID，`AppendLog` 会降级为普通应用日志，不应额外兜底报错。

## 重试规则

该项目有两层重试：

- Asynq 自动重试：`ProcessTask` 返回 error 后按入队的 `MaxRetry` 处理。
- Admin 手动重试：`POST /api/v1/admin/tasks/executions/:id/retry` 创建新的 `TaskExecution`，要求原任务状态为 failed、`Retryable=true`、`RetryCount < MaxRetry`。

修改重试语义时同时检查 `internal/task/executor.go`、`internal/model/task_execution.go`、`internal/apps/admin/task/routers.go` 和前端任务执行列表。

## Frontend/Admin 任务 UI

只有任务元数据变化时，通常不需要写新页面；现有 Admin UI 会根据 `DispatchableTasks` 和 `Params` 动态渲染。

若确实要改前端：

- 读 shadcn skill。
- 业务组件优先放在 `frontend/components/common/admin/`。
- API 访问走 `frontend/lib/services/` 的 service class 和 `services` export。
- 不使用 `any`。
- 页面根容器保持 `w-full`，不要加页面级 `max-w-*`。

## 验证

根据改动范围运行最小有效验证，最后提交前必须运行项目门禁。

- Handler 单测：覆盖成功、失败、日志关键路径；带参数任务覆盖 `ValidatePayload` 成功、空 payload、非法 JSON、缺失必填、标准化。
- Admin dispatch 单测：合法 payload 返回成功，非法 payload 返回 400 且错误清晰。
- Retry 单测：failed 且可重试能创建新执行记录；非 failed、`Retryable=false`、超过 `MaxRetry` 都拒绝。
- 目标包测试示例：

```bash
go test ./internal/task ./internal/apps/admin/task ./internal/apps/<module>
```

- API 改动后：

```bash
make swagger
```

- 提交前：

```bash
make code-check
```

如涉及前端或整体构建，补跑 `make build-test`。测试中需要 Redis/Asynq 时，可用现有测试模式或 `miniredis`；不要为了测试便利把 `internal/task` 反向塞进通用 util/testhelper，避免 import cycle。

## 相关 Skills

- Go 错误处理：在设计 handler 返回错误、包装底层错误或避免“记录并返回”时，参见 [go-error-handling](../go-error-handling/SKILL.md)。
- Go 测试：在编写 handler、dispatch 或 retry 单测时，参见 [go-testing](../go-testing/SKILL.md)。
- Go context：在任务业务逻辑传播取消、超时或 request scoped 值时，参见 [go-context](../go-context/SKILL.md)。
- Go logging：在决定任务日志、应用日志和日志级别边界时，参见 [go-logging](../go-logging/SKILL.md)。
- shadcn：在修改 Admin 任务 UI 时，参见 [shadcn](../shadcn/SKILL.md)。
