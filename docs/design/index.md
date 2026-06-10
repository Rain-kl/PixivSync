# Design Overview

本文档描述 PixEz Flutter 当前仓库的产品范围、系统边界、核心对象和目录结构。更细的模块拓扑见 [architecture.md](architecture.md)。

## 产品范围

PixEz Flutter 是使用 Flutter 编写的 pixiv 第三方客户端，面向 Android、iOS、Windows、macOS、Linux、OpenHarmony 等多平台运行环境。应用提供插画、漫画、小说、用户、收藏、关注、搜索、排行、历史、下载/保存、网络模式设置、主题与多语言等客户端能力。

本仓库同时包含：

- Flutter 应用主体。
- PixezServer/Wavelet 伴生后端，用于 PixEz 同步、镜像、收藏导出、任务与文件存储。
- 多平台原生宿主工程。
- 本地 Flutter 插件 `plugins/rhttp`。
- 国际化资源、图片资源、字体资源。
- GitHub Actions 构建工作流与 fastlane 元数据。
- AI/开发规范、设计文档、部署文档与参考手册。

## 系统边界

应用边界以内：

- Flutter UI、页面流转、主题、国际化与平台差异化界面。
- pixiv API/OAuth 请求封装、刷新 token 拦截、网络模式适配。
- 本地设置、账号、标签、历史、屏蔽、保存路径、任务等状态与持久化。
- 图片、ugoira、小说、WebView、分享、文件选择、系统设置、单实例等客户端功能。
- Android/iOS/desktop 原生集成与平台插件桥接。

应用边界以外：

- pixiv 官方服务与第三方图片/检索服务。
- 应用商店、GitHub Release、证书签名、分发渠道。
- 用户设备上的系统权限、网络环境和代理/DNS 能力。

## 设计文档索引

- [architecture.md](architecture.md)：Flutter 客户端、custom 扩展与 PixezServer 的模块拓扑。
- [pixez-server-wavelet.md](pixez-server-wavelet.md)：PixEzServer/Wavelet 伴生后端完整设计文档，含架构、数据模型、异步任务、API 接口与客户端接入。

## 核心对象

- `ApiClient`：pixiv App API 请求入口，封装请求头、缓存、刷新 token 拦截器与网络适配器。
- `OAuthClient`：pixiv OAuth 登录、授权码换 token、refresh token 续期入口。
- `UserSetting`：全局用户设置，包括主题、语言、网络模式等。
- `AccountStore`：账号状态、登录信息和账号加载。
- `SaveStore`：保存路径、下载/保存相关配置。
- `MuteStore`：屏蔽规则初始化和状态。
- `TagHistoryStore` / `NovelHistoryStore`：搜索标签和小说阅读历史。
- `TopStore`：顶层刷新/导航相关事件流。
- `Fetcher` / `IllustCacher`：后台抓取、图片缓存和预取相关能力。
- `SplashPage`：应用启动后的初始化和首屏分发入口。
- `Constants.isFluent`：Material UI 与 Fluent UI 分支选择。

## 当前仓库目录结构

以下结构基于当前仓库整理，省略 `.git/`、`.dart_tool/`、`build/`、IDE 缓存等生成或本地环境目录。

```text
.
├── AGENTS.md
├── README.md
├── LICENSE
├── analysis_options.yaml
├── l10n.yaml
├── pubspec.yaml
├── pubspec.lock
├── .fvmrc
├── .github/
│   ├── workflows/
│   ├── preview/
│   ├── board/
│   ├── FAQ.md
│   ├── README_en.md
│   └── README_id.md
├── .vscode/
├── .agent/
├── android/
├── ios/
├── macos/
├── windows/
├── ohos/
├── assets/
│   ├── emojis/
│   ├── fonts/
│   ├── images/
│   └── json/
├── docs/
│   ├── deployment/
│   ├── design/
│   ├── guideline/
│   ├── plan/
│   └── reference/
├── PixezServer/
│   ├── internal/apps/pixez/
│   ├── internal/service/pixez/
│   ├── internal/db/migrator/goose/
│   ├── internal/task/
│   └── frontend/
├── fastlane/
├── gradle/
├── lib/
│   ├── component/
│   ├── er/
│   ├── fluent/
│   ├── l10n/
│   ├── lighting/
│   ├── models/
│   ├── network/
│   ├── page/
│   ├── server/
│   ├── src/generated/
│   ├── store/
│   └── *.dart
├── plugins/
│   └── rhttp/
├── res/
├── test/
└── tools/scripts at root
```

## 目录作用说明

### 根目录

