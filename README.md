# WinVNC Go（mygo 版）

基于 [MyGo](https://mygo.egoist.dev/) native UI 的轻量级 VNC 远程控制工具，支持通过 Redis 控制中心自动发现设备。界面由 MyGo 直接在 GPU 上绘制，无 WebView2 依赖，Windows 7 可直接运行。

## 功能

- **设备 ID 管理**：自动生成 6 位随机 ID，用于标识和查找设备
- **VNC 服务**：内置 UltraVNC，启动后自动生成密码并与 ID 绑定
- **Redis 控制中心**：可选接入 Redis，实现设备 ID ↔ IP 的自动注册与查询
- **自动重连**：Redis 断线后自动重连（5 秒心跳检测），VNC 服务停止后自动重启
- **远程连接**：输入对方 ID 即可发起 VNC 连接，控制中心不可用时支持手动输入 IP
- **托盘菜单**：右键多级菜单（刷新/复制 ID、连接、状态显示开关、退出）
- **自定义字体**：内嵌 Maple Mono（Win7 自动按用户安装降级）

## 环境要求

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | 1.27.1 | 构建工具链（Win7 用 `D:\env\go-legacy-win7`） |
| Redis | - | 可选，用于设备中心注册 |
| UPX | - | 可选，用于压缩最终产物 |

## 构建

```powershell
powershell -File build.ps1
```

- 使用 `D:\env\go-legacy-win7` 工具链构建 windows/amd64（可用 `-GoRoot` 覆盖）
- `-ldflags "-H windowsgui"`：GUI 子系统，启动无控制台黑窗
- 自动嵌入根目录 `app.ico` 作为 exe 图标
- 输出 `bin\winvncgo.exe`

开发时直接 `go run .` 或 `go build` 即可（会带控制台窗口，便于看日志）。

## 运行

把 `bin\winvncgo.exe` 放到 `winvnc.exe`、`winvncgo.json` 同目录运行（两者模板在 `libs\` 下）。

- 主窗口：系统标准标题栏，420×570 固定大小
- 托盘图标：右键打开多级菜单
- 日志：`winvncgo.log`（与可执行文件同目录）
- 单实例：重复启动会激活已有窗口后退出

## 项目结构

```
winvncgo/
├── main.go                  # 入口（调 tasks.Run）
├── tasks/
│   ├── main.go              # mygo 窗口与启动保障（日志 / panic 兜底 / 单实例）
│   ├── ui.go                # native UI 界面（原前端页面的重写）
│   ├── tray.go              # 托盘图标与多级菜单
│   ├── font.go              # Maple Mono 字体注册与 Win7 降级安装
│   ├── app.go               # 后端门面（状态回调、GetStatus/RefreshID/Connect）
│   ├── manager.go           # 核心业务（ID 生成、VNC 启停、Redis 注册）
│   ├── vnc.go               # UltraVNC 进程管理
│   ├── redis.go             # 原生 TCP 实现的 Redis 客户端
│   ├── config.go            # 配置文件加载（winvncgo.json）
│   ├── network.go           # 本机 IP 获取
│   ├── id.go                # 6 位随机 ID 生成
│   ├── types.go             # 状态事件结构体
│   ├── icons/               # 内嵌图标（刷新/复制 SVG、托盘 PNG）
│   └── fonts/               # 内嵌 Maple Mono（TTF）
├── libs/
│   ├── ultravnc/            # UltraVNC 服务端与 DLL
│   └── winvncgo.json        # 默认配置（含注释示例）
├── cmd/win7diag/            # Win7 环境诊断工具
├── app.ico                  # exe 图标（build.ps1 经 rsrc 嵌入）
├── build.ps1                # 构建脚本（输出 bin\winvncgo.exe）
├── go.mod
└── go.sum
```

Wails / WebView2 版本在 `windows7` 分支。

## 配置文件

`winvncgo.json` 放在可执行文件同目录下，格式如下：

```json
{
  "AppSettings": {
    // 格式：redis://用户名:密码@主机:端口/键前缀?ttl=分钟
    //
    // 无认证
    "ctl_center": "redis://192.168.1.100:6379/vnc?ttl=30"
    //
    // 仅密码（用户名留空）
    // "ctl_center": "redis://:mypassword@10.0.0.1:6379/vnc?ttl=30"
    //
    // 用户名 + 密码
    // "ctl_center": "redis://user:pass@192.168.1.100:6379/vnc?ttl=30"
  }
}
```

- `ctl_center`：Redis 连接串，支持 URL 格式；留空或不配置则禁用控制中心
- `ttl`：设备注册信息过期时间（分钟），默认 30
- 键前缀（路径部分）：Redis key 前缀，用于隔离不同项目，默认 `vnc:`

配置文件支持 `//` 行注释（程序运行时自动去除）。

未找到配置文件时使用默认配置（Redis `127.0.0.1:6379`，无密码）。

## 工作流程

1. 启动时生成 6 位随机 ID，写入 UltraVNC 配置作为密码
2. 启动 UltraVNC 服务（隐藏窗口，监听 5900 端口）
3. 若 Redis 可用，将 `ID → IP:5900` 注册到 Redis（带 TTL）
4. 定时轮询状态（每 2 秒）：VNC 运行状态、Redis 连接状态、是否有活跃连接，经 `onStatus` 回调推给界面
5. Redis 断线后通过心跳（5 秒）自动检测并重连；VNC 服务停止后自动尝试重启
6. 要控制其他设备时：输入对方 ID → 从 Redis 查询对方 IP → 调用 VNC Viewer 发起连接
7. 刷新 ID 时：若有活跃连接则跳过，否则停止 VNC → 删除旧 Redis 记录 → 生成新 ID → 重新启动

## 开发说明

- 界面为 MyGo native UI（package `ui`），纯 Go，无 HTML/JS/前端构建
- Redis 客户端为原生 TCP 实现（未使用第三方库）
- 日志输出到 `winvncgo.log`（与可执行文件同目录）
- Win7 兼容性：字体注册依赖 Win10 1703+ 的 DirectWrite API，Win7 自动降级为按用户安装
