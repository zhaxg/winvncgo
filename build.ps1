# build.ps1 — 用 Win7 兼容工具链构建（windows/amd64）
#
# mygo native UI 无 WebView2 依赖，产物为单个 exe，
# -H windowsgui 保证启动无控制台黑窗。
param(
    [string]$GoRoot = "D:\env\go-legacy-win7",
    [string]$Out    = "bin\winvncgo.exe"
)

Set-Location $PSScriptRoot

$go = Join-Path $GoRoot "bin\go.exe"
if (-not (Test-Path $go)) {
    Write-Error "找不到工具链: $go"
    exit 1
}
# 锁定本机工具链，禁止 go.mod 触发在线下载/切换
$env:GOTOOLCHAIN = "local"

try {
    # 图标资源：根目录 app.ico（生成的 rsrc.syso 只在链接期间需要）
    if (Test-Path "app.ico") {
        Write-Host "嵌入 exe 图标: app.ico"
        & $go run github.com/akavel/rsrc@latest -arch=amd64 -ico app.ico -o rsrc.syso
        if ($LASTEXITCODE -ne 0) { exit 1 }
    } else {
        Write-Warning "app.ico 不存在，产物不带图标"
    }

    $outDir = Split-Path $Out -Parent
    if ($outDir) { New-Item -ItemType Directory -Force -Path $outDir | Out-Null }

    Write-Host "构建 mygo 版 (windows/amd64, GUI 子系统)..."
    & $go build -ldflags "-H windowsgui" -o $Out .
    if ($LASTEXITCODE -ne 0) {
        Write-Error "构建失败"
        exit 1
    }

    $size = [math]::Round((Get-Item $Out).Length / 1MB, 2)
    Write-Host "构建成功: $Out ($size MB)" -ForegroundColor Green

    # UPX 压缩（可选依赖，与原 wails 构建一致：--best --lzma）
    if (Get-Command upx -ErrorAction SilentlyContinue) {
        $before = (Get-Item $Out).Length
        Write-Host "UPX 压缩..."
        & upx --best --lzma $Out
        if ($LASTEXITCODE -eq 0) {
            $after = (Get-Item $Out).Length
            $pct = [math]::Round((1 - $after / [double]$before) * 100, 1)
            Write-Host ("UPX 完成: {0:N2} MB -> {1:N2} MB（缩小 {2}%）" -f ($before / 1MB), ($after / 1MB), $pct) -ForegroundColor Green
        } else {
            Write-Warning "UPX 压缩失败，保留未压缩产物继续使用"
        }
    } else {
        Write-Host "未检测到 upx，跳过压缩（可选步骤）"
    }
} finally {
    # 中间文件构建后即清理（图标已在链接时嵌入 exe）
    Remove-Item rsrc.syso -Force -ErrorAction SilentlyContinue
}
