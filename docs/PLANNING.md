# Quell - Project Roadmap & Design Doc

> **Project Vision:** > A minimalist, keyboard-centric terminal tool to effortlessly manage ports and processes.
> *"Quell the chaos. Free the port."*

## 1. Core Philosophy (核心理念)

* **Keyboard First:** 所有操作必须能通过键盘高效完成，无需鼠标介入。
* **Zero Dependency:** 单一二进制文件交付，无运行时依赖 (No Python/Node/JVM required)。
* **Fast & Furious:** 启动速度必须在毫秒级，占用极低内存。
* **Safety:** 在执行破坏性操作（Kill）前提供清晰的视觉反馈，防止误杀。

## 2. Technical Architecture (技术架构)

遵循 **Domain-Driven Design (DDD)** 简化版，保持代码解耦。

```text
quell/
├── cmd/                # 程序入口
├── internal/
│   ├── domain/         # 核心业务实体 (Process, Connection, Signal)
│   ├── sys/            # 操作系统交互层 (gopsutil 封装, syscall)
│   ├── tui/            # 表现层 (Bubble Tea Model/Update/View)
│   │   ├── components/ # 可复用 UI 组件 (List, Modal, StatusBar)
│   │   └── pages/      # 不同页面 (ProcessList, DetailView)
│   └── config/         # 配置文件处理 (Viper/YAML)
└── scripts/            # CI/CD 构建脚本

```

---

## 3. Development Phases (开发阶段规划)

### Phase 0: MVP (Current Status) ✅

* [x] **Core:** 扫描 TCP 监听端口。
* [x] **Core:** 基于 PID 杀进程 (SIGKILL)。
* [x] **UI:** 基础列表展示与过滤 (Fuzzy Search)。
* [x] **UI:** 简单的状态栏反馈。

### Phase 1: Usability & Precision (可用性增强) 🚧 **(Next Step)**

*目标：让工具变得顺手，不仅能杀，还能看清楚再杀。*

* [ ] **Feature: 详细信息面板 (Detail View)**
* 按 `Enter` 进入详情页，显示：
* 完整命令行参数 (`cmdline`)。
* 启动时间、运行用户。
* 内存占用 (RSS/VMS) 和 CPU 使用率。
* 该进程打开的所有端口 (不仅是当前选中的)。




* [ ] **Feature: 剪贴板支持**
* 按 `y` 复制当前选中的 PID。
* 按 `c` 复制完整的 Command Line。


* [ ] **UX: 优雅退出 (Graceful Kill)**
* 默认发送 `SIGTERM` (让程序有机会保存数据)。
* 按 `X` (大写) 强制发送 `SIGKILL`。


* [ ] **UI: 列表美化**
* 根据端口号或协议类型显示不同颜色的 Icon (如 HTTP, DB, SSH)。



### Phase 2: Batch & Power Tools (批量与高级功能)

*目标：提升效率，应对复杂场景。*

* [ ] **Feature: 多选模式 (Multi-Select)**
* 按 `Space` 标记多个进程。
* 一键 `Kill Selected`。


* [ ] **Feature: 树状视图 (Tree View)**
* 展示父子进程关系 (如 `nginx: master` -> `nginx: worker`)。
* 支持“杀掉整个进程树” (Kill Tree)。


* [ ] **Feature: 实时监控模式**
* 列表默认是静态快照。增加 `Live Mode` (按 `r` 开启)，每秒自动刷新 CPU/内存变化。


* [ ] **System:** 支持 macOS/Linux 的特异性处理 (如 macOS 下获取端口权限的特殊逻辑)。

### Phase 3: Network Insights (网络透视)

*目标：不仅管理进程，更是一个轻量级网络分析器。*

* [ ] **Feature: 流量嗅探 (Sniffer Lite)**
* 简单的带宽监控：显示当前端口的 Upload/Download 速率。


* [ ] **Feature: 远程连接查看**
* 不仅显示 LISTEN 端口，还能切换视图显示 ESTABLISHED 连接 (查看到底谁连了我的数据库)。


* [ ] **Feature: Port Knocking / Availability Test**
* 选中端口，按 `t` 进行本地连接测试 (Ping/Dial)，验证服务是否假死。



### Phase 4: Ecosystem & Distribution (生态与分发)

*目标：让所有人都能轻松安装和配置。*

* [ ] **Config:** 支持 `~/.config/quell/config.yaml`。
* 自定义快捷键绑定。
* 自定义配色主题 (Theme)。
* 常用端口别名 (如 8080 -> "Dev Server")。


* [ ] **Distribution:**
* GitHub Actions 自动构建 Release。
* 支持 `brew install quell` / `scoop install quell`。


* [ ] **Remote Mode (SSH):**
* 利用 Bubble Tea 的 SSH 能力，通过 `ssh quell.yourserver.com` 直接在终端打开远程服务器的 Quell 界面 (无需安装二进制)。



---

## 4. Design Details (设计细节)

### UI/UX 交互规范

| 按键 | 动作 | 说明 |
| --- | --- | --- |
| `j` / `↓` | 下移 | Vim 风格导航 |
| `k` / `↑` | 上移 | Vim 风格导航 |
| `/` | 搜索 | 激活模糊搜索框 |
| `Enter` | 详情 | 进入详情/侧边栏 |
| `x` | Terminate | 发送 SIGTERM (温和) |
| `Shift+X` | Kill | 发送 SIGKILL (强制) |
| `Space` | 标记 | 多选标记 |
| `?` | 帮助 | 显示快捷键提示 |

### 错误处理策略

* **权限拒绝 (Permission Denied):** 不要崩溃，弹出一个红色的 Toast/Modal，提示用户 `sudo` 或以管理员运行。
* **进程不存在:** 如果杀进程时进程已消失，静默刷新列表，提示 "Process already gone"。

---

## 5. Technology Stack Drill-down

* **Language:** Go 1.21+
* **TUI Framework:** Charmbracelet (Bubble Tea, Lip Gloss, Bubbles)
* **System Info:** `gopsutil/v3`
* **Config:** `viper` (后期引入)
* **Build Tool:** `GoReleaser`
