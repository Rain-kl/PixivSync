---
name: "new-api"
description: "Wavelet 项目专用：当新增或修改自定义业务 API、新增业务路由、新增 service 层核心逻辑时必须使用。本技能指导包职责划分、推荐文件结构、路由解耦、Swagger 文档生成与质量门禁验证。"
---

# 新增业务 API / 接口开发规范

本技能涵盖 Wavelet 的业务接口开发规范。开始开发前先阅读仓库根目录 [AGENTS.md](file:///Users/ryan/DEV/Go/Wavelet/AGENTS.md)，遵守项目级核心规则。

为了保持核心路由入口的稳定性，**所有新增的定制业务接口路由统一注册在独立的 go 文件中，严禁直接堆叠到 `router.go`**。

---

## 包职责划分 (Package Responsibilities)

按照 Go 语言最佳实践与 Google 的开发风格，接口开发应该进行严格的分层，以避免循环依赖和逻辑混乱。

| 目录/包名 | 职责定位 | 框架依赖限制 | 常见包含内容 |
| :--- | :--- | :--- | :--- |
| **`internal/router/`** | 路由分发层 | 依赖 Gin 框架 | `router.go` 核心路由、`custom.go` (自定义路由注册入口) |
| **`internal/apps/custom/`** | 功能模块层 (Feature Module) | 依赖 Gin 框架 (仅限路由/Handler 部分) | 高度内聚的业务功能块。接收 HTTP 请求、解析请求体、校验基础参数、提取 Session。业务服务逻辑直接实现在当前目录下的 `service.go` 或 `logics.go` 中，不依赖 Gin/HTTP 框架。 |
| **`internal/model/`** | 数据模型层 | 依赖 GORM / SQL 基础 | GORM 实体定义、表结构、主键生成、单表极简 SQL 查询方法。 |
| **`internal/db/`** | 数据存储层 | 依赖 SQL 驱动 / GORM 连接 | PostgreSQL, SQLite 等数据库连接管理与 Goose 数据库迁移文件。 |

---

## 建议创建/修改的文件结构

当新增一套定制的业务接口（例如名为 `custom` 的业务模块）时，根据逻辑复杂度建议采用以下文件结构：

```text
internal/
├── router/
│   └── custom.go               # [修改/创建] 仅用于注册定制路由，将路由委托给 apps/custom
└── apps/
    └── custom/
        ├── routers.go          # [新建] HTTP Handlers (Gin)，负责参数绑定、校验与响应
        ├── logics.go           # [新建] 承载功能模块内闭环的业务逻辑（可以使用 logics.go 或 service.go）
        └── errs.go             # [新建] 仅存放业务特有的错误常量定义（可选）
```

---

## 核心开发步骤 (Step-by-Step Flow)

### 步骤 1：如果有数据库变更，编写数据库迁移
如果需要新表或字段，请参考 [database-migration](../database-migration/SKILL.md) 技能，在 `internal/db/migrator/goose/` 目录下编写迁移文件。在 `internal/model/` 中定义 GORM 数据模型。

### 步骤 2：在模块内实现业务服务与逻辑 (Service / Logics)
在编写具体逻辑前，建议选择以下结构实现业务逻辑（均置于 `internal/apps/custom/` 下）：
- **方案 A（轻量化函数形式，推荐）**：在 `logics.go` 中定义独立的纯 Go 函数，这些函数不强依赖 `*gin.Context`。
- **方案 B（面向对象/结构体形式）**：在 `service.go` 中定义 Service 结构体和构造函数，如 `type CustomService struct`，并将逻辑作为其方法。这适用于需要注入依赖（如 DB 连接、外部 client 等）或有状态管理的对象。
参考示例：[logics_example.go](file:///Users/ryan/DEV/Go/Wavelet/.agent/skills/new-api/references/logics_example.go) 和 [service_example.go](file:///Users/ryan/DEV/Go/Wavelet/.agent/skills/new-api/references/service_example.go)

### 步骤 3：在 `internal/apps/custom/` 下编写 HTTP Handler
创建应用路由文件 `routers.go`，定义接口的请求和响应 DTO，编写 Handler 绑定参数并调用 Service，编写 Swagger 注释。
参考示例：[handler_example.go](file:///Users/ryan/DEV/Go/Wavelet/.agent/skills/new-api/references/handler_example.go)

### 步骤 4：在 `internal/router/custom.go` 中注册路由
创建路由挂载函数：
```go
package router

import (
	"github.com/Rain-kl/Wavelet/internal/apps/custom"
	"github.com/gin-gonic/gin"
)

func registerCustomRoutes(apiV1Router *gin.RouterGroup) {
	customRouter := apiV1Router.Group("/custom")
	{
		customRouter.POST("/action", custom.DoActionHandler)
	}
}
```
并在 [router.go](file:///Users/ryan/DEV/Go/Wavelet/internal/router/router.go) 中的 `/v1` 路由组末尾调用此函数。

---

## 质量验证与门禁 (Verification Quality Gates)

每当新增或修改 API 接口时，必须严格执行以下验证：

1. **生成授权许可**:
   新增 Go 文件后，运行自动添加许可证头部命令：
   ```bash
   make license
   ```

2. **生成 Swagger 文档**:
   在 Handler 编写完 `@Summary` 等 Swagger 注释后，必须生成更新：
   ```bash
   make swagger
   ```
   *注意：若 Swagger 生成失败，请仔细排查注释格式或数据类型引用是否规范。*

3. **静态代码检查与 Linting**:
   运行 `golangci-lint` 与前端 TypeScript 门禁，确保没有代码风格和类型安全隐患：
   ```bash
   make code-check
   ```

4. **编译与功能测试**:
   运行整包编译与自动化测试：
   ```bash
   make build-test
   ```

---

## 相关 Skills
* [go-context](../go-context/SKILL.md)：了解如何在 Service 层正确传递取消信号和追踪 Trace。
* [go-error-handling](../go-error-handling/SKILL.md)：了解如何优雅地将业务错误向上传递，并在 Handler 层决定响应状态码。
* [database-migration](../database-migration/SKILL.md)：当新增接口需要额外表结构或默认配置种子时配合使用。
