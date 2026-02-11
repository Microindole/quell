# 项目 Prompt 结构初始化完成

我已经在 `prompt` 目录下完成了以下操作：

1. **创建了 `skill` 目录**：用于后续存放针对 Agent 的专用技能或指令。
2. **创建并初始化了 `context.md`**：该文件包含了项目的基本背景、技术栈和目录结构说明，方便 Agent 快速对齐项目状态。

## 变更详情

- [NEW] `prompt/skill/maintenance/SKILL.md` (重构后的标准格式)
- [NEW] `prompt/skill/tui/SKILL.md` (专项：TUI 开发)
- [NEW] `prompt/skill/bilibili_api/SKILL.md` (专项：API 调用)
- [NEW] `prompt/skill/downloader/SKILL.md` (专项：下载逻辑)
- [DELETE] `prompt/skill/maintain.md` (原非标准格式文件)

## 验证结论

1. **标准化**: 所有技能均遵循 `文件夹/SKILL.md` 结构，并包含 YAML 元数据（name, description），符合 Antigravity 的原生技能加载规范。
2. **专业化分工**: 明确划分了 TUI、API 和下载逻辑三个核心领域，减少了冗余，提高了 Agent 读取的针对性。
3. **按需读取指引**: `context.md` 明确要求 Agent 在进入特定领域任务前先读取对应技能，确保开发一致性。

目前的 `prompt` 目录结构：
```text
prompt/
├── context.md (已更新读取规范)
├── task.md
├── walkthrough.md
└── skill/
    ├── maintenance/
    ├── tui/
    ├── bilibili_api/
    └── downloader/
```

`context.md` 已经包含了项目的主要信息（如 Go 版本、核心依赖和配置文件说明），您可以根据需要进一步完善该文件。
