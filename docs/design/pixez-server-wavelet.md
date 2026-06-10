# PixEzServer/Wavelet 伴生后端设计文档

本文档描述 PixEz 伴生后端（PixEzServer）的系统设计。后端基于 Wavelet 框架构建，使用其提供的认证、路由、数据库迁移、异步任务和文件存储等能力，承载账号同步、数据备份、镜像缓存、收藏导出追踪等业务。旧版独立的 Gin/SQLite 后端已废弃，仅作为历史参考，见 [pixez-sync-backend.md](pixez-sync-backend.md)。

## 1. 产品范围

PixEzServer 是专门为 PixEz Flutter 客户端设计的伴生后端，主要提供：

1. 在多设备间同步 Pixiv 登录凭证（Tokens）与 7 张本地核心业务表数据。
2. 支持"通过后端保存的凭证一键恢复 Pixiv 登录"的能力。
3. 提供"插画镜像加速缓存（Illustration Mirroring & Proxy）"功能，通过在本地或对象存储中缓存插画详情与图片，动态重写域名，实现客户端无需代理的加速体验。
4. 提供"小说镜像缓存"功能，缓存小说详情与正文 JSON，客户端在打开小说详情时可自动触发入队。
5. 提供"收藏导出与失效追踪"功能，由后台异步任务定期拉取 Pixiv 用户收藏，增量写入数据库并标记从 Pixiv 收藏夹中消失的插画与小说。

后端提供 REST API 与 Wavelet 内置 Admin 管理界面，不另行提供独立前端。

## 2. 系统目标与边界

- **兼容既有客户端接口**：保留 Flutter custom 层调用的 PixEz 业务路径 `/api/pixez/**` 与 `/mirror/**`。
- **统一响应格式**：`/api/pixez/**` 业务接口统一使用 Wavelet 规范的响应包裹：
  ```json
  { "error_msg": "", "data": {} }
  ```
- **镜像响应透传**：`/mirror/**` 保持 Pixiv 官方响应形态或输出原始二进制文件流，不使用系统 envelope。
- **统一认证机制**：鉴权全部接入 Wavelet 登录体系，客户端携带 `Authorization: Bearer <access_token>` 或 `X-Access-Token`。
- **现代化任务可观测性**：废弃旧版 SQLite 轮询队列，任务调度、重试、执行历史和日志由 Wavelet Asynq 和 `task_executions` 表统一管理。
- **统一文件存储**：镜像图片统一登记在 Wavelet `uploads` 表中，支持本地文件系统或 S3 兼容对象存储。

系统按单租户伴生服务设计，不引入多租户隔离与 `owner_user_id`。

## 3. 架构设计

### 模块拓扑

```text
PixEz Flutter custom layer
  |
  |  Authorization: Bearer <access_token>
  v
PixezServer/Wavelet Gin router
  |-- /api/pixez/**        -> internal/apps/pixez
  |-- /mirror/**           -> internal/apps/pixez mirror read handlers
  |-- /api/v1/admin/tasks  -> Wavelet Admin task APIs
  |
  v
internal/service/pixez
  |-- Pixiv App API client and token refresh
  |-- bookmark export semantics
  |-- mirror processing and Upload mapping
  |-- legacy server import
  |
  v
GORM models + goose migrations
  |-- PixEz sync/read-model tables
  |-- uploads
  |-- task_executions
  |
  v
Asynq Redis worker + Upload local/S3 storage
```

路由只在 `internal/router/router.go` 注册，业务 handler 位于 `internal/apps/pixez/`，跨 handler 的 Pixiv 请求与状态写入逻辑位于 `internal/service/pixez/`。

### 技术栈

- **语言**：Go 1.25+
- **Web 框架**：Gin
- **ORM**：GORM
- **数据库**：PostgreSQL（兼容 SQLite，goose 双方言迁移）
- **迁移工具**：goose
- **消息队列**：Asynq（Redis 驱动）
- **文件存储**：本地文件系统 / S3 兼容对象存储
- **部署**：Docker & docker-compose

