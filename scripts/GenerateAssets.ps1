Add-Type -AssemblyName System.Drawing

$assetDirectory = Join-Path $PSScriptRoot '..\Assets'
New-Item -ItemType Directory -Force -Path $assetDirectory | Out-Null

function New-FilespaceIcon {
    param([int]$Size, [string]$Path)

    $bitmap = [System.Drawing.Bitmap]::new($Size, $Size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
    $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $graphics.Clear([System.Drawing.Color]::FromArgb(255, 16, 19, 23))

    $scale = $Size / 512.0
    $folder = [System.Drawing.Drawing2D.GraphicsPath]::new()
    $folder.AddLine(112 * $scale, 148 * $scale, 112 * $scale, 148 * $scale)
    $folder.AddArc(112 * $scale, 108 * $scale, 80 * $scale, 80 * $scale, 180, 90)
    $folder.AddLine(192 * $scale, 108 * $scale, 246 * $scale, 108 * $scale)
    $folder.AddLine(246 * $scale, 108 * $scale, 286 * $scale, 156 * $scale)
    $folder.AddLine(286 * $scale, 156 * $scale, 360 * $scale, 156 * $scale)
    $folder.AddArc(320 * $scale, 156 * $scale, 80 * $scale, 80 * $scale, 270, 90)
    $folder.AddLine(400 * $scale, 196 * $scale, 400 * $scale, 354 * $scale)
    $folder.AddArc(360 * $scale, 314 * $scale, 80 * $scale, 80 * $scale, 0, 90)
    $folder.AddLine(360 * $scale, 394 * $scale, 152 * $scale, 394 * $scale)
    $folder.AddArc(112 * $scale, 354 * $scale, 80 * $scale, 80 * $scale, 90, 90)
    $folder.CloseFigure()
    $graphics.FillPath([System.Drawing.Brushes]::DodgerBlue, $folder)

    $body = [System.Drawing.Drawing2D.GraphicsPath]::new()
    $body.AddRectangle([System.Drawing.RectangleF]::new(112 * $scale, 196 * $scale, 288 * $scale, 166 * $scale))
    $graphics.FillPath([System.Drawing.Brushes]::CornflowerBlue, $body)

    $pen = [System.Drawing.Pen]::new([System.Drawing.Color]::FromArgb(255, 232, 243, 255), [Math]::Max(2, 24 * $scale))
    $pen.StartCap = [System.Drawing.Drawing2D.LineCap]::Round
    $pen.EndCap = [System.Drawing.Drawing2D.LineCap]::Round
    $graphics.DrawLine($pen, 174 * $scale, 252 * $scale, 338 * $scale, 252 * $scale)
    $graphics.DrawLine($pen, 174 * $scale, 302 * $scale, 286 * $scale, 302 * $scale)

    $bitmap.Save($Path, [System.Drawing.Imaging.ImageFormat]::Png)
    $pen.Dispose(); $body.Dispose(); $folder.Dispose(); $graphics.Dispose(); $bitmap.Dispose()
}

New-FilespaceIcon 512 (Join-Path $assetDirectory 'FilespaceLogo.png')
New-FilespaceIcon 150 (Join-Path $assetDirectory 'Square150x150Logo.png')
New-FilespaceIcon 50 (Join-Path $assetDirectory 'StoreLogo.png')
New-FilespaceIcon 44 (Join-Path $assetDirectory 'Square44x44Logo.png')
