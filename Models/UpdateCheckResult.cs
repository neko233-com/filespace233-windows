namespace Filespace233.Models;

public sealed class UpdateCheckResult
{
    public required UpdateManifest Manifest { get; init; }
    public required UpdateAsset Asset { get; init; }
    public required Uri ManifestUri { get; init; }
    public string DownloadUri => new Uri(ManifestUri, Asset.File).ToString();
}
