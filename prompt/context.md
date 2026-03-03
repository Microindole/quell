# Quell 项目上下文

## 项目简介
Quell 是一个基于 Go 语言开发的工具，主要用于 Bilibili 视频下载和管理。它支持两种界面模式：
- **TUI 模式**: 使用 Bubbletea + Lipgloss 构建的终端界面（默认启动）
- **GUI 模式**: 使用 Wails v2 构建的桌面应用（`-gui` 参数启动）

## 技术栈
- **语言**: Go (1.25.7)
- **GUI 框架**: [Wails v2](https://wails.io/) (WebView2 渲染，无边框窗口 + 自定义标题栏)
- **TUI 框架**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **核心依赖**:
  - FFmpeg (用于音视频合并)
  - Native Downloader (Go 实现，复刻 BBDown 逻辑并进行优化，无外部依赖)

## 下载与 API 逻辑参考 (BBDown)
通过分析 `bbdown_src`，项目在以下方面参考或复刻了 BBDown 的成熟逻辑：
- **多接口支持**: 涵盖 Web Wbi、TV、App、PGC (番剧) 以及 International API，支持杜比音效和无损音频。
- **签名算法**: 实现 Wbi (Web) 和 App 端的签名校验。
- **多线程下载**: 采用 HTTP Range 分段请求，支持并发下载。
- **鲁棒性**: 引入了分段重试、断点续传及多 CDN 链接备选。

## 关键目录结构
- `prompt/`: 存放 Agent 相关的提示词和上下文
  - `context.md`: 项目上下文（本文件）。**Agent 必须首先读取此文件**。
  - `skill/`: 存放按需读取的专项技能说明。
    - `maintenance/SKILL.md`: 自动维护规范。
    - `tui/SKILL.md`: 终端界面开发规范。
    - `bilibili_api/SKILL.md`: API 及签名逻辑规范。
    - `downloader/SKILL.md`: 下载与合并逻辑规范。
  - `task.md`: 实时任务进度。
  - `walkthrough.md`: 已完成任务总结。
- `internal/`: 内部逻辑模块
  - `config/`: 配置读写
  - `crawler/`: B站 API 爬取
  - `downloader/`: 视频下载器
  - `engine/`: 本地扫描与合并引擎
  - `domain/`: 领域模型
- `web/`: 前端资源（HTML/CSS/JS，Wails AssetServer 嵌入）
- `app.go`: Wails 后端绑定（Go 方法暴露给前端调用）
- `gui.go`: Wails 启动配置（窗口参数、绑定注册）
- `main.go`: 程序入口（TUI 模式）+ TUI 逻辑
- `main_cmds.go`: TUI 远程命令
- `embed.go`: 嵌入资源声明
- `wails.json`: Wails 项目配置
- `quell_config.json`: 用户配置文件

## GUI 架构 (Wails v2)
- **前端 -> 后端**: `go.main.App.MethodName()` (Promise)
- **后端 -> 前端**: `runtime.EventsEmit()` / `window.runtime.EventsOn()`
- **窗口**: 无边框 + 自定义标题栏 (CSS `--wails-draggable: drag`)
- **构建**: `wails dev` 开发 / `wails build` 生产
- **前端结构**: `web/css/`（base.css / layout.css / components.css / modal.css）+ `web/js/`（core.js / events.js / local.js / remote.js / settings.js）
- **注意**: `web/wailsjs/` 由 `wails dev` 自动生成，禁止手动修改

## 配置文件 (`quell_config.json`)
包含以下配置项：
- `bili_dir`: 视频下载保存路径
- `ffmpeg_path`: FFmpeg 执行程序路径
- `sessdata`: B站登录凭证（可选，解锁高画质）
- `output_format`: 合并导出格式，`"mp4"`（默认）或 `"mkv"`

## 近期进展（2026-03-03）
- 远程列表分页已支持：首页/尾页、跳页、每页数量（20/30/50）。
- 远程列表已支持多选并按队列顺序逐个下载。
- 远程下载不再写入额外 sidecar 标记文件（已移除 `*.quell.json` 机制）；“无本地缓存”判定基于是否能扫描到对应本地 B站缓存任务。

## 维护与使用规范
- **统一代码检查命令（跨平台）**:
  - 通用（推荐，三端一致）: `go run ./scripts/check <mode>`
  - `mode` 取值: `quick` / `quality` / `security` / `all`
  - Windows 便捷命令: `.\qc.cmd <mode>`（默认 `quick`）
  - macOS/Linux 便捷命令: `./qc <mode>`（首次若无执行权限先执行 `chmod +x qc`）
  - `security` 模式建议使用补丁工具链: `GOTOOLCHAIN=go1.25.7`
- **按需读取技能**: Agent 在涉及特定领域（如修改 UI 或调用 API）时，应主动读取 `prompt/skill/` 对应目录下的 `SKILL.md`。
- **自动同步**: 任何重大变更后，Agent 需同步更新 `prompt/` 内容，确保"记忆"持久化。
- **禁止使用 Emoji**: 在所有生成的代码、文档和回复中，严禁使用 Emoji 表情符号。
- **纯中文界面**: 界面文本（包括选项、Label、Button）必须使用纯中文，严禁出现 "中文 (English)" 这种括号备注形式。


*Ctx Updated: 2026-03-03 19:19:10*
