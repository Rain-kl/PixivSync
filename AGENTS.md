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
- `push-notification`：在添加或修改系统通知推送事件、修改消息推送底层设计、调用统一触发器投递消息、或开发带消息推送功能的业务功能时使用。

## 严格遵循事项 (Guardrails)

- 切勿删除 `frontend/node_modules`；如果需要刷新依赖，请使用 `pnpm install` 重新安装。
- 保持 `internal/util/` 绝对纯净且不引入任何框架。禁止从 `internal/util/` 及其子包中导入 Gin、GORM、sessions 等 HTTP/Web/数据库相关框架包（例如，Web 会话选项已收敛至 `internal/apps/oauth/session.go`）。
- 编写测试用例时，禁止使用硬编码的相对路径（如 `"uploads/test_cache"`）在源码目录下创建临时测试目录，必须统一使用 Go 内置的 `t.TempDir()` 以避免污染源码目录。
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
- `support-files/`：部署 and 数据库辅助文件。
- `bin/`：本地编译生成的二进制可执行文件。
- `data/`：本地运行时数据文件目录（如 PostgreSQL、Redis 数据等）。
- `uploads/`：本地文件上传存储目录。

后端目录：

- `internal/cmd/`：用于 API、worker、scheduler、root init 的 Cobra 命令。
- `internal/config/`：Viper 加载和配置结构体。运行时代码应使用 `config.Config.<Section>.<Field>`。
- `internal/router/`：唯一的 HTTP 路由注册点。
- `internal/apps/`：按功能（Feature-based）组织的 HTTP Handler、中间件、内部服务与模块逻辑。移除全局 service 层，模块内部业务逻辑（如验证码业务逻辑管理器 `internal/apps/cap/manager.go`）均收敛于各自模块中；管理端模块位于 `internal/apps/admin/`。
- `internal/apps/upload/`：上传记录、文件访问控制、本地/S3 文件响应、下载及图片 WebP 压缩。业务应复用这些入口，不直接操作底层文件。
- `internal/model/`：GORM 实体和模型级业务方法。
- `internal/db/`：PostgreSQL、Redis、ClickHouse、GORM 日志、ID 生成和 goose SQL 迁移的布线。
- `internal/diskcache/`：平台级磁盘字节缓存，通过 `diskcache.GetGlobalCache()` 提供 TTL、最大空间限制、LRU 淘汰、清空、状态统计和配置热更新。写入时使用 `DefaultExpiration`（全局默认 TTL）、正数 `time.Duration`（业务 TTL）或 `NoExpiration`（无 TTL，仍受空间限制和 LRU 淘汰）。
- `internal/storage/`：S3 兼容对象存储适配，提供对象上传、读取、删除、CDN/代理读取及远端对象本地缓存。
- `internal/task/`：Asynq 任务框架；参见 `new-async-task` 了解变更。
- `internal/common/`：共享的通用模型及响应（如 `internal/common/response`）、绑定（bind）、常量以及通用错误。
- `internal/util/`：纯底层工具包，无任何 HTTP/数据库框架依赖。
- `internal/listener/`：事件监听器和消息/Webhook 消费者。
- `internal/otel_trace/`：链路追踪（tracing）助手。
- `internal/testhelper/`：后端测试共享辅助能力。
- `internal/buildinfo/`：暴露在发布/构建工作流中注入的元数据（如版本号、编译时间等）。

