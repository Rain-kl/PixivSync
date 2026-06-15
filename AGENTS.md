# AGENTS.md — 项目AI助手工作操作手册

本文件面向 AI 开发助手，定义其职责与操作规范。

## Git 提交规范指南

### 提交信息基本格式

每次提交更改时，应当使用以下提交格式:

```text
<type>(<scope>): <subject>

<body>
```

* **Type**: 提交类型（例如 `feat`, `fix`, `refactor`, `perf`, `docs`, `chore` 等）。
* **Scope** (可选): 影响的范围（例如 `api`, `frontend`, `auth`, `mcp` 等）。
* **Subject**: 简短的一句话描述变更。
* **Body** (可选): 详细的说明，多行叙述。

## 务必阅读匹配的 Skill

- `new-api`：在添加或修改自定义业务 API、Handler、服务层逻辑或注册自定义端点时使用。
- `new-async-task`：在添加或修改 Asynq 任务、定时任务时使用。
- `new-setting`：在添加或修改基于数据库的系统/业务/公开设置、`/admin/system` 参数或 `/admin/settings` 图形化设置时使用。
- `database-migration`：在数据库升级流程时使用。
- Go skills：使用针对性的 `go-*` skills 来获取 Go 实现细节，如测试、错误处理、包、Context、并发、日志、文档和审查。
- `shadcn`：在添加、修改或组合 shadcn/ui 组件时使用。
- `code-review-skill`：在提交 PR 之前使用，检查代码质量、样式、潜在错误和最佳实践。

## 严格遵循事项 (Guardrails)

- 切勿删除 `frontend/node_modules`；如果需要刷新依赖，请使用 `pnpm install` 重新安装。
- 保持 `internal/util/` 不引入任何框架。不要从 `internal/util/` 及其子包中导入 Gin、GORM、sessions 或其他 HTTP/框架包。
- 所有 HTTP 路由仅在 `internal/router/router.go` 中注册。
- 当 API Handler 发生变化时，更新 Swagger 文档（运行 `make swagger`）。
- 在提交更改前运行 `make code-check`。
- 需要缓存或文件管理能力时，必须复用现有平台实现，禁止在业务包中自行创建缓存目录、直接管理缓存文件或重复封装存储后端。

## 项目介绍

### 技术栈

- 后端：Go 1.25+、Gin、GORM、PostgreSQL、可选 ClickHouse、Redis、Asynq、Cobra、Viper、Swaggo、OpenTelemetry、Zap、AWS SDK v2、Snowflake IDs。
- 前端：Next.js App Router、TypeScript、Tailwind CSS、pnpm、shadcn/ui。

### 目录结构与平台能力

顶层目录：

- `main.go`：程序入口，委派给 `internal/cmd`。
- `config.example.yaml`：已提交的配置模板。在添加配置字段时保持更新。
- `config.yaml`：本地运行时的配置文件。不要将其作为已提交的源码提交。
- `docker/`：集成的、仅前端的和仅后端的 Dockerfile。
- `docs/`：自动生成的 Swagger 文档。请勿手动编辑生成的文件。
- `frontend/`：Next.js 应用。
- `internal/`：私有 Go 后端代码。
- `pkg/`：公共 Go 库/工具包（留作扩展或存放不依赖特定业务的通用代码）。
- `scripts/`：本地和 CI 辅助脚本。
- `support-files/`：部署和数据库辅助文件。
- `bin/`：本地编译生成的二进制可执行文件。
- `data/`：本地运行时数据文件目录（如 PostgreSQL、Redis 数据等）。
- `uploads/`：本地文件上传存储目录。

后端目录：

- `internal/cmd/`：用于 API、worker、scheduler、root init 的 Cobra 命令。
- `internal/config/`：Viper 加载和配置结构体。运行时代码应使用 `config.Config.<Section>.<Field>`。
- `internal/router/`：唯一的 HTTP 路由注册点。
- `internal/apps/`：按领域组织的 HTTP Handler 和模块逻辑；管理端模块位于 `internal/apps/admin/`。
- `internal/apps/upload/`：上传记录、文件访问控制、本地/S3 文件响应、下载及图片 WebP 压缩。业务应复用这些入口，不直接操作底层文件。
- `internal/model/`：GORM 实体和模型级业务方法。
- `internal/db/`：PostgreSQL、Redis、ClickHouse、GORM 日志、ID 生成和 goose SQL 迁移的布线。
- `internal/diskcache/`：平台级磁盘字节缓存，通过 `diskcache.GetGlobalCache()` 提供 TTL、最大空间限制、LRU 淘汰、清空、状态统计和配置热更新。写入时使用 `DefaultExpiration`（全局默认 TTL）、正数 `time.Duration`（业务 TTL）或 `NoExpiration`（无 TTL，仍受空间限制和 LRU 淘汰）。
- `internal/storage/`：S3 兼容对象存储适配，提供对象上传、读取、删除、CDN/代理读取及远端对象本地缓存。
- `internal/task/`：Asynq 任务框架；参见 `new-async-task` 了解变更。
- `internal/service/`：当 Handler/Model 层次过于狭窄时使用的复杂业务服务。
- `internal/common/`：共享的响应、绑定（bind）、常量以及通用错误。
- `internal/util/`：纯实用工具，不导入任何框架。
- `internal/logger/`：Zap 和 OTel 日志助手。
- `internal/listener/`：事件监听器和消息/Webhook 消费者。
- `internal/otel_trace/`：链路追踪（tracing）助手。
- `internal/testhelper/`：后端测试共享辅助能力。
- `internal/buildinfo/`：暴露在发布/构建工作流中注入的元数据（如版本号、编译时间等）。
- `internal/httppool/`：管理全局共享且经过优化的 HTTP 传输客户端及连接池，并集成 OTel 链路追踪。

