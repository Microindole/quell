# scripts/merge.ps1
param(
    [string]$TargetDir,
    [string]$OutputName,
    [string]$FFmpegPath,
    [string]$CoverUrl      # 接收 Go 传来的封面链接
)

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# ================= FUNCTION: 自适应计算偏移量 =================
function Get-BiliOffset([string]$filePath) {
    # 读取前 128 字节 (足够找到头部了)
    $fs = [System.IO.File]::OpenRead($filePath)
    $buffer = New-Object byte[] 128
    $count = $fs.Read($buffer, 0, 128)
    $fs.Close()

    # 寻找 'ftyp' (Hex: 66 74 79 70)
    # 标准 MP4 结构: [Size 4B] [ftyp 4B] ...
    # 所以 'ftyp' 的索引位置减去 4，就是文件真正的起始位置
    for ($i = 0; $i -lt ($count - 3); $i++) {
        if ($buffer[$i]   -eq 0x66 -and 
            $buffer[$i+1] -eq 0x74 -and 
            $buffer[$i+2] -eq 0x79 -and 
            $buffer[$i+3] -eq 0x70) {
            
            $offset = $i - 4
            if ($offset -lt 0) { return 0 } # 理论上不应该发生
            return $offset
        }
    }
    return 0 # 没找到 ftyp，假设不需要偏移
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
# 只检测视频文件即可，音频通常采用相同的混淆策略
$detectedOffset = Get-BiliOffset $vSource
Write-Host "Detected Offset: $detectedOffset bytes" -ForegroundColor Cyan

# --- 4. 封面处理 (下载正确的封面) ---
$localCover = Join-Path $TargetDir "cover_fixed.jpg"
$useCover = $false

# 优先下载 JSON 指定的 CoverUrl
if (![string]::IsNullOrEmpty($CoverUrl)) {
    try {
        Write-Host "Downloading cover from: $CoverUrl"
        # 使用 Curl 或者 Invoke-WebRequest 下载
        Invoke-WebRequest -Uri $CoverUrl -OutFile $localCover -UseBasicParsing
        $useCover = $true
    } catch {
        Write-Host "Download cover failed, trying local files..." -ForegroundColor Yellow
    }
}

# 如果下载失败，尝试找本地已有的图片
if (-not $useCover) {
    $potentialCovers = $allFiles | Where-Object { $_.Extension -match "\.(jpg|png)$" } | Sort-Object Length -Descending
    if ($potentialCovers.Count -gt 0) {
        Copy-Item $potentialCovers[0].FullName $localCover -Force
        $useCover = $true
        Write-Host "Using local cover: $($potentialCovers[0].Name)"
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

    # 修复音频流 (假设偏移量相同)
    $in = [System.IO.File]::OpenRead($aSource)
    $out = [System.IO.File]::Create($aTemp)
    if ($detectedOffset -gt 0) { $in.Seek($detectedOffset, [System.IO.SeekOrigin]::Begin) | Out-Null }
    $in.CopyTo($out)
    $in.Close(); $out.Close()

    # 合并 (如果有封面，尝试嵌入)
    Write-Host "Merging..."
    
    if ($useCover) {
        # 复杂命令：嵌入封面 (将图片作为视频流的一部分，并标记为附带图片)
        # -map 0 (视频) -map 1 (音频) -map 2 (图片)
        # -c:v copy (复制视频流) -c:a copy (复制音频流) 
        # -c:v:1 mjpeg (将图片流转为 mjpeg 兼容格式) 
        # -disposition:v:1 attached_pic (标记为封面)
        $ffArgs = @(
            "-i", $vTemp, 
            "-i", $aTemp, 
            "-i", $localCover,
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
        # 普通合并
        $ffArgs = @("-i", $vTemp, "-i", $aTemp, "-c", "copy", "-y", $finalOutput, "-loglevel", "error")
    }
    
    & $ffmpegCmd $ffArgs

    # 成功后把封面图也复制一份出来放在旁边，方便查看
    if ($useCover -and (Test-Path $localCover)) {
        Copy-Item $localCover (Join-Path $TargetDir "$OutputName.jpg") -Force
    }

    Write-Host "SUCCESS"

} catch {
    Write-Error "Processing Failed: $_"
    exit 1
} finally {
    # 清理
    if (Test-Path $vTemp) { Remove-Item $vTemp -ErrorAction SilentlyContinue }
    if (Test-Path $aTemp) { Remove-Item $aTemp -ErrorAction SilentlyContinue }
    if (Test-Path $localCover) { Remove-Item $localCover -ErrorAction SilentlyContinue }
}