公共底层包 (`pkg/`)：
- `pkg/cache/disk/`：纯底层的通用本地磁盘缓存引擎。
- `pkg/cap/`：底层的通用验证码验证和生成库。
- `pkg/httppool/`：管理全局共享且经过优化的 HTTP 传输客户端及连接池，集成 OTel 链路追踪。
- `pkg/logger/`：Zap 和 OTel 日志助手。
- `pkg/push/`：推送渠道客户端集成（Lark/Telegram/Email）。
- `pkg/mail/`：邮件发送客户端。
- `pkg/trace/`：OpenTelemetry 链路追踪配置。
- `pkg/util/`：纯底层无副作用的系统工具（Crypto/Password/UUID）。

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
- 成功时统一通过导入 `"github.com/Rain-kl/Wavelet/internal/common/response"` 使用 `response.OK(data)` 或 `response.OKNil()` 返回。
- 失败时统一通过 `response.AbortWithError(c, code, msg)` 返回（这会自动将业务错误和状态码挂载到 Context 并中断请求，全局中间件捕获后会将其完整记录到 OpenTelemetry Trace/Jaeger，并格式化输出 JSON）。
- API 响应的外层结构必须为 `{ "error_msg": "", "data": ... }`。
- 分页响应在 `data` 下使用 `{ "total": 0, "results": [] }`。
- 每个 HTTP API 都需要有完整的 Swagger 注释；在 API 变更后运行 `make swagger`。

错误处理与日志:

- 任何关键错误在被吞掉、转换为通用响应，或由后台 worker 忽略之前，都必须通过 `pkg/logger` 打印日志。
- 禁止用 `_ = ...` 静默丢弃重要错误。如果某个错误因为 best-effort 操作或确认无害而需要忽略，必须添加简短注释说明原因。
- Handler 可以返回对用户安全的错误信息，但如果底层运行错误对生产问题排查有价值，仍然必须记录日志。
- 避免重复刷日志：在真正处理或抑制错误的边界记录一次，然后返回或响应。

路由与模块：