### Pixiv 请求封装

所有请求 Pixiv 官方接口的代码统一封装在 `internal/service/pixez/client.go` 中，统一构造 Pixiv App 请求头、处理 Authorization、刷新 token、解析响应模型，使用以下 User-Agent：

```text
PixivAndroidApp/5.0.166 (Android 16; PKX110)
```

## 4. 数据模型

### Pixiv 账号表 `pixiv_users`

| 字段名 | 类型 | 约束 | 说明 |
| :--- | :--- | :--- | :--- |
| `pixiv_user_id` | `TEXT` | PRIMARY KEY | Pixiv 用户画师 ID |
| `name` | `TEXT` | NOT NULL | 用户昵称 |
| `account` | `TEXT` | NOT NULL | 账号/画师账号名 |
| `mail_address` | `TEXT` | | 注册邮箱 |
| `user_image` | `TEXT` | | 头像 URL |
| `access_token` | `TEXT` | NOT NULL | Pixiv 访问令牌 |
| `refresh_token` | `TEXT` | NOT NULL | Pixiv 刷新令牌 |
| `device_token` | `TEXT` | | 设备令牌 |
| `is_premium` | `INTEGER` | DEFAULT 0 | 是否 Pixiv 会员 |
| `x_restrict` | `INTEGER` | DEFAULT 0 | 限制级别 |
| `is_mail_authorized` | `INTEGER` | DEFAULT 0 | 邮箱是否已验证 |
| `created_at` | `DATETIME` | | 创建时间 |
| `updated_at` | `DATETIME` | | 更新时间 |

### 7 张备份同步表

每行数据通过 `pixiv_user_id` 与对应账号关联，`id` 均为自增主键，`pixiv_user_id` 上有索引。

1. **`ban_comments`**（对应本地 `banComment`）：`comment_id`（TEXT）、`name`（TEXT）
2. **`ban_illusts`**（对应本地 `banillustid`）：`illust_id`（TEXT）、`name`（TEXT）
3. **`ban_tags`**（对应本地 `bantag`）：`name`（TEXT）、`translate_name`（TEXT）
4. **`ban_users`**（对应本地 `banuserid`）：`user_id`（TEXT）、`name`（TEXT）
5. **`illust_histories`**（对应本地 `illustpersist`）：`illust_id`（INTEGER）、`user_id`（INTEGER）、`picture_url`（TEXT）、`title`（TEXT）、`user_name`（TEXT）、`time`（INTEGER）
6. **`novel_histories`**（对应本地 `Novelpersist`）：`novel_id`（INTEGER）、`user_id`（INTEGER）、`picture_url`（TEXT）、`title`（TEXT）、`user_name`（TEXT）、`time`（INTEGER）
7. **`tag_histories`**（对应本地 `tag`）：`name`（TEXT）、`translated_name`（TEXT）、`type`（INTEGER）

### 镜像 Read-Model 表

**`mirror_illust`** — 插画镜像状态与详情缓存：

| 字段名 | 类型 | 约束 | 说明 |
| :--- | :--- | :--- | :--- |
| `illust_id` | `INTEGER` | PRIMARY KEY | Pixiv 插画 ID |
| `task_id` | `TEXT` | | 最近一次关联的 Asynq 任务 ID |
| `status` | `TEXT` | | `queued` / `processing` / `success` / `failed` |
| `detail_json` | `TEXT` | | Pixiv `v1/illust/detail` 原始响应 JSON |
| `image_files_json` | `TEXT` | | 已下载图片的 Upload 映射 JSON 数组，每项包含 `pixiv_url`、`page`、`upload_id`、`file_name`、`hash`、`mime`、`size`、`storage_key` |
| `request_urls_json` | `TEXT` | | 从 Pixiv 详情提取的 original 档图片 URL 数组 |
| `retry_urls_json` | `TEXT` | | 下载失败的图片 URL 数组 |
| `total_count` | `INTEGER` | DEFAULT 0 | 应下载的图片总数 |
| `success_count` | `INTEGER` | DEFAULT 0 | 已成功下载数，`> 0` 即视为镜像可用 |
| `failed_count` | `INTEGER` | DEFAULT 0 | 下载失败数 |
| `created_at` | `DATETIME` | | 创建时间 |
| `updated_at` | `DATETIME` | | 更新时间 |

