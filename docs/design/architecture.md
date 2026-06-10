# Architecture

本文档描述 PixEz Flutter 当前架构、模块职责和主要数据流。仓库目录结构见 [index.md](index.md)。

## 总体拓扑

```text
User
  |
  v
Flutter UI
  |-- Material branch: lib/page/ + lib/component/
  |-- Fluent branch:  lib/fluent/
  |
  v
State and orchestration
  |-- Global stores: lib/store/
  |-- Page stores/notifiers: lib/page/**/*
  |-- Riverpod scope: ProviderScope in lib/main.dart
  |
  v
Domain models and services
  |-- Models: lib/models/
  |-- Runtime helpers: lib/er/
  |-- Local server: lib/server/
  |
  v
Network and platform adapters
  |-- Pixiv API / OAuth: lib/network/
  |-- Platform channels/plugins: lib/*_plugin.dart
  |-- Local plugin: plugins/rhttp/
  |
  v
External systems
  |-- pixiv API and OAuth
  |-- Image hosts / WebView targets / third-party lookup services
  |-- Device file system, permissions, OS APIs
```

## 启动流程

1. `lib/main.dart` 调用 `Rhttp.init()` 和 `MmapCache.init()`。
2. Flutter binding 初始化。
3. Windows/Linux 初始化 sqlite FFI，并设置数据库目录。
4. Windows/Linux 初始化单实例能力。
5. `initFluent(args)` 处理 Fluent UI/桌面相关启动参数。
6. `runApp(ProviderScope(child: MyApp(...)))` 挂载应用。
7. `MyApp.initState()` 初始化用户设置、账号、收藏标签、静音规则等全局状态。
8. `MyApp.build()` 根据 `Constants.isFluent` 选择 Fluent UI 或 Material UI。
9. Material 分支进入 `SplashPage`，再由启动页分发到登录、初始化或主页面。

## UI 层

### Material 分支

Material 分支主要位于：

- `lib/page/`
- `lib/component/`

它覆盖移动端和默认 Flutter Material 风格页面。页面目录通常按业务对象拆分，例如 `picture/`、`novel/`、`search/`、`user/`、`login/`。页面级状态文件常与页面放在同一目录下，如 `*_store.dart`、`*_notifier.dart`。

### Fluent 分支

Fluent 分支主要位于：

- `lib/fluent/`
- `lib/fluent/page/`
- `lib/fluent/component/`

它提供桌面风格 UI 和导航框架。多数业务页面与 Material 分支保持语义对应，但布局、导航和控件风格独立实现。

### 组件层

`lib/component/` 负责复用 UI 片段和展示控件，例如：

- 图片展示与屏蔽态处理。
- 插画、画师、Spotlight 卡片。
- 空状态、失败态、展开动画。
- ugoira 播放、HTML 文本、评论表情。

二开新增组件不应直接散落在此目录，除非是在原公共组件上增加极小且向后兼容的可选能力。

## 状态管理

当前项目同时存在多种状态管理方式：

- 全局对象和 MobX store：`lib/main.dart` 中创建并导出全局 store。
- `flutter_mobx`：UI 通过 `Observer` 响应全局或页面 store。
- Riverpod/Hooks Riverpod：应用由 `ProviderScope` 包裹，部分页面使用 notifier/provider。
- 页面局部 store/notifier：随页面目录维护。

全局状态包括：

- `userSetting`
- `saveStore`
- `muteStore`
- `accountStore`
- `tagHistoryStore`
- `novelHistoryStore`
- `topStore`
- `bookTagStore`
- `splashStore`
- `fullScreenStore`

设计要求：

- 新增功能优先在独立目录内维护自己的 provider/controller/store。
- 必须接入全局状态时，只在入口处追加最小注册或读取逻辑。
- 不重命名、不移动、不重构现有全局 store。

## 网络层

网络层位于 `lib/network/`，核心职责如下：

- `api_client.dart`：pixiv App API 请求封装。
- `oauth_client.dart`：登录、授权码和 token 刷新请求封装。
- `refresh_token_interceptor.dart`：访问 token 失效后的刷新逻辑。
- `network_mode.dart` / `pixez_network_settings.dart`：网络模式和 rhttp 客户端配置。
- `onezero_client.dart`、`account_client.dart`：特定服务或业务对象请求入口。

网络请求主要经由 Dio 发起，并使用 `dio_compatibility_layer` 将 `plugins/rhttp` 提供的兼容客户端接入 Dio adapter。图片和 API 请求可根据 `userSetting.networkMode` 选择不同网络配置。

设计要求：

- 不直接把二开接口塞进现有 `ApiClient`。
- 自定义接口应放在 `lib/custom/services/`，通过适配器或组合方式复用现有网络封装。
- 涉及 token、请求头、缓存策略的改动要谨慎评估登录态、刷新 token 和图片加载影响。

## 模型与持久化

`lib/models/` 保存 API 响应模型、本地持久化对象和业务实体。项目使用 `json_annotation`、`json_serializable`、`freezed` 等生成代码能力。

本地持久化和设置散布在以下模块：

- `shared_preferences`：用户设置、轻量状态。
- `sqflite` / `sqflite_common_ffi`：移动端和桌面端数据库能力。
- `path_provider` 和平台路径插件：保存路径、缓存路径和数据库路径。
- 页面/store 自身的持久化对象：历史、收藏标签、屏蔽、任务等。

设计要求：

- 新增模型优先放在自定义目录。
- 适配原模型时优先写 adapter，避免修改原模型字段。
- 修改生成模型后必须运行代码生成并检查生成文件。