- 仅在 `internal/router/router.go` 中作为统一高层入口进行路由分发委派，不允许在 `router.go` 中直接挂载业务 Handler。
- 关于所有的路由归属划分、接口开发隔离防线以及详细的注册和开发步骤，请直接阅读并严格遵循 [new-api](file:///Users/ryan/DEV/Go/Wavelet/.agent/skills/new-api/SKILL.md) 技能。

中间件：

- 全局中间件属于路由设置：`gin.Recovery()`、`otelgin.Middleware()`、日志中间件 and session 中间件。
- 对于登录路由组，使用 `oauth.LoginRequired()`。
- 对于管理路由组，使用 `admin.LoginAdminRequired()`。

配置管理：

- 运行时代码从 `config.Config` 中读取配置，绝对不要直接从 `os.Getenv()` 中读取。
- 当添加配置时，同时更新 `config.example.yaml` and `internal/config/model.go`。

数据库操作：

- 简单查询可以直接从 model 层使用 GORM。
- 管理员代码应首选 `db.DB(ctx)` 以获得链路追踪感知的 DB 访问。
- 不要在 Handler 中放置复杂的 SQL；将其移至 `internal/model/` 或模块内的业务服务层（如 `internal/apps/<module>/service.go` 或 `logics.go`）。
- 在 `internal/db/migrator/goose/` 下使用 goose SQL 迁移；不要添加基于 GORM AutoMigrate 的 Schema 升级。
- 不要创建物理数据库外键。改为关系字段添加显式索引。
- 数据库默认值必须与 Go 模型零值（`nil`、`0`、`false`、`""`）匹配，以避免意外的插入。

严格依赖防线：

- `internal/util/` 及其子包必须保持无框架依赖。
- 不要从 `internal/util/` 中导入 `github.com/gin-gonic/gin`、`gorm.io/gorm`、`github.com/gin-contrib/sessions` 或 HTTP 中间件/框架包。
- 如果实用工具逻辑需要 web 胶水，请将纯验证/计算保留在 `internal/util/` 中，并将 Gin 中间件/响应处理放在 `internal/apps/` 中。

新增接口与模块开发工作流：

- 关于自定义业务接口（如 Admin/User/Custom 模块等）的详细包职责、文件结构和核心开发步骤，请直接阅读并严格遵循 [new-api](file:///Users/ryan/DEV/Go/Wavelet/.agent/skills/new-api/SKILL.md) 技能。

### 前端规则

在进行任何 Next.js 工作之前，请在 `node_modules/next/dist/docs/` 中找到并阅读相关文档。您的训练数据已过时 —— 这些文档是唯一的真理来源。

请直接查看并参考项目提供的示例和 Demo 代码：[frontend/app/(main)/admin/demo](file:///Users/ryan/DEV/Go/Wavelet/frontend/app/(main)/admin/demo)。

样式规范：

- shadcn/ui 基础组件应该使用它们的 `variant` 系统和全局 CSS 变量。当组件的变体（variant）应该拥有某种外观时，不要在业务 `className` 中硬编码颜色、背景或阴影。
- 如果现有的变体不足以满足需求，请扩展 shadcn/ui 组件的变体，而不是硬编码一次性的颜色。

页面标题栏规范 (新人开发与重构必读)：

- **容器与对齐机制**：
    - 标题容器统一使用 `flex items-center gap-2`。如果右侧有操作按钮（如“新增”、“刷新”），请使用 `justify-between` 布局让操作区与标题双向分布。
    - 为了确保所有页面在进入/切换时，顶部的呼吸感和视觉高度完全一致，页面最外层容器**必须**统一使用 `py-6 px-1` 或 `py-6` 进行上边距对齐。
- **图标标准**：图标作为视觉辅助点缀，**必须**直接嵌套在标题容器中，直接使用 Lucide 图标组件，样式大小限制为 `size-5 text-primary`。**严禁**为图标包裹任何背景小卡片、圆角边框或额外的修饰容器。
- **标题文字标准**：标题文字使用且仅使用 `h1 className="text-2xl font-semibold tracking-tight"`。不要自行定义字号、字量（如使用 `font-bold`）或添加任何渐变色，保持整个系统的字形规范化。
- **Tabs 模块化与文件拆分规范**：凡是带有多个 Tab 页切换的复杂页面，**禁止**将所有 Tab 的渲染逻辑堆积在同一个主文件内。每个 Tab 的具体渲染内容必须单独拆分为独立的 React 组件文件（如 `tabs/events-tab.tsx`）。主页面文件应该仅用于导入子组件、注册 Tabs 触发器以及管理 Tab 的切换激活状态。这有利于防止单文件过大（避免单文件行数超过 600 行限制），并大幅度提高代码的可读性与编译维护效率。
- **扁平化结构与避免冗余中间件**：为了消除无意义的“中间代理文件”，所有作为路由物理入口的 Tabs 状态维护、骨架及外层布局代码，**必须**直接定义在 Next.js 的 `app/` 页面文件（即 `page.tsx`）中。禁止在 `page.tsx` 中仅写一个单纯的 `<AnotherComponent />` 转发，而在外部新建一个同名中转容器。
- **复杂度驱动的组件拆分规范**：组件的拆分不应局限于“跨页面复用”。当一个路由页面的复杂度变高时（如渲染逻辑膨胀、存在大型嵌套弹窗或多层状态管理，如单文件代码行数超过 600 行），必须主动将其拆分为子组件以维持单文件的高可读性与低耦合度。拆分时遵循就近原则：特定于该路由且不复用的子组件应放置在最邻近该路由的特征目录（Feature Folder，如 `components/` 局部文件夹）中；只有真正具备跨页面复用价值的通用业务/基础 UI 组件才应存放在全局 `components/` 共享目录下。
    - **最佳实践标杆案例（数据管理 `/admin/database`）**：
      该页面由于整合了“运行状态概览”、“物理表网格浏览器”、“磁盘缓存管理”和“SQL 交互控台”多个复杂大区块，重构前单文件接近 1000 行。
      重构后，主页面 `page.tsx` 仅做高级页面骨架与排版排布，维护全局刷新机制与终端视图切换；而“数据表浏览器 (`table-browser.tsx`)”、“缓存管理 (`cache-manager.tsx`)”与“SQL 终端 (`sql-console.tsx`)”等独立高状态密度区块均被抽离为局部子组件，存放在 `frontend/app/(main)/admin/database/components/`。这保证了代码结构层次清晰、单文件小巧好维护。所有复杂页面的新开发和重构必须遵循此模式。

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