**`mirror_novel`** — 小说镜像状态与正文缓存：

| 字段名 | 类型 | 约束 | 说明 |
| :--- | :--- | :--- | :--- |
| `novel_id` | `INTEGER` | PRIMARY KEY | Pixiv 小说 ID |
| `task_id` | `TEXT` | | 最近一次关联的 Asynq 任务 ID |
| `status` | `TEXT` | | `queued` / `processing` / `success` / `failed` |
| `detail_json` | `TEXT` | | Pixiv `v2/novel/detail` 原始响应 JSON |
| `text_json` | `TEXT` | | Pixiv `webview/v2/novel` 原始正文 JSON |
| `success_count` | `INTEGER` | DEFAULT 0 | `1` 表示正文已成功缓存 |
| `created_at` | `DATETIME` | | 创建时间 |
| `updated_at` | `DATETIME` | | 更新时间 |

### 收藏导出表

**`bookmark_export_runs`** — 单次导出运行记录：

| 字段名 | 类型 | 约束 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | `TEXT` | PRIMARY KEY | 运行 ID |
| `pixiv_user_id` | `TEXT` | NOT NULL, INDEX | 被导出的 Pixiv 用户 ID |
| `restrict` | `TEXT` | NOT NULL, INDEX | `public` / `private` |
| `status` | `TEXT` | NOT NULL, INDEX | `running` / `success` / `failed` |
| `total_count` | `INTEGER` | DEFAULT 0 | 本轮成功解析的作品数 |
| `new_count` | `INTEGER` | DEFAULT 0 | 本轮新插入数 |
| `updated_count` | `INTEGER` | DEFAULT 0 | 本轮更新数 |
| `removed_count` | `INTEGER` | DEFAULT 0 | 本轮标记移除数 |
| `error_message` | `TEXT` | | 失败原因 |
| `started_at` | `DATETIME` | NOT NULL | 开始时间 |
| `finished_at` | `DATETIME` | | 完成时间 |
| `next_url` | `TEXT` | | 最近一次响应的下一页 URL |
| `last_request_url` | `TEXT` | | 最近一次请求 URL |
| `created_at` | `DATETIME` | | 创建时间 |
| `updated_at` | `DATETIME` | | 更新时间 |

**`bookmark_illusts`** — 插画收藏导出 Read-Model：

| 字段名 | 类型 | 约束 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | `INTEGER` | PRIMARY KEY AUTOINCREMENT | 本地行 ID |
| `pixiv_user_id` | `TEXT` | NOT NULL | 收藏所属 Pixiv 用户 ID |
| `restrict` | `TEXT` | NOT NULL | 收藏范围 |
| `illust_id` | `INTEGER` | NOT NULL | Pixiv 插画 ID |
| `title` | `TEXT` | | 标题冗余字段 |
| `type` | `TEXT` | | `illust` / `manga` / `ugoira` |
| `user_id` | `INTEGER` | | 作者 Pixiv 用户 ID |
| `user_name` | `TEXT` | | 作者名称 |
| `page_count` | `INTEGER` | | 页数 |
| `width` / `height` | `INTEGER` | | 尺寸 |
| `sanity_level` | `INTEGER` | | 内容级别 |
| `x_restrict` | `INTEGER` | | R-18 等限制标识 |
| `total_view` | `INTEGER` | | 浏览数 |
| `total_bookmarks` | `INTEGER` | | 收藏数 |
| `visible` | `INTEGER` | DEFAULT 0 | Pixiv 响应中的可见性 |
| `is_muted` | `INTEGER` | DEFAULT 0 | 是否被屏蔽 |
| `illust_ai_type` | `INTEGER` | | AI 类型 |
| `illust_json` | `TEXT` | NOT NULL | Pixiv 返回的完整插画 JSON |
| `last_export_run_id` | `TEXT` | NOT NULL, INDEX | 最近一次看到该插画的导出运行 ID |
| `last_seen_at` | `DATETIME` | NOT NULL | 最近一次在收藏页出现的时间 |
| `removed` | `INTEGER` | DEFAULT 0, INDEX | 是否已从收藏导出结果中消失 |
| `removed_at` | `DATETIME` | | 首次标记移除时间 |
| `created_at` | `DATETIME` | | 创建时间 |
| `updated_at` | `DATETIME` | | 更新时间 |

