# 项目 Prompt 结构初始化完成

我已经在 `prompt` 目录下完成了以下操作：

1. **创建了 `skill` 目录**：用于后续存放针对 Agent 的专用技能或指令。
2. **创建并初始化了 `context.md`**：该文件包含了项目的基本背景、技术栈和目录结构说明，方便 Agent 快速对齐项目状态。

## 变更详情

- [NEW] [context.md](file:///d:/works/quell/prompt/context.md) (包含新的维护规范)
- [NEW] [maintain.md](file:///d:/works/quell/prompt/skill/maintain.md) (定义了 Agent 的更新职责)
- [NEW] [task.md](file:///d:/works/quell/prompt/task.md) (本地同步的进度文档)
- [NEW] [walkthrough.md](file:///d:/works/quell/prompt/walkthrough.md) (本地同步的总结文档)
- [NEW] `prompt/skill/` 目录

## 验证结论

1. **`context.md` 更新**: 增加了“维护规范”章节，明确了 Agent 自动同步项目状态的义务，并列出了同步文件的目录结构。
2. **新增维护技能**: 在 `prompt/skill/maintain.md` 中详细列出了触发更新的时机，包括周期性同步 Antigravity 内部 Artifacts。
3. **Artifacts 本地化**: 成功将 `task.md` 和 `walkthrough.md` 拷贝到 `prompt/` 目录下，确保上下文在本地有记录。

目前的 `prompt` 目录结构：
```text
prompt/
├── TODO.md
├── context.md
├── task.md
├── walkthrough.md
└── skill/
    └── maintain.md
```

`context.md` 已经包含了项目的主要信息（如 Go 版本、核心依赖和配置文件说明），您可以根据需要进一步完善该文件。
