# Quell 项目上下文

## 项目简介
Quell 是一个基于 Go 语言开发的工具，主要用于 Bilibili 视频下载和管理。它使用了 Bubbletea 和 Lipgloss 构建了终端用户界面 (TUI)。

## 技术栈
- **语言**: Go (1.25.2)
- **界面**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **核心依赖**:
  - FFmpeg (用于音视频合并)
  - BBDown (外部下载引擎)

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
- `bbdown_src/`: 可能包含 BBDown 的源码或相关工具
- `internal/`: 内部逻辑模块
- `quell_config.json`: 项目配置文件
- `main.go`: 程序入口

## 配置文件 (`quell_config.json`)
包含以下配置项：
- `bili_dir`: 视频下载保存路径
- `ffmpeg_path`: FFmpeg 执行程序路径
- `bbdown_path`: BBDown 执行程序路径

## 维护与使用规范
- **按需读取技能**: Agent 在涉及特定领域（如修改 UI 或调用 API）时，应主动读取 `prompt/skill/` 对应目录下的 `SKILL.md`。
- **自动同步**: 任何重大变更后，Agent 需同步更新 `prompt/` 内容，确保“记忆”持久化。