唯一约束：`UNIQUE(pixiv_user_id, restrict, illust_id)`。

**`bookmark_novels`** — 小说收藏导出 Read-Model，与 `bookmark_illusts` 结构类似，以 `novel_id` 作为业务键，包含 `novel_json`（TEXT）字段，唯一约束为 `UNIQUE(pixiv_user_id, restrict, novel_id)`。

### 数据库迁移

迁移文件位于 `internal/db/migrator/goose/`（包含 `postgres/` 和 `sqlite/` 双版本），关系通过显式索引和唯一约束表达，不使用物理外键：

- `202606090002_pixez_sync_core.sql`：账号与 7 张备份同步表
- `202606100001_pixez_mirror_bookmark.sql`：镜像与收藏导出表
- `202606100002_add_pixez_mirror_concurrency_config.sql`：并发与下载间隔系统设置
- `202606100003_create_schedules.sql`：定时任务调度表
- `202606100004_seed_pixez_auto_mirror_schedule.sql`：自动入队镜像定时任务种子数据

## 5. 异步任务

### 任务类型

PixEz 后台任务通过 Wavelet Admin 任务体系下发，Worker 统一进入 `task.ProcessTask`，执行记录写入 `task_executions`。

| Asynq Task（Worker 接收） | Admin Task Type（管理员下发） | 说明 |
| :--- | :--- | :--- |
| `pixez:mirror` | `pixez_mirror` | 抓取 Pixiv 插画或小说详情与正文并保存镜像记录（`target_type` 参数：`0` 插画，`1` 小说） |
| `pixez:export_bookmarks` | `pixez_export_bookmarks` | 增量导出同步账号的收藏并维护 removed 状态（`target_type` 参数：`0` 插画，`1` 小说，留空全部） |
| `pixez:auto_enqueue_bookmark_mirrors` | `pixez_auto_enqueue_bookmark_mirrors` | 扫描收藏 Read-Model，批量把未镜像或镜像失败的条目下发到镜像队列 |
| `pixez:import_legacy_server` | `pixez_import_legacy_server` | 从旧 `server/pixez-sync.db` 一次性导入历史业务数据与本地镜像文件 |

### 定时调度

`pixez_auto_enqueue_bookmark_mirrors` 通过 `202606100004_seed_pixez_auto_mirror_schedule.sql` 自动写入 `schedules` 表并激活，默认 cron 为 `*/10 * * * *`（每 10 分钟执行一次）。

### 状态分离

- **PixEz 业务状态**：保存在 `mirror_illust` / `mirror_novel`，值为 `queued`、`processing`、`success`、`failed`。
- **Wavelet 执行状态**：保存在 `task_executions`，值为 `pending`、`running`、`succeeded`、`failed`。

### 配置与并发控制

镜像并发与下载节奏通过 `system_configs` 动态配置：

| Key | 默认值 | 说明 |
| :--- | :--- | :--- |
| `pixez_mirror_download_interval_seconds` | `1` | 插画多图下载间隔（秒），规避 Pixiv 速率限制 |
| `pixez_mirror_illust_concurrency` | `5` | 同时执行插画镜像的最大任务并发数 |
| `pixez_mirror_novel_concurrency` | `5` | 同时执行小说镜像的最大任务并发数 |