前端目录：

- `frontend/app/`：App Router 页面、路由组、根布局、全局配置。
- `frontend/components/ui/`：shadcn/ui 基础组件。
- `frontend/components/common/`：跨页面的业务组件。
- `frontend/components/layout/`：Header、Sidebar、Footer 等应用布局组件。
- `frontend/components/auth/`、`home/`、`animate-ui/`、`providers/`：特定作用域的 UI 组件。
- `frontend/lib/services/`：基于 `BaseService` 的类型化 API 服务，按业务域拆分并由 `services` 对象统一导出。
- `frontend/contexts/`、`hooks/`、`lib/`、`types/`、`public/`：共享状态、Hook、客户端与实用工具、TypeScript 类型、静态资产。
- `frontend/scripts/`：前端构建和维护脚本。
- `frontend/.next/`、`frontend/out/`、`frontend/node_modules/`：本地生成或安装的产物，不作为业务源码编辑。

重要的公共组件：

- `frontend/components/common/admin/task-manager.tsx`：任务管理和分发入口。
- `frontend/components/common/admin/task-executions.tsx`：任务执行日志和重试 UI。
- `frontend/components/common/admin/task-schedules.tsx`：定时任务管理 UI。
- `frontend/components/common/admin/system.tsx`：系统参数管理。
- `frontend/components/common/admin/files.tsx`：上传文件管理。
- `frontend/components/common/admin/users.tsx`：用户管理。
- `frontend/components/common/admin/access-analytics.tsx`：访问分析与图表展示。
- `frontend/components/common/admin/access-logs.tsx`：访问日志审计 UI。
- `frontend/components/common/admin/app-logs.tsx`：应用日志查看 UI。
- `frontend/components/common/admin/database-manage.tsx`：数据库备份、恢复与管理 UI。
- `frontend/components/common/admin/file-list.tsx`：管理员文件列表管理组件。
- `frontend/components/common/admin/file-stats.tsx`：文件存储状态与统计 UI。
- `frontend/components/common/admin/status.tsx`：系统运行状态与监控 UI。
- `frontend/components/common/admin/storage-config-tab.tsx`：存储策略与配置 tab 页。
- `frontend/components/common/admin/system-logs.tsx`：系统日志查看组件。
- `frontend/components/common/general/manage-pannel.tsx`：通用列表/详情管理器。
- `frontend/components/common/general/password-dialog.tsx`：敏感操作密码确认对话框。
- `frontend/components/common/settings/system-settings.tsx`：管理员图形化系统设置。
- `frontend/components/common/user/file-manager.tsx`：用户端文件管理组件。


## 开发要求

### 后端规则

命名规范：

- Go 包和文件使用小写蛇形命名（lowercase snake case）：如 `auth_source`、`postgres_logger.go`。
- 导出的 Go 标识符使用 PascalCase；未导出的标识符使用 camelCase。
- 请求/响应结构体使用 camelCase 并带有后缀，例如 `listUsersRequest` 和 `listUsersResponse`。
- 错误消息常量是 camelCase 字符串 `const`值，而不是包级别的 `error` 值。
- YAML 配置键使用小写蛇形命名（lowercase snake case）。

Handler 规范：

- Handler 命名为 动词 + 名词，例如 `ListUsers`。
- 使用 `ShouldBindQuery` 或 `ShouldBindJSON` 进行绑定。
- 成功时通过 `util.OK(data)`、`util.OKNil()` 或 `response.RespondSuccess` 返回。
- 失败时通过 `util.Err(msg)` 或 `response.RespondFailure` 返回。
- API 响应的外层结构必须为 `{ "error_msg": "", "data": ... }`。
- 分页响应在 `data` 下使用 `{ "total": 0, "results": [] }`。
- 每个 HTTP API 都需要有完整的 Swagger 注释；在 API 变更后运行 `make swagger`。

错误处理与日志:

