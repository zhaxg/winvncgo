# WinVNC Go — Windows 7 支持说明

本分支（`windows7`）在保持功能不变的前提下，使程序可以运行在 **Windows 7 SP1 x64** 上。

## 系统要求

| 项目 | 要求 |
|------|------|
| 系统 | Windows 7 SP1 x64（Windows 8 / 8.1 亦适用同一方案） |
| 补丁 | KB2999226（Universal C Runtime）、KB4474419（SHA-2 代码签名，安装 WebView2 时需要） |
| WebView2 运行时 | **109.0.1518.78 / 109.0.1518.140**（最后一个支持 Win7 的版本），以固定版本形式随程序分发 |
| 权限 | 管理员（winvnc 捕获屏幕需要，UAC 隔离桌面限制） |
| 架构 | x64（UltraVNC 依赖为 64 位） |

## 产物内容

```
winvncgo-win7\
├── winvncgo.exe            # 主程序（Windows 7 兼容构建）
├── winvnc.exe / vncviewer.exe / *.dll   # UltraVNC 1.8.2.4
├── winvncgo.json / ultravnc.ini         # 配置文件
├── webview2\               # （可选）固定版本 WebView2 109 运行时
└── README-win7.md
```

## WebView2 运行时（必须）

界面基于 Microsoft Edge WebView2。微软对 Windows 7/8/8.1 的支持止步于 **109.x**（2023 年 1 月），
此后的 WebView2 在线安装器/运行时**都无法**在 Win7 上安装，只能使用 109 固定版本：

1. **推荐**：把固定版本运行时目录（内含 `msedgewebview2.exe` 和 `EBWebView\`）放到程序目录下的 `webview2\`，开箱即用；
2. 或者在目标机器上安装过 109.x 版本的 WebView2 运行时（系统级），程序会自动使用它。

运行时获取：运行 `scripts\fetch-webview2-109.ps1`（在能联网的机器上）下载并解压到 `build\webview2-runtime\`，
然后执行 `.\build.ps1 win7` 重新打包，运行时会被自动打进 `webview2\` 目录。

程序启动时会自动选择运行时，无需手动配置。检测顺序：
1. `webview2\` 目录或 `Microsoft.WebView2.FixedVersionRuntime.*\` 目录
2. 系统已安装的 WebView2（109.x）
3. 都找不到时会弹出中文提示（不会走在线安装流程）

## 与主分支的差异

| 改动 | 说明 |
|------|------|
| Mica 背景效果仅 Windows 11 启用 | Win7/8/10 上透明窗口背景会导致显示异常，自动跳过并记录日志 |
| WebView2 固定版本运行时支持 | 自动查找 `webview2\` 目录并通过 `WebviewBrowserPath` 指定 |
| 明确的缺运行时报错 | 缺少 WebView2 时给出中文提示，而不是触发必败的在线安装 |

其余功能（设备 ID、VNC 服务管理、Redis 控制中心、自动重连、远程连接）与主分支完全一致。

## 已知限制

- **无 Mica/云母背景**：Windows 11 专属视觉效果，Win7 上界面为普通背景。
- **无系统通知**：Windows 10+ 的 Toast 通知在 Win7 不可用（本程序未使用该功能）。
- **屏幕捕获方式**：UltraVNC 在 Win7 上使用 vnchooks 钩子捕获（Win8+ 的桌面复制引擎 ddengine 不可用），性能略低但功能完整。
- **仅 x64**：UltraVNC 运行库为 64 位。

## 在 Windows 7 上的验证要点

1. 右键 `winvncgo.exe` → 以管理员身份运行；
2. 窗口正常显示（不是白屏/黑屏）→ WebView2 运行时加载成功；
3. 设备 ID 显示、VNC 状态指示灯正常；
4. `winvnc.exe` 服务启动后，用另一台机器输入 ID 可连接并看到画面；
5. Redis 不可用时界面给出提示，不影响其他功能。
