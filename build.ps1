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
        Write-Host "=== Windows 7 发行构建 ===" -ForegroundColor Yellow

        wails build

        if ($LASTEXITCODE -ne 0) {
            Write-Host "构建失败" -ForegroundColor Red
            exit 1
        }

        # 组装 Windows 7 发行包到 dist\winvncgo-win7\
        $dist = Join-Path $PSScriptRoot "dist\winvncgo-win7"
        if (Test-Path $dist) {
            Remove-Item $dist -Recurse -Force
        }
        New-Item -ItemType Directory -Path $dist | Out-Null

        Copy-Item build\bin\winvncgo.exe $dist
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
