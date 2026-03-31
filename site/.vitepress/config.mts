import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "Quell",
  description: "Bilibili 视频下载与管理神器",
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: '首页', link: '/' },
      { text: '快速开始', link: '/guide/getting-started' },
      { text: '配置指南', link: '/config' }
    ],

    sidebar: [
      {
        text: '使用手册',
        items: [
          { text: '快速开始', link: '/guide/getting-started' },
          { text: '常见问题', link: '/guide/faq' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/Microindole/quell' }
    ],

    footer: {
      message: '基于 MIT 协议开放源代码',
      copyright: 'Copyright © 2026-present Quell Project Group'
    },

    docFooter: {
      prev: '上一页',
      next: '下一页'
    },

    darkModeSwitchLabel: '主题模式',
    outlineTitle: '本页目录',
    sidebarMenuLabel: '菜单',
    returnToTopLabel: '返回顶部'
  }
})
