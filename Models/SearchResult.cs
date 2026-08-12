namespace Filespace233.Models;

public sealed class SearchResult
{
    public string Name { get; init; } = string.Empty;
    public string FullPath { get; init; } = string.Empty;
    public bool IsDirectory { get; init; }
    public string TypeBadge => IsDirectory ? "DIR" : "FILE";
    public string ParentPath => Path.GetDirectoryName(FullPath) ?? string.Empty;
}
