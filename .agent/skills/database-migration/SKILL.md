---
name: "database-migration"
description: "Wavelet 项目专用：当新增或修改数据库表结构、索引、初始化数据、系统配置 seed、模板 seed、默认管理员、goose SQL 迁移、internal/db/migrator 或数据库升级流程时必须使用。本技能指导在 internal/db/migrator/goose 下编写 PostgreSQL/SQLite 双方言 SQL 迁移，并完成验证。"
---

# Wavelet 数据库升级操作指南

Wavelet 使用 `github.com/pressly/goose/v3` 执行 SQL 迁移。迁移入口是 `internal/db/migrator.Migrate()`，SQL 文件嵌入在二进制中。

## 基本规则

- SQL 迁移文件放在：
    - `internal/db/migrator/goose/postgres/`
    - `internal/db/migrator/goose/sqlite/`
- PostgreSQL 和 SQLite 必须使用同一个版本号、同一个语义文件名。
- 迁移文件使用 goose SQL 标记：

```sql
-- +goose Up
...

-- +goose Down
...
```

- 不要把表结构、默认系统配置、默认模板、默认管理员初始化写回 Go 代码。
- 不要添加物理外键；关系字段使用显式索引。
- 数据库默认值应匹配 Go model 零值或业务兜底值。
- 系统配置仍然保存字符串值；布尔值写 `"true"` / `"false"`，数字写十进制字符串，复杂结构写合法 JSON 字符串。

## 新增迁移流程

1. 先确认涉及的 Go model、读写路径和前端/接口消费方。
2. 选择下一个递增版本号，格式建议 `YYYYMMDDNNNN`，例如：

```text
202606090002_add_example_column.sql
```

3. 在 PostgreSQL 和 SQLite 目录各新增同名 SQL 文件。
4. 写 `Up`：
   - 表结构变更使用 SQL DDL。
   - 初始化/seed 数据使用 SQL `INSERT`。
   - 需要幂等时使用 `IF NOT EXISTS` 或 `ON CONFLICT ... DO NOTHING`。
5. 写 `Down`：
   - 能安全回滚的结构变更写反向 DDL。
   - seed 数据按 key/name 等稳定标识删除。
6. 如果变更 API handler，运行 `make swagger`。
7. 至少运行：

```bash
go test ./internal/db/migrator
go test ./internal/model ./internal/apps/config ./internal/apps/admin/system_config
make code-check
```

## 方言注意事项

- PostgreSQL 自增主键用 `BIGSERIAL`；SQLite 自增主键用 `INTEGER PRIMARY KEY AUTOINCREMENT`。
- PostgreSQL 时间类型优先 `TIMESTAMPTZ`；SQLite 使用 `DATETIME`。
- PostgreSQL JSON 字段用 `JSONB`；SQLite 用 `JSON` 或 `TEXT`。
- 两个方言目录的字段名、索引名、seed 数据语义必须保持一致。

## 修改默认系统配置

- 新增或调整系统配置 seed 时，更新两个方言的 SQL 文件。
- `visibility` 使用常量语义：`0` 不公开，`1` 通过 `/api/v1/config/public` 返回。
- 公共配置 API 直接返回所有 `visibility = 1` 的配置键值，不要在 handler 中重新硬编码 key 列表。

## 验证重点

- goose 能在空库上完整执行。
- `system_configs`、默认 `admin`、内置模板能按预期初始化。
- 新增表/列与 Go model 的列名、类型和默认值兼容。
- 前端或接口消费的公共配置值仍按字符串解析。
