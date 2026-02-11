---
name: bilibili_api
description: Bilibili 平台 API 的调用规范与 WBI 签名处理
---

# Bilibili API 技能 (Bilibili API)

## 接口范围
- 视频信息查询 (Web/App API)
- 播放链接获取 (WBI 签名认证)
- 用户信息与收藏夹同步

## 关键流程: WBI 签名
1. 获取 `img_key` 和 `sub_key` (通常由 `nav` 接口返回)。
2. 对参数进行排序并根据算法生成 `wts` 和 `w_rid`。
3. **注意**: WBI 签名具有时效性，需在调用前动态生成。

## 代码规范
- 所有的 API 调用应封装在 `internal/biliapi` 目录下。
- 使用强类型的 Struct 来解析 JSON 响应。
- 必须包含错误处理，特别是针对 403 Forbidden 和 API 速率限制。
