# 下载并解压 WebView2 109.x 固定版本运行时（最后一个支持 Windows 7 的版本）
# 用法: .\scripts\fetch-webview2-109.ps1 [-Url <直链>] [-OutDir <目录>]
# 默认输出到 build\webview2-runtime\，之后运行 .\build.ps1 win7 会把它打进发行包。
param(
    [string]$Url = "",
    [string]$OutDir = ""
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
if (-not $OutDir) { $OutDir = Join-Path $root "build\webview2-runtime" }

# WebView2 109.0.1518.78 / 109.0.1518.140 是最后支持 Win7 的版本。
# 微软官方“Fixed Version”下载页只保留较新版本，109 需要从下列来源获取；
# 均可被 -Url 参数覆盖。
$defaultUrls = @(
    "https://msedge.sf.dl.delivery.mp.microsoft.com/filestreamingservice/files/Microsoft.WebView2.FixedVersionRuntime.109.0.1518.78.x64.cab"
    "https://msedge.sf.dl.delivery.mp.microsoft.com/filestreamingservice/files/Microsoft.WebView2.FixedVersionRuntime.109.0.1518.140.x64.cab"
)

if ($Url) {
    $defaultUrls = @($Url) + $defaultUrls
}

$workDir = Join-Path ([System.IO.Path]::GetTempPath()) "wv2-109-fetch"
if (Test-Path $workDir) { Remove-Item $workDir -Recurse -Force }
New-Item -ItemType Directory -Path $workDir | Out-Null

$downloaded = $null
foreach ($u in $defaultUrls) {
    Write-Host "尝试下载: $u" -ForegroundColor Cyan
    $ext = if ($u -match '\.zip$') { ".zip" } else { ".cab" }
    $target = Join-Path $workDir ("webview2-109" + $ext)
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $u -OutFile $target -UseBasicParsing -TimeoutSec 600
        if ((Get-Item $target).Length -gt 20MB) {
            $downloaded = $target
            break
        }
        Write-Host "  文件过小，判定为无效，继续下一个来源" -ForegroundColor Yellow
    } catch {
        Write-Host "  失败: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

if (-not $downloaded) {
    Write-Host ""
    Write-Host "所有下载来源均失败。" -ForegroundColor Red
    Write-Host "请手动获取 Microsoft.WebView2.FixedVersionRuntime.109.0.1518.78.x64.cab（或 .zip）后：" -ForegroundColor Yellow
    Write-Host "  .\scripts\fetch-webview2-109.ps1 -Url <本地文件路径或直链>" -ForegroundColor Yellow
    Write-Host "或手动解压到 build\webview2-runtime\，使其中包含 msedgewebview2.exe 和 EBWebView\。" -ForegroundColor Yellow
    exit 1
}

Write-Host "解压到 $OutDir ..." -ForegroundColor Cyan
if (Test-Path $OutDir) { Remove-Item $OutDir -Recurse -Force }
New-Item -ItemType Directory -Path $OutDir | Out-Null

if ($downloaded -like "*.cab") {
    # cab 解压: expand.exe -F:* 展开所有文件（含子目录）
    & expand.exe $downloaded -F:* $OutDir | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "expand.exe 解压失败" }
} else {
    Expand-Archive -Path $downloaded -DestinationPath $OutDir -Force
}

# 解压结果可能多包一层目录，统一归位到 $OutDir 根
if (-not (Test-Path (Join-Path $OutDir "msedgewebview2.exe"))) {
    $inner = Get-ChildItem $OutDir -Directory | Select-Object -First 1
    if ($inner -and (Test-Path (Join-Path $inner.FullName "msedgewebview2.exe"))) {
        Get-ChildItem $inner.FullName | Move-Item -Destination $OutDir -Force
        Remove-Item $inner.FullName -Recurse -Force
    }
}

if (-not (Test-Path (Join-Path $OutDir "msedgewebview2.exe"))) {
    Write-Host "解压后未找到 msedgewebview2.exe，目录结构不符合预期。" -ForegroundColor Red
    exit 1
}
if (-not (Test-Path (Join-Path $OutDir "EBWebView\x64\EmbeddedBrowserWebView.dll"))) {
    Write-Host "警告: 未找到 EBWebView\x64\EmbeddedBrowserWebView.dll，运行时可能不完整。" -ForegroundColor Yellow
}

$size = [math]::Round((Get-ChildItem $OutDir -Recurse -File | Measure-Object -Property Length -Sum).Sum / 1MB, 2)
Write-Host "完成: $OutDir ($size MB)" -ForegroundColor Green
Write-Host "现在运行 .\build.ps1 win7 即可把运行时打进发行包。" -ForegroundColor Green
