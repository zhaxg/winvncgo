param(
    [ValidateSet("build", "release", "dev")]
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
}
