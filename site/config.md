# 配置与依赖说明

Quell 采用 JSON 格式存储配置。首次启动程序时，会在根目录自动生成 `quell_config.json`。

## FFmpeg 核心依赖 (重要)

由于 B 站的高画质视频采用音视频流分离技术，Quell 需要通过 FFmpeg 对其进行高效的无损合并。

### 您有两种选择：

1.  **Bundled (推荐)**: 
    *   下载文件名带有 `bundled` 的版本。
    *   **优点**: 无需任何配置。Quell 会自动加载并使用内置的 FFmpeg 环境。
    *   **适用场景**: 绝大多数用户。

2.  **Standard (手动配置)**:
    *   如果您希望使用系统已有的 FFmpeg 或自定义版本。
    *   **优点**: 二进制文件体积极小。
    *   **配置方式**: 
        *   在 `quell_config.json` 中配置 `ffmpeg_path` 指向可执行文件（如 `C:\\tools\\ffmpeg.exe`）。
        *   或者将 FFmpeg 加入系统 PATH 环境变量。

---

## 配置文件字段详解 (`quell_config.json`)

### bili_dir
*   **类型**: String
*   **说明**: B 站本地缓存目录（用于扫描已有的手机或 PC 客户端缓存任务）。

### output_dir
*   **类型**: String
*   **说明**: 视频合并后的导出目录。留空则默认使用 `bili_dir`。

### output_format
*   **类型**: String
*   **说明**: 导出格式。可选值为 `"mp4"` (推荐) 或 `"mkv"`。

### sessdata
*   **类型**: String
*   **说明**: Bilibili 的登录凭证。解锁 1080P/4K 画质的关键。

---

## 修改配置

*   **GUI 模式**: 直接在设置页面修改并保存，程序会自动同步到配置文件。
*   **手动编辑**: 关闭程序后，使用任一文本编辑器打开 `quell_config.json`，修改后保存并重新启动程序。

> **提示**: 在 JSON 中编写路径时，请务必使用双反斜杠进行转义（例如 `D:\\Videos`）。
