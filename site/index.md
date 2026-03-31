---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "Quell"
  text: "Bilibili 下载艺术"
  tagline: "极速、简洁、强大。支持 TUI 和 GUI 双模式。"
  image:
    src: /package.svg
    alt: Quell Logo
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: GitHub 仓库
      link: https://github.com/Microindole/quell

features:
  - title: 多线程下载
    details: 内置 HTTP Range 分段请求，极速榨干带宽，支持断点续传。
    icon:
      src: /speed.svg
  - title: 智能合并
    details: 自动识别并跳过 B 站私有文件头，一键合并音视频轨道。
    icon:
      src: /merge.svg
  - title: 双模共生
    details: 无论是追求效率的终端党还是喜爱直观的视觉控，都能找到归宿。
    icon:
      src: /layout.svg
  - title: 内置 FFmpeg
    details: 支持 Bundled 版本，无需手动配置环境，开箱即用。
    icon:
      src: /package.svg

---

<div class="showcase">
  <h2 class="showcase-title">界面全览</h2>
  <div class="showcase-grid">
    <div class="showcase-item">
      <img class="light" src="/assets/远程视频下载-UID.png" alt="远程视频下载" />
      <img class="dark" src="/assets/远程视频下载-UID-暗.png" alt="远程视频下载" />
      <p>智能远程视频搜索与下载</p>
    </div>
    <div class="showcase-item">
      <img class="light" src="/assets/本地缓存视频查看.png" alt="本地视频管理" />
      <img class="dark" src="/assets/本地缓存视频查看-暗.png" alt="本地视频管理" />
      <p>直观的本地视频管理界面</p>
    </div>
  </div>
</div>

<div class="tech-stack">
  <a href="https://go.dev" target="_blank" rel="noopener noreferrer">
    <img src="https://cdn.simpleicons.org/go/00ADD8" alt="Go" class="tech-icon" />
  </a>
  <a href="https://wails.io" target="_blank" rel="noopener noreferrer">
    <img src="https://cdn.simpleicons.org/wails/0175C2" alt="Wails" class="tech-icon" />
  </a>
  <a href="https://ffmpeg.org" target="_blank" rel="noopener noreferrer">
    <img src="https://cdn.simpleicons.org/ffmpeg/007808" alt="FFmpeg" class="tech-icon" />
  </a>
</div>

<div class="cta-section">
  <h2 class="showcase-title" style="margin-top: 60px;">准备好开始了吗？</h2>
  <p style="margin-bottom: 30px; color: var(--vp-c-text-2)">加入 Quell 用户行列，开启高效的 Bilibili 管理体验。</p>
  <a href="/guide/getting-started" class="VPButton brand">立即开始</a>
</div>

<style>
.showcase { padding: 40px 0; text-align: center; }
.showcase-title { font-size: 32px; font-weight: 800; margin-bottom: 40px; }
.showcase-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 40px; }
.showcase-item img { border-radius: 16px; border: 1px solid var(--vp-c-divider); transition: all 0.5s cubic-bezier(0.2, 1, 0.2, 1); width: 100%; height: auto; }
.showcase-item img:hover { transform: translateY(-12px) scale(1.02) rotate(2deg); border-color: var(--vp-c-brand); box-shadow: 0 25px 50px rgba(0,0,0,0.4); }
.showcase-item p { margin-top: 20px; color: var(--vp-c-text-2); font-size: 15px; font-weight: 500; }

html:not(.dark) .showcase-item img.dark { display: none; }
html.dark .showcase-item img.light { display: none; }
</style>
