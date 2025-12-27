
# Quell ⚡

> 一个终端进程管理器 (TUI)。

```text
   ____  __  __________    __ 
  / __ \/ / / / ____/ /   / / 
 / / / / / / / __/ / /   / /  
/ /_/ / /_/ / /___/ /___/ /___
\___\_\____/_____/_____/_____/

```

**Quell** 是一个进程管理工具。它基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 框架构建。

## 🚀 安装指南

### 从源码安装

确保你安装了 Go 1.20+ 环境：

```bash
git clone https://github.com/Microindole/quell.git
cd quell
go build -ldflags="-s -w" -o quell .
```

### 直接运行

```bash
./quell
```

## ⌨️ 快捷键手册

Quell 支持以下快捷键：

### 基础导航

| 按键        | 功能                                   |
|-----------|--------------------------------------|
| `↑` / `k` | 上移光标                                 |
| `↓` / `j` | 下移光标                                 |
| `Enter`   | **查看详情** (包含实时波形图)                   |
| `Tab`     | 切换排序方式 (PID / CPU / Memory / Status) |
| `t`       | 切换 **树状视图 / 平铺视图**                   |

### 进程操作

| 按键      | 功能                    |
|---------|-----------------------|
| `Space` | **多选模式** (勾选/取消勾选当前行) |
| `x`     | **杀进程** (Kill) - 支持批量 |
| `s`     | **暂停进程** (Suspend)    |
| `c`     | **恢复进程** (Continue)   |

### 系统命令

| 按键       | 功能                                     |
|----------|----------------------------------------|
| `/`      | 进入命令模式 (支持 `/help`, `/pkill`, `/kill`) |
| `Esc`    | 清空选中状态 / 返回 / 退出                       |
| `q`      | 退出程序                                   |
| `Ctrl+C` | 强制退出                                   |

## ⚙️ 配置文件

Quell 会自动在用户目录下生成配置文件：

* **Windows**: `C:\Users\YourName\.quell\config.json`
* **Linux/Mac**: `~/.quell/config.json`

**主要保存内容：**

1. **用户偏好**：上次使用的排序方式、是否开启树状图。
2. **暂停列表**：你手动暂停的进程信息（PID + 创建时间戳）。这使得 Quell 即使在重启后，也能准确找回并标记那些被“挂起”的进程。

## 🛠️ 技术栈

* **UI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
* **Styling**: [Lip Gloss](https://github.com/charmbracelet/lipgloss)
* **System Info**: [gopsutil](https://github.com/shirou/gopsutil)

## 📄 License

[MIT License](./LICENSE)