## 6. 业务流程

### 插画镜像流程

1. `POST /api/pixez/illusts/:illust_id/mirror`：幂等下发 Asynq 任务（若已有非 failed 记录则直接返回当前状态）。
2. Worker 读取最近同步的 Pixiv 用户凭证，请求 Pixiv `v1/illust/detail`。
3. 从详情中收集 original 档图片 URL。
4. 按 `pixez_mirror_download_interval_seconds` 间隔逐图下载，写入 Wavelet Upload 存储并创建 `uploads` 记录。
5. `mirror_illust.image_files_json` 保存 `{pixiv_url, page, upload_id, file_name, hash, mime, size, storage_key}` 数组。
6. `/mirror/v1/illust/detail` 返回原始 Pixiv 详情 JSON，并把 `i.pximg.net` / `s.pximg.net` 改写为 `/mirror/pximg`。
7. `/mirror/pximg/*path` 根据 original/master/square 文件名映射查找 Upload 记录并流式输出；本地未命中时直接代理 Pixiv 原始地址。

### 小说镜像流程

1. `POST /api/pixez/novels/:novel_id/mirror`：幂等下发 Asynq 任务。
2. Worker 请求 Pixiv `v2/novel/detail` 与 `webview/v2/novel`。
3. `mirror_novel.detail_json` 保存详情原始 JSON，`mirror_novel.text_json` 保存正文 JSON。
4. `/mirror/v1/novel/detail` 与 `/mirror/webview/v2/novel` 返回 Pixiv 形态 JSON，不套 envelope。

Flutter 客户端可在"数据同步设置"中开启"自动镜像小说"开关，打开小说详情页时自动调用入队接口；该开关只影响客户端行为，不改变后端 API 的幂等语义。

### 收藏导出与失效追踪

收藏导出按 Pixiv 用户和 `restrict=public|private` 分页执行：

1. 插画调用 Pixiv `GET /v1/user/bookmarks/illust`，小说调用 `GET /v1/user/bookmarks/novel`。
2. 按 `next_url` 逐页拉取直到结束，将作品 JSON 增量 upsert 到数据库（禁止先删后查）。
3. 已存在且未标记 removed 的 active 记录只更新本轮运行 ID 与最后出现时间，不重写完整 JSON。
4. 图片 URL 包含 `limit_unknown_360` 的插画、封面 URL 包含 `limit_unknown_100` 的小说立即标记 `removed=true`，并且不计入本轮有效出现。
5. 分页**全部成功完成**后，把本轮未出现但历史仍 active 的记录标记 `removed=true` 并写 `removed_at`。
6. 分页中途失败时不执行缺失标记，避免误判。

客户端查询已移除插画：

```
GET /api/pixez/users/:pixiv_user_id/bookmarks/illust/removed
```

返回 Wavelet envelope，`data` 内保持 Pixiv 列表形态：`illusts` 与 `next_url`。支持可选参数 `restrict=public|private`、`offset`、`limit`（默认 30，最大 100）。

### Legacy 导入

旧后端数据通过 Admin 任务 `pixez_import_legacy_server` 一次性导入，参数示例：

```json
{
  "sqlite_path": "server/pixez-sync.db",
  "mirror_dir": "server/data/mirror",
  "dry_run": false
}
```

规则：

- `pixiv_users` 和 7 张同步表导入到 PixezServer 数据库。
- `bookmark_illusts`、`bookmark_novels` 使用唯一键 upsert，支持幂等重复运行。
- 旧镜像文件存在时计算 hash/mime/size 并登记为 Wavelet Upload，文件缺失只记录 missing 计数不伪造成功。
- 旧 `mirror_tasks` 只读取历史状态、URL 和计数填充 read-model，不作为新队列。