## 平台能力

根目录下的 `lib/*_plugin.dart` 文件和平台工程共同提供平台能力，包括但不限于：

- 剪贴板、深链、文档、系统设置。
- Android SAF、路径解析、安全存储。
- Windows/桌面单实例、窗口/平台能力。
- WebView、分享、文件选择、权限、动态颜色等第三方插件能力。

平台工程职责：

- `android/`：包名、签名、SDK、ABI、原生依赖、资源和 Android 插件接入。
- `ios/`：Runner、Pods、权限宏、entitlements 和 iOS 打包。
- `macos/`：macOS Runner、Pods、桌面宿主。
- `windows/`：Windows Runner、CMake、资源、MSIX 配置。
- `ohos/`：OpenHarmony 宿主工程。

## 本地插件 `plugins/rhttp`

`plugins/rhttp` 是本仓库内的路径依赖，提供 rhttp 和 Flutter/Rust Bridge 相关能力。应用层通过 `package:rhttp/rhttp.dart` 初始化并创建兼容客户端。

维护要点：

- 修改插件 Dart/Rust 接口后，在 `plugins/rhttp` 内执行依赖恢复和代码生成。
- 应用主体也需要执行代码生成，确保引用的 generated/freezed 文件同步。
- Windows CI 会先构建插件生成代码，再构建应用。

## 国际化

国际化资源位于 `lib/l10n/*.arb`，生成文件位于 `lib/src/generated/i18n/`。

维护要点：

- 新增展示文案应进入 ARB 文件，避免硬编码散落。
- 修改 ARB 后执行 Flutter 生成流程。
- 多语言不完整时，应至少保证默认语言可用，并标记待补翻译。

## 构建与发布链路

当前仓库内可见的自动化构建包括：

- `.github/workflows/build_ios.yml`：手动触发 iOS 无签名 IPA 构建。
- `.github/workflows/build_windows.yml`：push、PR、手动触发 Windows release 构建；tag 触发 GitHub Release 发布。

Windows 工作流使用 GitVersion 计算版本，构建 release 二进制；存在证书 secret 时额外生成 MSIX 和安装器。

Android、macOS、OpenHarmony 可通过对应 Flutter/平台命令本地构建，具体见部署文档。

## 二开接入边界

新增功能建议结构：

```text
lib/custom/
├── custom_entry.dart
├── custom_routes.dart
├── custom_menu.dart
├── custom_config.dart
├── pages/
├── widgets/
├── services/
├── models/
├── providers/
├── utils/
└── constants/
```

接入原则：

- 路由、菜单、依赖注入和配置开关是允许触碰原项目的主要入口。
- 原业务页面、网络层、模型层和全局 store 只做必要兼容，不做顺手重构。
- 自定义功能应可关闭、可删除、可迁移。
- 涉及平台工程的改动要说明签名、权限、分发和上游同步风险。

### PixEz Sync 客户端扩展

PixEz Sync 相关 Flutter 侧扩展集中在 `lib/custom/`：

- `lib/custom/services/sync_config.dart` 保存同步服务器、Wavelet AccessToken、同步周期与小说自动镜像开关。
- `lib/custom/services/sync_service.dart` / `sync_api.dart` 封装 `/api/pixez/**` 与 `/mirror/**` 请求，避免污染原 `ApiClient`。
- `lib/custom/services/novel_auto_mirror_service.dart` 负责小说详情页打开后的自动镜像入队判断；原小说页面只保留一处入口调用。
- `lib/custom/pages/sync_settings_page.dart` 承载“设置 -> 数据同步”的配置界面。

Flutter custom 层只在 service 中集中处理 Wavelet envelope：`error_msg == ""` 视为成功，页面不直接解析 envelope。认证头统一使用 `Authorization: Bearer <access_token>`，不再配置 PixEz Basic Auth。

小说自动镜像只在客户端配置启用、同步服务器地址存在且“自动镜像小说”开关开启时发起。入队请求仍使用后端 `POST /api/pixez/novels/{novel_id}/mirror`，后端通过 PixEz read-model 和 Asynq 任务状态避免重复入队。

## 伴生后端服务 (PixezServer/Wavelet)

当前同步后端位于 `PixezServer/`，基于 Wavelet 的 Gin 路由、GORM、goose 双方言迁移、AccessToken 登录体系、Asynq 任务、TaskExecution 可观测性和 Upload/File 存储能力实现。

职责边界：

- `internal/apps/pixez/`：PixEz HTTP handler、mirror 读取、收藏 removed 查询、镜像管理 API 和 PixEz task handler。
- `internal/service/pixez/`：Pixiv App API 请求、token refresh、placeholder 检测、收藏导出、镜像处理、Legacy 导入和 Upload 映射。
- `internal/db/migrator/goose/`：PostgreSQL/SQLite 双方言 PixEz 表结构迁移。
- `internal/task/`：PixEz Asynq 类型、Admin 可下发任务元数据、handler 注册和 worker mux。

`/api/pixez/**` 保留业务路径并统一返回 `{ "error_msg": "", "data": ... }`；`/mirror/**` 也需要 Wavelet AccessToken，但返回 Pixiv 形态 JSON 或二进制文件，不套系统 envelope。

详细设计见 [pixez-server-wavelet.md](pixez-server-wavelet.md)，包含完整数据模型、任务体系与 API 接口说明。旧 [pixez-sync-backend.md](pixez-sync-backend.md) 为已废弃的独立后端 legacy 参考。
