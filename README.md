# Pixez Cloud

为 PixEz 打造的云端备份与同步服务

[![Lic ense: Apache2.0](https://img.shields.io/badge/License-Apache2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org/)
[![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)](https://nextjs.org/)
[![React](https://img.shields.io/badge/React-19-blue.svg)](https://reactjs.org/)

Pixez Cloud 是专为 PixEz 用户打造的云端数据伴侣。无论你拥有几台设备，它都能让你的 Pixiv 账号凭证、浏览记录、屏蔽名单等关键数据无缝同步，告别每台设备都要单独配置的烦恼。

![hero_page.png](docs/assets/hero_page.png)

## 它能为你做什么

### 🔐 一键恢复登录

换个设备登录 Pixiv？不用再翻找密码。Pixez Cloud 帮你保管登录凭证，新设备上点一下就能恢复账号状态。

### 💨 图片加速访问

网络不畅时，插画加载总是转圈圈？云端自动缓存你浏览过的作品，图片加载更快更稳定。

### 📦 收藏不怕丢

辛苦攒下的几千个收藏，万一丢失怎么办？云端定期备份你的收藏列表，还能帮你追踪哪些作品已经失效。

### 📱 多设备随心切换

手机、平板、多台手机——所有设备的浏览记录、屏蔽设置、搜索历史，统统保持一致。

![dashboard.png](docs/assets/dashboard.png)


## 快速开始

### Docker Compose 部署

```yaml
services:
  wavelet:
    image: ghcr.io/rain-kl/pixezserver:latest
    restart: unless-stopped
    env_file: .env
    environment:
      TZ: ${TZ:-Asia/Shanghai}
    ports:
      - "${APP_PORT:-8061}:8061"
    volumes:
      - ./uploads:/app/uploads
      - ./data/:/app/data
    depends_on:
      redis:
        condition: service_healthy

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: ["redis-server", "--appendonly", "yes"]
    volumes:
      - ./data/redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 5s
```

下载仓库的 .env.example 文件到本地：

```bash
cp .env.example .env
```

- 将 `APP_SESSION_SECRET` 改为足够长的随机字符串。
- 本地 HTTP 测试时设置 `APP_SESSION_SECURE=false`。
- 只有在 HTTPS 部署时保留 `APP_SESSION_SECURE=true`。

启动服务：

```bash
docker compose up -d
```

初始本地管理员账号：

```text
username: admin
password: 12345678
```

## 接入 PixEz Flutter

下载修改版的 PixEz Flutter 客户端，安装到设备上：

地址: https://github.com/Rain-kl/pixez-flutter/releases

1. 登录 Pixez Sync Web 管理端。
2. 打开 `/settings/access-token`，为客户端创建 AccessToken。
3. 在 PixEz Flutter 的自定义数据同步设置页填写服务器地址。
4. 地址示例：`https://pixez.example.com`。
5. 粘贴 AccessToken。

Flutter custom 层应把 Wavelet envelope 解包集中保留在 sync API service 内：`error_msg == ""` 表示接口成功。`/mirror/**` 是 Pixiv 形态响应，不参与 envelope 解包。



## 系统架构

```text
PixEz Flutter custom layer
  |
  |  Authorization: Bearer <access_token>
  v
PixEzServer / Wavelet
  |-- /api/pixez/**        Wavelet envelope 包裹的业务接口
  |-- /mirror/**           Pixiv 形态镜像读取和图片流
  |-- /api/v1/admin/tasks  任务下发、重试、调度和日志
  |
  v
internal/service/pixez
  |-- Pixiv App API client 与 token refresh
  |-- sync-data 备份与 hash 对比
  |-- 插画 / 小说镜像处理
  |-- 收藏导出与 removed 状态追踪
  |-- 旧 SQLite 与镜像文件导入
  |
  v
GORM models + goose migrations + uploads + task_executions
  |
  v
Redis / Asynq worker + 本地磁盘或 S3 兼容存储
```

## 核心能力

### Pixiv 账号同步

PixEz Flutter 可以上报 Pixiv 用户信息、access token、refresh token、device token、会员标记和限制标记。后端保存到 `pixiv_users`，对外提供不含 token 的用户列表，并在客户端需要一键恢复登录时返回完整凭证。

### 本地数据备份

同步 API 会按 Pixiv 用户保存 7 张 PixEz 本地表：

- `ban_comments`
- `ban_illusts`
- `ban_tags`
- `ban_users`
- `illust_histories`
- `novel_histories`
- `tag_histories`

Hash 接口用于让客户端跳过未变化的表，避免每次全量上传。

### 插画镜像

`POST /api/pixez/illusts/:illust_id/mirror` 会幂等下发 Asynq 任务。Worker 请求 Pixiv `v1/illust/detail`，下载 original 图片，经 Wavelet Upload 存储写入本地或 S3，并把映射保存到 `mirror_illust`。

`/mirror/v1/illust/detail` 返回 Pixiv 形态详情 JSON，并把 pximg 域名改写到 `/mirror/pximg`；`/mirror/pximg/*path` 优先流式输出已缓存文件，未命中时可回退代理 Pixiv 原始地址。

### 小说镜像

`POST /api/pixez/novels/:novel_id/mirror` 会下发小说镜像任务。Worker 保存 Pixiv `v2/novel/detail` 与 `webview/v2/novel` 原始 JSON 到 `mirror_novel`。

Flutter custom 层可以在小说详情页打开时自动入队镜像，该行为由客户端“自动镜像小说”同步设置控制。

### 收藏导出与失效追踪

后台任务按 public/private 分页导出插画和小说收藏。导出规则是增量式的：

- 新作品插入数据库。
- 已存在且仍 active 的记录，只更新本轮运行 ID 和最近出现时间。
- 包含 `limit_unknown_360`、`limit_unknown_100` 等占位图的作品立即标记 removed。
- 只有整轮分页成功结束后，才把本轮未出现的历史 active 记录标记为 removed。
- 分页中途失败时不做缺失标记，避免误判。

PixEz 管理界面基于这些 read-model 展示镜像进度、失败项、不可见作品和最近导出批次。

### 任务与运维控制台

PixEz 任务接入 Wavelet Admin 任务体系：

| Admin task type | Asynq task | 用途 |
| --- | --- | --- |
| `pixez_mirror` | `pixez:mirror` | 镜像单个插画或小说 |
| `pixez_export_bookmarks` | `pixez:export_bookmarks` | 导出收藏 read-model 并维护 removed 状态 |
| `pixez_auto_enqueue_bookmark_mirrors` | `pixez:auto_enqueue_bookmark_mirrors` | 扫描未镜像或失败的收藏条目并批量入队 |
| `pixez_import_legacy_server` | `pixez:import_legacy_server` | 导入旧 PixEz sync SQLite 数据与镜像文件 |

`task_executions` 记录执行状态、日志、重试信息和失败原因。默认收藏自动入队镜像调度为 `*/10 * * * *`。

## 本地源码开发

环境要求：

- Go 1.25+
- Node.js 18+
- pnpm 8+
- Redis 6+
- 如果启用 PostgreSQL，需要 PostgreSQL 14+

启动本地 Redis：

```bash
docker compose up -d redis
```

准备配置：

```bash
cp config.example.yaml config.yaml
```

SQLite 开发可将 `database.enabled` 设为 `false` 并设置 `database.sqlite_path`。PostgreSQL 开发则保留 `database.enabled=true` 并更新数据库连接信息。

运行后端进程：

```bash
go mod tidy
go run main.go api
go run main.go scheduler
go run main.go worker
```

运行前端：

```bash
cd frontend
pnpm install
pnpm dev
```

常用质量门禁：

```bash
make swagger       # API handler 变化后执行
make code-check    # 提交前必跑
make build-test    # 前后端构建验证
make build-embedded
```

## License

This project is licensed under the [Apache 2.0 License](LICENSE).
