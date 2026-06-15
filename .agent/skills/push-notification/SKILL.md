---
name: "push-notification"
description: "Wavelet 项目专用：当需要开发或接入新的系统通知推送事件、修改消息推送底层设计、调用统一触发器投递消息、或开发带消息推送功能的业务功能时必须使用。本技能指导元数据声明、触发流程、解耦防线和动态同步机制。"
---

# 新增消息推送与通知事件开发规范

本技能涵盖 Wavelet 的系统通知推送开发规范。开始开发前先阅读仓库根目录 [AGENTS.md](file:///Users/ryan/DEV/Go/Wavelet/AGENTS.md)，遵守项目级核心规则。

---

## 消息推送架构设计 (Architecture)

Wavelet 的消息推送机制采用了**元数据驱动 + 统一触发器 + 异步任务派发**的解耦设计，其分层及职责划分如下：

| 目录/包名 | 职责定位 | 包含内容与设计细节 |
| :--- | :--- | :--- |
| **`pkg/push/`** | 推送基础设施层 | 静态定义、不依赖系统数据库和任何框架。定义了统一接口 `Pusher`、单例 `PusherPool` 和多实现（Lark, Webhook, Email 等），提供配置验证及发送功能。 |
| **`internal/apps/admin/push/`** | 通知服务与后台任务层 | 包含以下核心文件：<br>1. [events.go](file:///Users/ryan/DEV/Go/Wavelet/internal/apps/admin/push/events.go)：定义通知事件的结构模型（`NotificationMessage`, `EventMetadata`）、内置事件的动态注册中心（`BuiltInEvents` 及 `RegisterBuiltInEvent` 函数）以及统一触发器类 `EventTrigger`（包括其底层的派发引擎逻辑）。<br>2. [tasks.go](file:///Users/ryan/DEV/Go/Wavelet/internal/apps/admin/push/tasks.go)：定义 Asynq 后台异步发送任务、处理器 `PushHandler` 及其校验逻辑，并记录推送历史审计。<br>3. [routers.go](file:///Users/ryan/DEV/Go/Wavelet/internal/apps/admin/push/routers.go)：管理端接口，负责获取事件配置列表和更新配置。 |
| **`internal/apps/admin/push/custom_events/`** | 自定义通知事件包 | 自定义的事件定义和触发函数单独放到此包下，**一个 Go 文件代表一个事件**。例如：<br>[admin_login.go](file:///Users/ryan/DEV/Go/Wavelet/internal/apps/admin/push/custom_events/admin_login.go) 代表管理员登录事件。 |
| **数据库审计表** | 状态与历史审计 | `w_push_events` 存放每个通知事件的启用状态、启用渠道、发送目标和自定义渲染模板。<br>`w_push_histories` 存放消息发送记录用于审计。 |

---

## 核心开发步骤 (Step-by-Step Flow)

如果某个新业务（如“新用户注册”或“订单创建”）需要带有消息推送功能，请严格按照以下步骤开发：

### 步骤 1：在 `custom_events/` 中以一个文件声明事件元数据
在 `internal/apps/admin/push/custom_events/` 下新建一个 Go 文件（如 `user_registered.go`），声明其事件元数据并利用 `init()` 动态注册。

```go
package custom_events

import (
	"context"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/admin/push"
	"github.com/Rain-kl/Wavelet/internal/model"
)

// NewUserRegistered is the metadata definition for the user registered event.
var NewUserRegistered = push.EventMetadata{
	Key:  "user_registered",
	Name: "新用户注册提醒",
	DefaultTemplate: push.NotificationMessage{
		Title:   "新用户注册通知",
		Content: "新用户 {{user.username}} (邮箱: {{user.email}}) 于 {{time}} 成功注册。",
		Level:   "INFO",
	},
	Description: "当系统有新用户注册成功时，向管理员或指定目标发送通知",
}

func init() {
	push.RegisterBuiltInEvent(NewUserRegistered)
}
```

### 步骤 2：在该事件文件中编写触发封装函数 (Wrapper)
为了使业务层调用方便且类型安全，在同一个事件 Go 文件中为该事件定义一个封装函数。

> [!NOTE]
> - 底层框架 `EventTrigger.Trigger` 本身已经内置了**异步 Goroutine 执行**以及 **`context.WithoutCancel(ctx)` 衍生上下文转换**逻辑。
> - 开发者只需在封装函数中组装数据，并直接调用 `DefaultTrigger.Trigger` 即可，无需在外部手动写 `go func()` 也不需要处理上下文防取消问题，从而通过框架底层强制约束了异步投递行为。

```go
// TriggerNewUserRegisteredEvent triggers the user registration notification event.
func TriggerNewUserRegisteredEvent(ctx context.Context, user *model.User) {
	if user == nil {
		return
	}
	body := map[string]any{
		"user": user,
		"time": time.Now().Format("2006-01-02 15:04:05"),
	}
	push.DefaultTrigger.Trigger(ctx, NewUserRegistered, body)
}
```

### 步骤 3：在业务代码中调用触发函数
在业务逻辑完成处（例如 `internal/apps/user/routers.go` 的注册 Handler 中）导入 `custom_events` 并调用该函数。

```go
import (
	"github.com/Rain-kl/Wavelet/internal/apps/admin/push/custom_events"
)

func Register(c *gin.Context) {
	// ... 注册成功逻辑 ...

	// 异步触发通知推送事件
	custom_events.TriggerNewUserRegisteredEvent(ctx, user)
}
```

### 步骤 4：在主路由器或初始化模块进行匿名导入以确保注册
由于事件是在 `custom_events` 的 `init()` 中注册到 `push` 包的，所以应用程序的执行路径（例如 [router.go](file:///Users/ryan/DEV/Go/Wavelet/internal/router/router.go)）必须匿名导入 `custom_events` 包，以确保其在程序启动时被加载和初始化。

```go
import (
	_ "github.com/Rain-kl/Wavelet/internal/apps/admin/push/custom_events"
)
```

当程序启动后，系统在初始化阶段的 `SyncEvents` 流程中会自动将新声明的 `user_registered` 事件元数据插入数据库表 `w_push_events` 中。此后，管理员即可直接在管理端前端界面上为该事件配置推送渠道。

---

## 模板模板渲染与支持的系统变量 (Template Rendering & Variables)

消息的 `title`、`content` 以及 `ext` 字段中的字符串值都支持变量占位符替换，采用双花括号形式 `{{variable}}`。

### 1. 通用事件参数 (Common Variables)
任何通知事件触发时，均支持以下通用参数的渲染：
- `{{time}}`：事件发生/触发的具体时间（格式：`2006-01-02 15:04:05`）

### 2. 特定事件携带的业务变量 (Event Specific Variables)
特定事件在触发时会携带复杂的业务对象（例如 `user`），可以通过点（`.`）路径语法获取其属性：

- **管理员登录提醒 (`admin_login`)**
  - `{{user.id}}`：管理员 ID
  - `{{user.username}}`：管理员用户名
  - `{{user.email}}`：管理员邮箱地址
  - `{{ip}}`：管理员登录来源的客户端 IP
  - `{{time}}`：管理员登录成功时间

- **新用户注册提醒 (`user_registered`)**
  - `{{user.id}}`：新注册用户 ID
  - `{{user.username}}`：新注册用户名
  - `{{user.email}}`：新注册用户邮箱地址
  - `{{time}}`：注册成功时间

---

## 严格遵循事项与防线 (Guardrails)

### 1. 严格禁止包循环引用 (No Circular Dependencies)
- `custom_events` 包在定义事件时会导入并依赖 `push` 包的 `EventMetadata` 与 `DefaultTrigger` 等底层逻辑。
- **因此，`push` 包本身绝对不能导入 `custom_events` 包**（否则编译会抛出 package dependency cycle 错误）。
- 对新事件的自动注册只能在 `custom_events` 内部使用 `init()` 调用 `push.RegisterBuiltInEvent` 来实现。

### 2. 单元测试防循环依赖隔离 (Unit Testing Isolation)
- 在对 `push` 包自身进行单元测试（如 `push_test.go`）时，由于 `custom_events` 依赖 `push`，测试文件 `push_test.go` 也无法直接导入 `custom_events`。
- **防线策略**：在 `push_test.go` 的 `init()` 中本地声明并使用 `RegisterBuiltInEvent` 注册测试专用的 `EventMetadata`，以此来完成 `push` 包的隔离自测。

### 3. 禁止绕过统一触发器 (Always Use EventTrigger)
- 所有推送请求必须经过 `EventTrigger.Trigger`，以确保进行“事件是否启用”、“目标渠道过滤”、“全局推送配置读取”及“发送日志审计”等流程。

### 4. 代码质量与零 Lint 警报
- **魔法值防范**：对于推送日志级别（如 `"INFO"`），不要在多个文件里写硬编码字符串，应统一在 `constants.go` 中引用 `defaultLevelInfo` 常量。
- **命名规范**：不要定义容易造成 Stuttering 的导出类型，例如在 `push` 包内不要使用 `PushSendPayload`，应重命名为 `SendPayload`。

---

## 验证计划 (Verification Guide)

1. **授权许可头部补全**：
   新增/修改 Go 文件后，运行自动许可证生成：
   ```bash
   make license
   ```
2. **Swagger 文档重新构建**：
   如果修改了路由或 Swagger 注释：
   ```bash
   make swagger
   ```
3. **静态代码质量门禁 (0 Issues)**：
   运行静态检查，必须保证后端与前端均无任何警告：
   ```bash
   make code-check
   ```
4. **单元测试通过**：
   ```bash
   go test ./internal/apps/admin/push/...
   ```
