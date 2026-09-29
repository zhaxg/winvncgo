# WinVNC Go — Windows 7 支持说明

本分支（`windows7`）在保持功能不变的前提下，使程序可以运行在 **Windows 7 SP1 x64** 上。

> **技术栈**：Go 1.20.14（官方支持 Win7 的最后一个 Go 版本）+ [wails-win7](https://github.com/cllsm/wails-win7)
> （Wails 的 Win7 兼容分叉）。主分支的 Go 1.26 + Wails v2.15 在 Win7 上运行时初始化即崩溃，
> 因此 Win7 构建固定使用上述组合；构建脚本 `build.ps1 win7` 会自动检查工具链。

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
├── winvncgo.exe            # 主程序（Go 1.20 + wails-win7 构建）
├── winvncgo-console.exe    # 同款程序的控制台版（启动失败时黑窗口会显示错误）
├── win7diag.exe            # 诊断工具（控制台），启动失败时先运行它
├── win7diag-x86.exe        # 32 位版诊断工具（判断系统架构用）
├── check-win7.bat          # 一键诊断：自动收集信息到 diag-report.txt
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

## 启动失败排查（重要）

程序是 GUI 程序，失败时没有任何控制台输出。按下面顺序排查：

### 0. 最省事：一键诊断

双击 `check-win7.bat`：它会自动运行诊断工具和控制台版主程序、记录退出码、
抓取事件日志，汇总到 `diag-report.txt` 并用记事本打开。把这个文件发回即可。

### 1. 先看日志

启动后（或启动失败后）看程序目录下的 `winvncgo.log`。日志从进程最早期开始记录：
Go 版本、系统版本、WebView2 检测结果、每一步的启动进度。

- **日志完全没生成** → 进程根本没跑起来（Go 运行时/系统补丁问题，见第 2 步）；
- **日志停在 "进入 wails.Run" 之后** → Wails/WebView2 阶段失败，看 `wails.Run 失败` 行；
- **有 panic 堆栈** → 把堆栈发回分析。

启动失败时程序还会弹出错误对话框（不会无声退出）。

### 2. 运行诊断工具 win7diag.exe

把 `win7diag.exe` 放到 winvncgo.exe 同目录，在 **cmd 命令行** 里运行：

```
cd /d 程序目录
win7diag.exe
```

它是控制台程序，即使 Go 运行时崩溃也会把错误打印在屏幕上。
结果同时写入 `win7diag.log`，请把该文件发回。重点看：

| 检查项 | 异常表现 | 处理 |
|--------|----------|------|
| `[1] Go 运行时` | 屏幕出现 `fatal error: ... not found` | Go 运行时与系统不兼容，见下方补丁要求 |
| `[2] 系统版本` | 不是 6.1 build 7601+ | 需要 Win7 SP1 |
| `[3] 进程架构` | 32 位系统 | winvncgo.exe 仅支持 64 位 Win7 |
| `[4] LoadLibraryExW` | 失败 | **缺 KB2533623**，Go 程序会启动即崩溃 |
| `[7] WebView2` | 注册表无记录且无 webview2\ 目录 | 安装 WebView2 109.x 或放置 webview2\ 目录 |
| `[7] WebView2` | pv=110+ | 版本过高，Win7 只支持 109.x |

### 3. 系统补丁要求

| 补丁 | 作用 |
|------|------|
| Windows 7 SP1 | 基线 |
| **KB2533623** | LoadLibraryExW 安全加载支持，**缺失时 Go 程序无法启动** |
| KB2999226 | Universal C Runtime（WebView2/Chromium 需要） |
| KB4474419 / KB4490628 | SHA-2 代码签名支持（安装 WebView2 时需要） |

### 4. 其他检查

- 右键 `winvncgo.exe` → 以管理员身份运行（winvnc 捕获屏幕需要）；
- 杀毒软件是否拦截了 `winvncgo.exe` 或 `win7diag.exe`；
- 如果 `win7diag.exe` 正常但 `winvncgo.exe` 起不来：把 `winvncgo.log` 和 `win7diag.log` 一起发回分析。



1. 右键 `winvncgo.exe` → 以管理员身份运行；
2. 窗口正常显示（不是白屏/黑屏）→ WebView2 运行时加载成功；
3. 设备 ID 显示、VNC 状态指示灯正常；
4. `winvnc.exe` 服务启动后，用另一台机器输入 ID 可连接并看到画面；
5. Redis 不可用时界面给出提示，不影响其他功能。
