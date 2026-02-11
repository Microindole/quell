---
name: maintenance
description: 负责项目 prompt 目录及文档的自动化同步与维护
---

# 项目维护技能 (Maintenance Skill)

## 目标
确保项目的所有上下文信息（包括 `context.md`、`task.md`、`walkthrough.md` 等）与代码状态高度一致，防止 Agent 在长对话或新对话中丢失上下文。

## 更新触发时机
1. **架构/文件结构变更**: 立即同步 `context.md` 中的目录树。
2. **依赖项增减**: 更新 `context.md` 中的技术栈。
3. **关键功能交付**: 更新 `context.md` 中的功能描述。
4. **阶段性总结**: 将 Antigravity 的内部 Artifacts (`task.md`, `walkthrough.md`, `implementation_plan.md`) 复制到项目 `prompt/` 目录。

## 操作规范
- 路径引用需使用绝对路径或基于项目根目录的相对路径。
- 文档内容应简洁明了，避免冗余。
