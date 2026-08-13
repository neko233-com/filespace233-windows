namespace Filespace233.Models;

public sealed class UpdateManifest
{
    public int SchemaVersion { get; init; }
    public string Version { get; init; } = string.Empty;
    public string Tag { get; init; } = string.Empty;
    public string ReleasePage { get; init; } = string.Empty;
    public IReadOnlyList<UpdateAsset> Assets { get; init; } = Array.Empty<UpdateAsset>();
}

public sealed class UpdateAsset
{
    public string Runtime { get; init; } = string.Empty;
    public string File { get; init; } = string.Empty;
    public string Sha256 { get; init; } = string.Empty;
    public long Size { get; init; }
}
