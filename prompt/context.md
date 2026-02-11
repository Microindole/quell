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
  - `context.md`: 项目上下文（本文件）。**Agent 需在完成重大变更后自动更新此文件**。
  - `skill/`: 存放 Agent 的专用技能或指令。**Agent 需根据需要维护此处内容**。
  - `task.md`: 同步自 Agent 内部，记录当前或最近任务的进度。
  - `walkthrough.md`: 同步自 Agent 内部，记录已完成任务的总结。
  - `implementation_plan.md`: (如有) 同步自 Agent 内部，记录待执行的方案。
- `bbdown_src/`: 可能包含 BBDown 的源码或相关工具
- `internal/`: 内部逻辑模块
- `quell_config.json`: 项目配置文件
- `main.go`: 程序入口

## 配置文件 (`quell_config.json`)
包含以下配置项：
- `bili_dir`: 视频下载保存路径
- `ffmpeg_path`: FFmpeg 执行程序路径
- `bbdown_path`: BBDown 执行程序路径

## 维护规范
- **自动更新**: Agent 在进行重大功能开发、依赖变更或架构调整后，必须主动检查并更新 `prompt/` 目录下的相关内容。
- **一致性**: 确保 `context.md` 的项目描述与实际代码状态保持同步。
