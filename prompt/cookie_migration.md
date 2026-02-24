# Bilibili 完整 Cookie 支持方案

## 背景与目的
由于 B 站风控策略升级，单纯依靠 `SESSDATA` 已难以通过部分高画质接口的校验（如 WBI 签名与特定风险控制）。为了确保下载器的稳定性并解锁最高画质，需要支持传入完整的 Cookie 字符串。

## 配置变更
在 `quell_config.json` 中，原有的 `sessdata` 字段保持兼容，但新增 `cookie` 字段用于存放完整的 Cookie 字符串。

```json
{
  "sessdata": "...",
  "cookie": "buvid3=...; _uuid=...; SESSDATA=...; bili_jct=...;"
}
```

## 实现细节
1. **优先级逻辑**: 
   - 程序启动时优先检查 `cookie` 字段。
   - 如果 `cookie` 存在且不为空，则直接使用该完整字符串填充请求头 `Cookie`。
   - 如果 `cookie` 为空但 `sessdata` 存在，则构造最小化 Cookie：`SESSDATA=...`。
2. **请求头注入**:
   - 在 `internal/downloader` 和 `internal/crawler` 的所有请求中，确保 `User-Agent` 与 Cookie 保持一致。
   - 建议使用常用的 Web 端 User-Agent 以降低风控风险。

## 风控应对策略
- **WBI 签名**: 完整 Cookie 包含 `buvid3` 等字段，有助于生成更稳定的 WBI 签名。
- **验证码处理**: 当遇到 412 (Precondition Failed) 或验证码拦截时，提醒用户更新最新获取的完整 Cookie。
- **自动解析**: 核心逻辑应具备从完整 Cookie 字符串中提取 `SESSDATA` 和 `bili_jct` (CSRF) 的能力，以便于需要特定字段的 API 调用。

## 验证步骤
1. 在浏览器登录 B 站，通过开发者工具 (F12) 复制完整的 `document.cookie`。
2. 填入 `quell_config.json`。
3. 启动 Quell，验证是否能成功解析 4K/杜比画质的 `playurl`。
