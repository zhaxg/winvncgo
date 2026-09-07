# WinVNC Go

基于 [Wails v2](https://wails.io/) 的轻量级 VNC 远程控制工具，支持通过 Redis 控制中心自动发现设备。

## 功能

- **设备 ID 管理**：自动生成 6 位随机 ID，用于标识和查找设备
- **VNC 服务**：内置 UltraVNC，启动后自动生成密码并与 ID 绑定
- **Redis 控制中心**：可选接入 Redis，实现设备 ID ↔ IP 的自动注册与查询
- **远程连接**：输入对方 ID 即可发起 VNC 连接，控制中心不可用时支持手动输入 IP
- **Windows 11 Mica 效果**：原生窗口背景融合

## 环境要求

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | ≥ 1.25 | 编译后端 |
| Node.js | - | 前端依赖（本项目为原生 HTML，无实际构建） |
| Wails CLI | v2 | 开发和构建工具 |
| Redis | - | 可选，用于设备中心注册 |
| UPX | - | 可选，用于压缩最终产物 |

## 安装 Wails CLI

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

确保 `$env:GOPATH\bin` 在系统 PATH 中。

## 快速开始

### 启动热加载开发模式

```powershell
.\build.ps1 dev
```

或直接：

```powershell
wails dev
```

启动后应用窗口会自动弹出，修改前端或 Go 代码后自动重新编译和刷新。

### 开发构建

```powershell
.\build.ps1 build
```

产物位于 `build\bin\`，会自动执行 UPX 压缩并复制 UltraVNC 依赖文件。

### 生产构建

```powershell
.\build.ps1 release
```

## 项目结构

```
winvncgo/
├── main.go                 # 入口，嵌入 frontend 资源
├── tasks/
│   ├── main.go             # Wails 应用配置与启动
│   ├── app.go              # 绑定到前端的 App 结构体
│   ├── manager.go          # 核心业务逻辑：ID 生成、VNC 启停、Redis 注册
│   ├── vnc.go              # UltraVNC 进程管理
│   ├── redis.go            # 原生 TCP 实现的 Redis 客户端
│   ├── config.go           # 配置文件加载（winvncgo.json）
│   ├── network.go          # 本机 IP 获取
│   ├── id.go               # 6 位随机 ID 生成
│   ├── types.go            # 状态事件结构体
│   ├── dwm_windows.go      # Windows 11 Mica 背景效果
│   └── dwm_other.go        # 非 Windows 平台占位
├── frontend/               # 前端（原生 HTML + JS + CSS）
│   ├── index.html
│   ├── app.js
│   ├── style.css
│   └── wailsjs/            # Wails 自动生成的 JS 绑定
├── libs/
│   ├── ultravnc/           # UltraVNC 二进制及 DLL
│   └── winvncgo.json        # 默认配置文件
├── build/                  # 构建产物
├── build.ps1               # 构建脚本（dev / build / release）
├── wails.json              # Wails 项目配置
├── go.mod
└── go.sum
```

## 配置文件

`winvncgo.json` 放在可执行文件同目录下，示例：

```json
{
  "Redis": {
    "Enabled": true,
    "Host": "127.0.0.1",
    "Port": 6379,
    "Password": "",
    "User": "",
    "KeyPrefix": "vnc:",
    "TTL": 30
  }
}
```

- `Enabled`：是否启用 Redis 控制中心
- `TTL`：设备注册信息过期时间（分钟）
- `KeyPrefix`：Redis key 前缀，用于隔离不同项目

未找到配置文件时使用默认配置（Redis `127.0.0.1:6379`，无密码）。

## 工作流程

1. 启动时生成 6 位随机 ID，写入 UltraVNC 配置作为密码
2. 启动 UltraVNC 服务（隐藏窗口，监听 5900 端口）
3. 若 Redis 可用，将 `ID → IP:5900` 注册到 Redis（带 TTL）
4. 定时轮询状态（VNC 运行状态、Redis 连接状态、是否有活跃连接）
5. 要控制其他设备时：输入对方 ID → 从 Redis 查询对方 IP → 调用 VNC Viewer 发起连接
6. 刷新 ID 时：若有活跃连接则跳过，否则停止 VNC → 删除旧 Redis 记录 → 生成新 ID → 重新启动

## 开发说明

- 前端为原生 HTML/JS/CSS，无构建步骤
- Wails 自动生成 `frontend/wailsjs/` 下的 JS 绑定代码
- Redis 客户端为原生 TCP 实现（未使用第三方库）
- 日志输出到 `winvncgo.log`（与可执行文件同目录）
