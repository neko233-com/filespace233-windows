namespace Filespace233.Models;

public sealed class FileItem
{
    public string Name { get; init; } = string.Empty;
    public string FullPath { get; init; } = string.Empty;
    public bool IsDirectory { get; init; }
    public long Size { get; init; }
    public DateTime ModifiedUtc { get; init; }
    public string TypeBadge => IsDirectory ? "DIR" : "FILE";
    public string SizeDisplay => IsDirectory ? "" : FormatSize(Size);
    public string ModifiedDisplay => ModifiedUtc == default ? "" : ModifiedUtc.ToLocalTime().ToString("yyyy-MM-dd HH:mm");
    public string Extension => IsDirectory ? "" : Path.GetExtension(Name).ToLowerInvariant();

    private static string FormatSize(long bytes)
    {
        if (bytes < 1024) return $"{bytes} B";
        if (bytes < 1024 * 1024) return $"{bytes / 1024d:0.0} KB";
        if (bytes < 1024L * 1024 * 1024) return $"{bytes / (1024d * 1024):0.0} MB";
        return $"{bytes / (1024d * 1024 * 1024):0.0} GB";
    }
}