- 任何关键错误在被吞掉、转换为通用响应，或由后台 worker 忽略之前，都必须通过 `internal/logger` 打印日志。
- 禁止用 `_ = ...` 静默丢弃重要错误。如果某个错误因为 best-effort 操作或确认无害而需要忽略，必须添加简短注释说明原因。
- Handler 可以返回对用户安全的错误信息，但如果底层运行错误对生产问题排查有价值，仍然必须记录日志。
- 避免重复刷日志：在真正处理或抑制错误的边界记录一次，然后返回或响应。

路由与模块：

- 仅在 `internal/router/router.go` 中注册路由。
- 在 `internal/apps/<module>/` 中，使用：
    - `routers.go` 或 `controllers.go` 作为 HTTP Handler。
    - `middlewares.go` 作为模块特定的中间件。
    - `errs.go` 仅包含字符串错误常量。
    - `constants.go` 包含非错误的业务常量。
- 对于管理（Admin）模块，首选 `internal/apps/admin/<module>/`。
- 如果 Handler 文件超过 600 行、包含复杂的多个步骤逻辑，或混合了独立领域，请将业务逻辑拆分到 `logic.go` 或 `logics.go` 中。保持 `routers.go` 仅用于绑定、调用逻辑和响应。

中间件：

- 全局中间件属于路由设置：`gin.Recovery()`、`otelgin.Middleware()`、日志中间件和 session 中间件。
- 对于登录路由组，使用 `oauth.LoginRequired()`。
- 对于管理路由组，使用 `admin.LoginAdminRequired()`。

配置管理：

- 运行时代码从 `config.Config` 中读取配置，绝对不要直接从 `os.Getenv()` 中读取。
- 当添加配置时，同时更新 `config.example.yaml` 和 `internal/config/model.go`。

数据库操作：

- 简单查询可以直接从 model 层使用 GORM。
- 管理员代码应首选 `db.DB(ctx)` 以获得链路追踪感知的 DB 访问。
- 不要在 Handler 中放置复杂的 SQL；将其移至 `internal/model/` 或 `internal/service/`。
- 在 `internal/db/migrator/goose/` 下使用 goose SQL 迁移；不要添加基于 GORM AutoMigrate 的 Schema 升级。
- 不要创建物理数据库外键。改为关系字段添加显式索引。
- 数据库默认值必须与 Go 模型零值（`nil`、`0`、`false`、`""`）匹配，以避免意外的插入。

严格依赖防线：

- `internal/util/` 及其子包必须保持无框架依赖。
- 不要从 `internal/util/` 中导入 `github.com/gin-gonic/gin`、`gorm.io/gorm`、`github.com/gin-contrib/sessions` 或 HTTP 中间件/框架包。
- 如果实用工具逻辑需要 web 胶水，请将纯验证/计算保留在 `internal/util/` 中，并将 Gin 中间件/响应处理放在 `internal/apps/` 中。

管理（Admin）模块工作流：

1. 在 `internal/model/` 中定义或扩展模型。
2. 在 `internal/db/migrator/goose/` 下添加 goose SQL 迁移。
3. 创建 `internal/apps/admin/<module>/routers.go` 和可选的 `errs.go`。
4. 在 `internal/router/router.go` 中注册路由。
5. 运行 `make swagger`。

### 前端规则

在进行任何 Next.js 工作之前，请在 `node_modules/next/dist/docs/` 中找到并阅读相关文档。您的训练数据已过时 —— 这些文档是唯一的真理来源。

样式规范：

- shadcn/ui 基础组件应该使用它们的 `variant` 系统和全局 CSS 变量。当组件的变体（variant）应该拥有某种外观时，不要在业务 `className` 中硬编码颜色、背景或阴影。
- 如果现有的变体不足以满足需求，请扩展 shadcn/ui 组件的变体，而不是硬编码一次性的颜色。
- 使用 Lucide 图标来满足常见的图标需求。将自定义图标作为命名导出放在 `frontend/components/icons/` 中。

页面宽度：

- 页面根容器必须支持全宽。使用 `w-full`。
- 不要硬编码页面级的最大宽度，如 `max-w-6xl` 或 `max-w-4xl`；主布局（main layout）拥有正常/全宽的限制。

组件放置：

- 跨页面的业务组件属于 `frontend/components/common/`。
- shadcn/ui 原生组件（primitives）属于 `frontend/components/ui/`。
- 特定于路由/页面的组件放在最邻近的特征（feature）目录中。

服务类（Services）：

- 前端 API 访问通过服务类和导出的 `services` 对象进行。
- 新增服务结构如下：

```text
frontend/lib/services/<service-name>/
  types.ts
  <service-name>.service.ts
  index.ts
```

- 服务类继承 `BaseService`，定义 `basePath`，并暴露有类型的静态方法。
- 在 `frontend/lib/services/index.ts` 中注册新服务。