## 7. API 接口

`/mirror/**` 是 Pixiv 官方接口的镜像命名空间，只承载与 Pixiv 官方请求语义一致的镜像接口；镜像任务管理等业务操作注册在 `/api/pixez/**` 下。所有接口均需 Wavelet AccessToken 鉴权。

Swagger/OpenAPI 文档由 `swaggo/swag` 根据 Go 注释生成，可通过 `GET /swagger/index.html` 访问。

### `/api/pixez/**` 接口列表

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/pixez/ping` | 健康检查 & 鉴权验证 |
| `GET` | `/api/pixez/users` | 获取所有已同步 Pixiv 用户列表（不返回 token） |
| `GET` | `/api/pixez/users/:pixiv_user_id` | 获取特定 Pixiv 用户完整凭证（含 token，用于一键恢复登录） |
| `PUT` | `/api/pixez/users/:pixiv_user_id` | 新增或更新 Pixiv 用户凭证（Upsert，path 中 ID 为权威值） |
| `DELETE` | `/api/pixez/users/:pixiv_user_id` | 删除 Pixiv 用户及关联的所有备份同步数据 |
| `GET` | `/api/pixez/users/:pixiv_user_id/sync-data` | 下载用户备份数据，支持 `?tables=` 选择性下载 |
| `POST` | `/api/pixez/users/:pixiv_user_id/sync-data` | 替换上报备份数据（事务内只替换请求中包含的表） |
| `GET` | `/api/pixez/users/:pixiv_user_id/sync-data/hashes` | 获取各同步表的 MD5 hash（空表返回 `"empty"`） |
| `GET` | `/api/pixez/users/:pixiv_user_id/bookmarks/illust/removed` | 查询已标记失效的收藏插画 |
| `POST` | `/api/pixez/illusts/:illust_id/mirror` | 幂等下发插画镜像任务 |
| `GET` | `/api/pixez/illusts/:illust_id/mirror` | 查询插画镜像状态 |
| `POST` | `/api/pixez/illusts/mirror/batch` | 批量查询已镜像的插画 ID |
| `POST` | `/api/pixez/novels/:novel_id/mirror` | 幂等下发小说镜像任务 |
| `GET` | `/api/pixez/novels/:novel_id/mirror` | 查询小说镜像状态 |
| `POST` | `/api/pixez/novels/mirror/batch` | 批量查询已镜像的小说 ID |
| `GET` | `/api/pixez/mirror/illusts` | 分页查询插画镜像管理列表 |
| `GET` | `/api/pixez/mirror/novels` | 分页查询小说镜像管理列表 |
| `DELETE` | `/api/pixez/mirror/illusts/:illust_id` | 删除单条插画镜像记录并标记关联 Upload 删除 |
| `DELETE` | `/api/pixez/mirror/novels/:novel_id` | 删除单条小说镜像记录 |
| `POST` | `/api/pixez/mirror/batch-delete` | 批量删除指定类型的镜像记录 |

### `/mirror/**` 接口列表

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/mirror/v1/illust/detail` | 读取已缓存的插画详情 JSON，改写 pximg URL 后返回 Pixiv 形态响应 |
| `GET` | `/mirror/pximg/*path` | 从 Upload 存储输出镜像图片流；master/square 路径自动映射到 original 文件；本地未命中时代理 Pixiv 原始地址 |
| `GET` | `/mirror/v1/novel/detail` | 读取已缓存的小说详情 JSON，返回 Pixiv 形态响应 |
| `GET` | `/mirror/webview/v2/novel` | 读取已缓存的小说正文 JSON，返回 Pixiv 形态响应 |

### 接口响应示例

**`GET /api/pixez/ping`** — 健康检查：
```json
{ "error_msg": "", "data": { "status": "ok" } }
```

**`GET /api/pixez/users`** — 用户列表（不含 token）：
```json
{
  "error_msg": "",
  "data": [
    {
      "pixiv_user_id": "12345678",
      "name": "PixivUser",
      "account": "user_account",
      "mail_address": "user@example.com",
      "user_image": "https://pximg.net/...",
      "is_premium": 0,
      "x_restrict": 0,
      "is_mail_authorized": 1,
      "created_at": "2026-06-06T16:00:00Z",
      "updated_at": "2026-06-06T16:15:00Z"
    }
  ]
}
```

**`GET /api/pixez/users/:pixiv_user_id`** — 含 token 的完整凭证：
```json
{
  "error_msg": "",
  "data": {
    "pixiv_user_id": "12345678",
    "name": "PixivUser",
    "account": "user_account",
    "mail_address": "user@example.com",
    "user_image": "https://pximg.net/...",
    "access_token": "pixiv_access_token_value",
    "refresh_token": "pixiv_refresh_token_value",
    "device_token": "pixiv_device_token_value",
    "is_premium": 0,
    "x_restrict": 0,
    "is_mail_authorized": 1,
    "created_at": "2026-06-06T16:00:00Z",
    "updated_at": "2026-06-06T16:15:00Z"
  }
}
```

**`PUT /api/pixez/users/:pixiv_user_id`** — 上报/更新凭证，请求体：
```json
{
  "name": "PixivUser",
  "account": "user_account",
  "mail_address": "user@example.com",
  "user_image": "https://pximg.net/...",
  "access_token": "pixiv_access_token_value",
  "refresh_token": "pixiv_refresh_token_value",
  "device_token": "pixiv_device_token_value",
  "is_premium": 0,
  "x_restrict": 0,
  "is_mail_authorized": 1
}
```
响应：`{ "error_msg": "", "data": null }`

**`POST /api/pixez/illusts/:illust_id/mirror`** — 下发镜像任务后立即返回当前状态：
```json
{
  "error_msg": "",
  "data": {
    "task_id": "01J...",
    "illust_id": 123,
    "status": "queued",
    "mirrored": false,
    "total_count": 0,
    "success_count": 0,
    "failed_count": 0
  }
}
```

**`GET /api/pixez/illusts/:illust_id/mirror`** — 镜像完成时的状态示例：
```json
{
  "error_msg": "",
  "data": {
    "task_id": "01J...",
    "illust_id": 123,
    "status": "success",
    "mirrored": true,
    "total_count": 40,
    "success_count": 28,
    "failed_count": 12,
    "request_urls_json": "[\"https://i.pximg.net/...\"]",
    "retry_urls_json": "[\"https://i.pximg.net/...\"]"
  }
}
```

**`GET /api/pixez/users/:pixiv_user_id/bookmarks/illust/removed`** — 已失效收藏：
```json
{
  "error_msg": "",
  "data": {
    "illusts": [],
    "next_url": "/api/pixez/users/12345678/bookmarks/illust/removed?offset=30&limit=30"
  }
}
```

## 8. Flutter 客户端接入

Flutter custom 层扩展集中在 `lib/custom/`：

- `lib/custom/services/sync_config.dart`：保存同步服务器地址、Wavelet AccessToken、同步周期与"自动镜像小说"开关；不再包含 username/password 配置。
- `lib/custom/services/sync_api.dart`：封装所有 `/api/pixez/**` 与 `/mirror/**` 请求；统一通过 `Authorization: Bearer <token>` 携带凭证；统一解包 `{ error_msg, data }` 响应（`error_msg == ""` 为成功），解包逻辑集中在服务层，不扩散到页面。
- `lib/custom/services/novel_auto_mirror_service.dart`：小说详情页打开后的自动入队判断，仅在开关开启且服务器地址存在时触发。
- `lib/custom/pages/sync_settings_page.dart`：数据同步配置界面，输入 AccessToken 而非用户名/密码。

AccessToken 获取路径：用户在 PixezServer Web 管理端登录后，通过现有 AccessToken 管理功能创建并复制 token，再填入 Flutter 同步设置页。
