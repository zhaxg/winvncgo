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

        # 只清理 bin 目录，保留 build\windows 资产（清单等会被 Wails 以默认内容重建）
        if (Test-Path build\bin) { Remove-Item build\bin -Recurse -Force }

        wails build

        if ($LASTEXITCODE -ne 0) {
            Write-Host "构建失败" -ForegroundColor Red
            exit 1
        }

        # UPX 压缩
        if (Get-Command upx -ErrorAction SilentlyContinue) {
            Write-Host "UPX 压缩中..."
            upx --best --lzma build\bin\wegovnc.exe
        }

        # 复制依赖
        Copy-Item libs\wegovnc.json build\bin\ -Force
        Copy-Item libs\ultravnc\* build\bin\ -Force

        $size = [math]::Round((Get-Item "build\bin\wegovnc.exe").Length / 1MB, 2)
        Write-Host "构建成功: build\bin\ ($size MB)" -ForegroundColor Green
    }
    "release" {
        Write-Host "=== 生产构建 ===" -ForegroundColor Yellow

        # 只清理 bin 目录，保留 build\windows 资产（清单等会被 Wails 以默认内容重建）
        if (Test-Path build\bin) { Remove-Item build\bin -Recurse -Force }

        wails build -clean

        if ($LASTEXITCODE -ne 0) {
            Write-Host "编译失败" -ForegroundColor Red
            exit 1
        }

        # UPX 压缩
        if (Get-Command upx -ErrorAction SilentlyContinue) {
            Write-Host "UPX 压缩中..."
            upx --best --lzma build\bin\wegovnc.exe
        }

        # 复制依赖
        Copy-Item libs\wegovnc.json build\bin\ -Force
        Copy-Item libs\ultravnc\* build\bin\ -Force

        $totalSize = [math]::Round((Get-ChildItem build\bin -File | Measure-Object -Property Length -Sum).Sum / 1MB, 2)
        Write-Host "`n=== 分发目录: build\bin\ ===" -ForegroundColor Yellow
        Write-Host "总计: $totalSize MB" -ForegroundColor Cyan
        Get-ChildItem build\bin -File | ForEach-Object {
            Write-Host "  $($_.Name) ($([math]::Round($_.Length / 1MB, 2)) MB)" -ForegroundColor Gray
        }
    }
}