- `pubspec.yaml` / `pubspec.lock`：Flutter 包、资源、字体、MSIX、代码生成和本地插件依赖声明。
- `analysis_options.yaml`：Dart/Flutter 静态分析配置。
- `l10n.yaml`：Flutter 国际化生成配置。
- `.fvmrc`：项目期望使用的 Flutter SDK 版本来源。
- `README.md`：项目介绍、下载渠道、贡献信息和反馈入口。
- `AGENTS.md`：AI 代理接手入口，指向开发约束、设计、部署和计划文档。
- `comment_emoji2map.py`、`to_lower.dart`、`comment_element.html`：仓库级辅助脚本或资源转换文件。

### `lib/`

Flutter 应用主体。

- `main.dart`：应用启动入口，初始化 `rhttp`、缓存、桌面 sqlite、单实例、Fluent 配置和全局 stores，并挂载 `ProviderScope`。
- `constants.dart`、`utils.dart`、`exts.dart`、`i18n.dart`：全局常量、工具函数、扩展方法和国际化上下文入口。
- `*_plugin.dart`：平台能力桥接或插件封装，如剪贴板、深链、文档、设置、单实例、Windows、SAF 等。
- `component/`：Material 分支复用组件，如插画卡片、图片组件、空状态、ugoira 播放、HTML 渲染等。
- `page/`：Material UI 页面和页面级 store/notifier，覆盖启动、登录、首页、搜索、插画、小说、用户、关注、历史、主题、WebView、任务等功能。
- `fluent/`：Fluent UI 分支，包括桌面风格导航框架、组件和对应页面实现。
- `models/`：API 响应、本地持久化对象、业务实体与序列化模型。
- `network/`：Dio/rhttp 网络层、OAuth、App API、OneZero、网络模式、刷新 token 拦截器。
- `store/`：全局 MobX/状态对象，如账号、设置、收藏标签、保存、静音、顶层事件。
- `er/`：应用运行支撑服务与抽象，如图片源、缓存、下载/分享、host、更新、toast、leader。
- `lighting/`：沉浸/灯箱类展示状态和页面。
- `l10n/`：ARB 多语言资源。
- `src/generated/`：Flutter 国际化等生成文件。
- `server/`：本地服务能力，目前包含 `weiss_server.dart`。

### 平台工程目录

- `android/`：Android Gradle 工程、应用配置、原生模块、资源、签名和构建脚本。`android/app/build.gradle.kts` 决定包名、SDK、ABI split、签名和 `IS_GOOGLEPLAY` dart define。
- `ios/`：iOS Runner 工程、Podfile、权限宏、资源、entitlements 和 Xcode 配置。
- `macos/`：macOS Runner 工程、Podfile 和桌面宿主配置。
- `windows/`：Windows Runner、CMake 配置、资源和 `deploy.dart`。
- `ohos/`：OpenHarmony 工程配置、entry、hvigor 和 AppScope。

### 资源与插件

- `assets/images/`：应用图片资源。
- `assets/json/`：配置类 JSON 资源，例如 `host.json`。
- `assets/emojis/`：评论或表情资源。
- `assets/fonts/`：图标字体等字体资源。
- `plugins/rhttp/`：本地 Flutter 插件，包含 Dart API、平台目录、Rust crate、flutter_rust_bridge 配置和代码生成。
- `res/`：额外资源目录，供平台或辅助流程使用。

### 文档与发布辅助

- `docs/guideline/`：开发约束和低侵入二开准则。
- `docs/design/`：系统设计与架构文档。
- `docs/deployment/`：构建、部署和维护文档。
- `docs/reference/`：配置、命令行、参数等参考手册。
- `docs/plan/`：正在进行的实现计划和交接记录。
- `PixezServer/`：Wavelet 伴生后端，承载 PixEz 账号同步、镜像、收藏导出、Legacy 导入、Upload 存储和 Admin 任务。
- `.github/workflows/`：GitHub Actions 构建工作流，目前包含 iOS 和 Windows。
- `fastlane/`：移动端发布元数据。
- `test/`：Flutter 测试入口。

## 二开设计约束

本项目二次开发遵循低侵入原则：

- 新功能优先集中在 `lib/custom/` 或现有等价扩展目录。
- 原项目文件只做最小入口接入，如路由、菜单、依赖注入、配置开关。
- 自定义资源优先放入 `assets/custom/`，避免混入既有资源目录。
- 不随意修改 `lib/network/`、`lib/models/`、`lib/store/`、原页面和公共组件。
- 必须新增依赖时，仅修改必要声明，并说明维护成本和替代方案。

## 设计文档维护规则

- 新功能或重要特性开发前，应在 `docs/design/` 下补充设计或更新本文档。
- 涉及模块边界、状态流、网络层、平台宿主、构建分发方式变化时，同步更新 [architecture.md](architecture.md) 或部署文档。
- 事实不确定时使用 `[需要确认]` 标记，避免把推测写成确定设计。
