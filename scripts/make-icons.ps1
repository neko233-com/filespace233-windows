# Regenerates resources/icon.png, resources/tray.png and resources/app.ico
# from resources/icon-src.png. Requires Windows PowerShell 5.1+ (.NET drawing).

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$root = Split-Path -Parent $PSScriptRoot
$srcPath = Join-Path $root 'resources\icon-src.png'
$iconPath = Join-Path $root 'resources\icon.png'
$trayPath = Join-Path $root 'resources\tray.png'
$icoPath = Join-Path $root 'resources\app.ico'

if (-not (Test-Path -LiteralPath $srcPath)) {
    throw "missing source icon: $srcPath"
}

function Resize-Png([System.Drawing.Image]$source, [int]$size, [string]$path, [System.Drawing.Rectangle]$srcRect) {
    $bmp = New-Object System.Drawing.Bitmap $size, $size, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.Clear([System.Drawing.Color]::Transparent)
    $g.CompositingMode = [System.Drawing.Drawing2D.CompositingMode]::SourceCopy
    $g.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
    $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
    $g.DrawImage($source, (New-Object System.Drawing.Rectangle 0, 0, $size, $size), $srcRect, [System.Drawing.GraphicsUnit]::Pixel)
    $g.Dispose()
    $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    $bmp.Dispose()
}

function Png-Bytes([System.Drawing.Image]$source, [int]$size, [System.Drawing.Rectangle]$srcRect) {
    $bmp = New-Object System.Drawing.Bitmap $size, $size, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.Clear([System.Drawing.Color]::Transparent)
    $g.CompositingMode = [System.Drawing.Drawing2D.CompositingMode]::SourceCopy
    $g.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
    $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
    $g.DrawImage($source, (New-Object System.Drawing.Rectangle 0, 0, $size, $size), $srcRect, [System.Drawing.GraphicsUnit]::Pixel)
    $g.Dispose()
    $ms = New-Object System.IO.MemoryStream
    $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
    $bmp.Dispose()
    $bytes = $ms.ToArray()
    $ms.Dispose()
    return ,$bytes
}

function Get-AlphaCrop([System.Drawing.Bitmap]$bmp) {
    $minX = $bmp.Width
    $minY = $bmp.Height
    $maxX = -1
    $maxY = -1
    for ($y = 0; $y -lt $bmp.Height; $y += 2) {
        for ($x = 0; $x -lt $bmp.Width; $x += 2) {
            if ($bmp.GetPixel($x, $y).A -gt 8) {
                if ($x -lt $minX) { $minX = $x }
                if ($x -gt $maxX) { $maxX = $x }
                if ($y -lt $minY) { $minY = $y }
                if ($y -gt $maxY) { $maxY = $y }
            }
        }
    }
    if ($maxX -lt 0) {
        return New-Object System.Drawing.Rectangle 0, 0, $bmp.Width, $bmp.Height
    }
    $side = [Math]::Max($maxX - $minX + 1, $maxY - $minY + 1)
    $side = [int]($side * 1.06)
    $cx = [int](($minX + $maxX) / 2)
    $cy = [int](($minY + $maxY) / 2)
    $left = [Math]::Max(0, $cx - [int]($side / 2))
    $top = [Math]::Max(0, $cy - [int]($side / 2))
    $side = [Math]::Min($side, [Math]::Min($bmp.Width - $left, $bmp.Height - $top))
    return New-Object System.Drawing.Rectangle $left, $top, $side, $side
}

$src = [System.Drawing.Image]::FromFile($srcPath)
$crop = Get-AlphaCrop $src
Resize-Png $src 1024 $iconPath $crop
Resize-Png $src 32 $trayPath $crop

$sizes = @(16, 24, 32, 48, 64, 128, 256)
$images = New-Object 'System.Collections.Generic.List[byte[]]'
foreach ($size in $sizes) {
    $images.Add([byte[]](Png-Bytes $src $size $crop))
}
$src.Dispose()

$fs = [System.IO.File]::Create($icoPath)
$bw = New-Object System.IO.BinaryWriter $fs
$bw.Write([UInt16]0)
$bw.Write([UInt16]1)
$bw.Write([UInt16]$sizes.Count)
$offset = 6 + 16 * $sizes.Count
for ($i = 0; $i -lt $sizes.Count; $i++) {
    $size = $sizes[$i]
    $dim = if ($size -ge 256) { 0 } else { $size }
    $bw.Write([Byte]$dim)
    $bw.Write([Byte]$dim)
    $bw.Write([Byte]0)
    $bw.Write([Byte]0)
    $bw.Write([UInt16]1)
    $bw.Write([UInt16]32)
    $bw.Write([UInt32]$images[$i].Length)
    $bw.Write([UInt32]$offset)
    $offset += $images[$i].Length
}
foreach ($img in $images) {
    $bw.Write($img)
}
$bw.Dispose()
$fs.Dispose()

Write-Output "icon.png, tray.png, app.ico regenerated"
