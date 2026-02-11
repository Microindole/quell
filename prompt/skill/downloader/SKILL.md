---
name: downloader_logic
description: 视频/音频下载流程、多线程优化及 FFmpeg 封装
---

# 下载逻辑技能 (Downloader Logic)

## 核心流程
1. **多线程下载**: 使用 HTTP Range 请求进行分片下载。
2. **下载器隔离**: 抽象 `Downloader` 接口，以便在原生实现和 BBDown 外部调用之间切换。
3. **合并 (Merging)**: 使用 FFmpeg 对音视频轨道进行流式合并或后期重封装。

## FFmpeg 调用规范
- 优先寻找系统环境变量中的 `ffmpeg`。
- 命令应简洁: `ffmpeg -i video.m4s -i audio.m4s -c copy output.mp4 -y`。
- 处理标准错误流以获取进度反馈。

## 安全与健壮性
- 检查磁盘空间。
- 处理断点续传。
- 任务完成后清理临时分片文件。
