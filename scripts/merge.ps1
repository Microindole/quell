# scripts/merge.ps1
param(
    [string]$TargetDir,
    [string]$OutputName,
    [string]$FFmpegPath,
    [string]$CoverUrl,      # 接收 Go 传来的封面链接
    [string]$LocalCoverPath # [新增] 接收 Go 传来的本地封面路径
)

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# ================= FUNCTION: 自适应计算偏移量 =================
function Get-BiliOffset([string]$filePath) {
    # 读取前 128 字节 (足够找到头部了)
    if (-not (Test-Path $filePath)) { return 0 }
    
    $fs = [System.IO.File]::OpenRead($filePath)
    $buffer = New-Object byte[] 128
    $count = $fs.Read($buffer, 0, 128)
    $fs.Close()

    # 寻找 'ftyp' (Hex: 66 74 79 70)
    for ($i = 0; $i -lt ($count - 3); $i++) {
        if ($buffer[$i]   -eq 0x66 -and 
            $buffer[$i+1] -eq 0x74 -and 
            $buffer[$i+2] -eq 0x79 -and 
            $buffer[$i+3] -eq 0x70) {
            
            $offset = $i - 4
            if ($offset -lt 0) { return 0 }
            return $offset
        }
    }
    return 0
}
# ==============================================================

# --- 1. 环境检查 ---
$ffmpegCmd = "ffmpeg"
if (![string]::IsNullOrEmpty($FFmpegPath) -and (Test-Path $FFmpegPath)) { 
    $ffmpegCmd = $FFmpegPath 
}

try {
    & $ffmpegCmd -version | Out-Null
} catch {
    Write-Error "FFmpeg not found."
    exit 1
}

# --- 2. 寻找源文件 ---
Write-Host "Scanning: $TargetDir"
$allFiles = Get-ChildItem -Path $TargetDir -Recurse -Force
$m4sFiles = $allFiles | Where-Object { $_.Name -like "*.m4s" } | Sort-Object Length -Descending

if ($m4sFiles.Count -lt 2) {
    Write-Error "Not enough m4s files found. Files found: $($m4sFiles.Count)"
    exit 1
}

$vSource = $m4sFiles[0].FullName
$aSource = $m4sFiles[1].FullName

# --- 3. 计算偏移量 (自适应) ---
$detectedOffset = Get-BiliOffset $vSource
Write-Host "Detected Offset: $detectedOffset bytes" -ForegroundColor Cyan

# --- 4. 封面处理 (逻辑优化) ---
$localCoverTemp = Join-Path $TargetDir "cover_fixed.jpg"
$useCover = $false

# [新增] 优先级 1: 如果 JSON 里指明了本地封面路径，且文件存在，直接用
if (![string]::IsNullOrEmpty($LocalCoverPath) -and (Test-Path $LocalCoverPath)) {
    Write-Host "Using local cover from JSON info: $LocalCoverPath"
    Copy-Item $LocalCoverPath $localCoverTemp -Force
    $useCover = $true
}

# 优先级 2: 如果没有本地路径，但有 URL，尝试下载
if (-not $useCover -and ![string]::IsNullOrEmpty($CoverUrl)) {
    try {
        Write-Host "Downloading cover from: $CoverUrl"
        Invoke-WebRequest -Uri $CoverUrl -OutFile $localCoverTemp -UseBasicParsing
        $useCover = $true
    } catch {
        Write-Host "Download cover failed, trying other local files..." -ForegroundColor Yellow
    }
}

# 优先级 3: 还没找到，扫描目录下最大的图片
if (-not $useCover) {
    $potentialCovers = $allFiles | Where-Object { $_.Extension -match "\.(jpg|png)$" } | Sort-Object Length -Descending
    if ($potentialCovers.Count -gt 0) {
        Write-Host "Using found image: $($potentialCovers[0].Name)"
        Copy-Item $potentialCovers[0].FullName $localCoverTemp -Force
        $useCover = $true
    }
}

# --- 5. 修复与合并 ---
$vTemp = Join-Path $TargetDir "v_temp.mp4"
$aTemp = Join-Path $TargetDir "a_temp.mp4"
$finalOutput = Join-Path $TargetDir "$OutputName.mp4"

try {
    # 修复视频流
    $in = [System.IO.File]::OpenRead($vSource)
    $out = [System.IO.File]::Create($vTemp)
    if ($detectedOffset -gt 0) { $in.Seek($detectedOffset, [System.IO.SeekOrigin]::Begin) | Out-Null }
    $in.CopyTo($out)
    $in.Close(); $out.Close()

    # 修复音频流
    $in = [System.IO.File]::OpenRead($aSource)
    $out = [System.IO.File]::Create($aTemp)
    if ($detectedOffset -gt 0) { $in.Seek($detectedOffset, [System.IO.SeekOrigin]::Begin) | Out-Null }
    $in.CopyTo($out)
    $in.Close(); $out.Close()

    # 合并
    Write-Host "Merging..."
    
    if ($useCover) {
        $ffArgs = @(
            "-i", $vTemp, 
            "-i", $aTemp, 
            "-i", $localCoverTemp,
            "-map", "0", "-map", "1", "-map", "2",
            "-c:v", "copy", "-c:a", "copy", 
            "-c:v:1", "mjpeg", 
            "-disposition:v:1", "attached_pic",
            "-metadata:s:v:1", "title=Cover", 
            "-metadata:s:v:1", "comment=Cover",
            "-y", $finalOutput, 
            "-loglevel", "error"
        )
    } else {
        $ffArgs = @("-i", $vTemp, "-i", $aTemp, "-c", "copy", "-y", $finalOutput, "-loglevel", "error")
    }
    
    & $ffmpegCmd $ffArgs

    # 成功后如果生成了独立的 jpg，清理或保留(这里保留一张作为封面文件)
    if ($useCover -and (Test-Path $localCoverTemp)) {
        Copy-Item $localCoverTemp (Join-Path $TargetDir "$OutputName.jpg") -Force
    }

    Write-Host "SUCCESS"

} catch {
    Write-Error "Processing Failed: $_"
    exit 1
} finally {
    if (Test-Path $vTemp) { Remove-Item $vTemp -ErrorAction SilentlyContinue }
    if (Test-Path $aTemp) { Remove-Item $aTemp -ErrorAction SilentlyContinue }
    if (Test-Path $localCoverTemp) { Remove-Item $localCoverTemp -ErrorAction SilentlyContinue }
}