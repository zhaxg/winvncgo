param(
    [ValidateSet("build", "release", "dev", "win7")]
    [string]$Action = "build"
)

Set-Location $PSScriptRoot

# 确保 Wails CLI 在 PATH 中
$wailsPath = Join-Path $env:GOPATH "bin"
if (-not $env:PATH.Contains($wailsPath)) {
    $env:PATH = "$wailsPath;$env:PATH"
}

switch ($Action) {
    "dev" {
        Write-Host "=== 开发模式 ===" -ForegroundColor Yellow
        wails dev
    }
    "build" {
        Write-Host "=== 开发构建 ===" -ForegroundColor Yellow

        wails build

        if ($LASTEXITCODE -ne 0) {
            Write-Host "构建失败" -ForegroundColor Red
            exit 1
        }

        # UPX 压缩
        if (Get-Command upx -ErrorAction SilentlyContinue) {
            Write-Host "UPX 压缩中..."
            upx --best --lzma build\bin\winvncgo.exe
        }

        # UltraVNC 运行时：没有 winvnc.exe 才整目录复制
        if (-not (Test-Path build\bin\winvnc.exe)) {
            Copy-Item libs\ultravnc\* build\bin\ -Force
        }
        # 配置文件：没有才复制，有就跳过（保留用户修改）
        if (-not (Test-Path build\bin\winvncgo.json)) {
            Copy-Item libs\winvncgo.json build\bin\
        }
        if (-not (Test-Path build\bin\ultravnc.ini)) {
            Copy-Item libs\ultravnc.ini build\bin\
        }

        $size = [math]::Round((Get-Item "build\bin\winvncgo.exe").Length / 1MB, 2)
        Write-Host "构建成功: build\bin\ ($size MB)" -ForegroundColor Green
    }
    "release" {
        Write-Host "=== 生产构建 ===" -ForegroundColor Yellow

        wails build -clean

        if ($LASTEXITCODE -ne 0) {
            Write-Host "编译失败" -ForegroundColor Red
            exit 1
        }

        # UPX 压缩
        if (Get-Command upx -ErrorAction SilentlyContinue) {
            Write-Host "UPX 压缩中..."
            upx --best --lzma build\bin\winvncgo.exe
        }

        # UltraVNC 运行时：没有 winvnc.exe 才整目录复制
        if (-not (Test-Path build\bin\winvnc.exe)) {
            Copy-Item libs\ultravnc\* build\bin\ -Force
        }
        # 配置文件：没有才复制，有就跳过（保留用户修改）
        if (-not (Test-Path build\bin\winvncgo.json)) {
            Copy-Item libs\winvncgo.json build\bin\
        }
        if (-not (Test-Path build\bin\ultravnc.ini)) {
            Copy-Item libs\ultravnc.ini build\bin\
        }

        $totalSize = [math]::Round((Get-ChildItem build\bin -File | Measure-Object -Property Length -Sum).Sum / 1MB, 2)
        Write-Host "`n=== 分发目录: build\bin\ ===" -ForegroundColor Yellow
        Write-Host "总计: $totalSize MB" -ForegroundColor Cyan
        Get-ChildItem build\bin -File | ForEach-Object {
            Write-Host "  $($_.Name) ($([math]::Round($_.Length / 1MB, 2)) MB)" -ForegroundColor Gray
        }
    }
    "win7" {
        Write-Host "=== Windows 7 发行构建（Go 1.20 + wails-win7） ===" -ForegroundColor Yellow

        # Win7 兼容构建必须用 Go 1.20（官方支持 Win7 的最后一个版本）编译。
        # 优先用 go1.20.14 SDK：go1.20.14 download 安装到 %USERPROFILE%\sdk\go1.20.14
        $sdk = Join-Path $env:USERPROFILE "sdk\go1.20.14\bin"
        if (Test-Path (Join-Path $sdk "go.exe")) {
            $env:PATH = "$sdk;$env:PATH"
        }
        $gov = (& go version) -join ""
        if ($gov -notmatch "go1\.20\.") {
            Write-Host "当前 Go 是 '$gov'，Win7 构建需要 go1.20.x。" -ForegroundColor Red
            Write-Host "请先执行:  go install golang.org/dl/go1.20.14@latest; go1.20.14 download" -ForegroundColor Yellow
            exit 1
        }
        Write-Host "使用工具链: $gov" -ForegroundColor Cyan

        # wails-win7 CLI（支持 Win7 的 Wails 分叉），没有则先安装
        $wailsWin7 = Join-Path $PSScriptRoot "build\tools\wails-win7\wails.exe"
        if (-not (Test-Path $wailsWin7)) {
            Write-Host "安装 wails-win7 CLI ..." -ForegroundColor Yellow
            $oldGobin = $env:GOBIN
            $env:GOBIN = Join-Path $PSScriptRoot "build\tools\wails-win7"
            go install github.com/cllsm/wails-win7/v2/cmd/wails@v2.0.0-20240914020515-2742ebadb0a7
            $env:GOBIN = $oldGobin
            if (-not (Test-Path $wailsWin7)) {
                Write-Host "wails-win7 CLI 安装失败" -ForegroundColor Red
                exit 1
            }
        }

        & $wailsWin7 build

        if ($LASTEXITCODE -ne 0) {
            Write-Host "构建失败" -ForegroundColor Red
            exit 1
        }

        # 诊断工具（控制台程序）：Win7 上启动失败时先跑它定位问题
        go build -o build\bin\win7diag.exe .\cmd\win7diag
        if ($LASTEXITCODE -ne 0) {
            Write-Host "win7diag 构建失败" -ForegroundColor Red
            exit 1
        }
        # 控制台版主程序：运行时 fatal error 会直接打印在黑窗口上
        go build -tags "desktop,production" -o build\bin\winvncgo-console.exe .
        # 32 位版诊断工具：万一目标是 32 位 Win7，用它确认系统架构
        $oldArch = $env:GOARCH
        $env:GOARCH = "386"
        go build -o build\bin\win7diag-x86.exe .\cmd\win7diag
        $env:GOARCH = $oldArch

        # 组装 Windows 7 发行包到 dist\winvncgo-win7\
        $dist = Join-Path $PSScriptRoot "dist\winvncgo-win7"
        if (Test-Path $dist) {
            Remove-Item $dist -Recurse -Force
        }
        New-Item -ItemType Directory -Path $dist | Out-Null

        Copy-Item build\bin\winvncgo.exe $dist
        Copy-Item build\bin\winvncgo-console.exe $dist -ErrorAction SilentlyContinue
        Copy-Item build\bin\win7diag.exe $dist -ErrorAction SilentlyContinue
        Copy-Item build\bin\win7diag-x86.exe $dist -ErrorAction SilentlyContinue
        Copy-Item scripts\check-win7.bat $dist -ErrorAction SilentlyContinue
        Copy-Item libs\ultravnc\* $dist -Force
        Copy-Item libs\winvncgo.json $dist
        Copy-Item libs\ultravnc.ini $dist
        Copy-Item README-win7.md (Join-Path $dist "README-win7.md") -ErrorAction SilentlyContinue

        # 固定版本 WebView2 运行时（可选）：scripts\fetch-webview2-109.ps1 下载后解压到 build\webview2-runtime\
        $wv2 = Join-Path $PSScriptRoot "build\webview2-runtime"
        if (Test-Path (Join-Path $wv2 "msedgewebview2.exe")) {
            Copy-Item $wv2 (Join-Path $dist "webview2") -Recurse
            Write-Host "已包含固定版本 WebView2 运行时" -ForegroundColor Green
        } else {
            Write-Host "未找到 build\webview2-runtime\，产物不含 WebView2 运行时。" -ForegroundColor Yellow
            Write-Host "请在目标机器安装 WebView2 109.x，或运行 scripts\fetch-webview2-109.ps1 后重新打包。" -ForegroundColor Yellow
        }

        $totalSize = [math]::Round((Get-ChildItem $dist -File | Measure-Object -Property Length -Sum).Sum / 1MB, 2)
        Write-Host "`n=== Windows 7 分发目录: $dist ===" -ForegroundColor Yellow
        Write-Host "总计: $totalSize MB" -ForegroundColor Cyan
        Get-ChildItem $dist -File | ForEach-Object {
            Write-Host "  $($_.Name) ($([math]::Round($_.Length / 1MB, 2)) MB)" -ForegroundColor Gray
        }
    }
}
