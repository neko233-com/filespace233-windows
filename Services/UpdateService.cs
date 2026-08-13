using System.Net.Http.Headers;
using System.Diagnostics;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Security.Cryptography;
using System.Text.Json;
using Filespace233.Models;

namespace Filespace233.Services;

public sealed class UpdateService
{
    public const string DefaultManifestUrl = "https://github.com/neko233-com/filespace233-windows/releases/latest/download/latest.json";
    public const string DefaultReleasePage = "https://github.com/neko233-com/filespace233-windows/releases/latest";
    private static readonly HttpClient HttpClient = CreateHttpClient();
    private static readonly JsonSerializerOptions JsonOptions = new() { PropertyNameCaseInsensitive = true };

    public string CurrentVersion => NormalizeVersion(Assembly.GetEntryAssembly()?.GetName().Version?.ToString() ?? "0.1.0");

    public async Task<UpdateCheckResult?> CheckAsync(SettingsService settings, CancellationToken cancellationToken = default)
    {
        var candidates = BuildManifestCandidates(settings);
        foreach (var candidate in candidates)
        {
            try
            {
                using var response = await HttpClient.GetAsync(candidate, HttpCompletionOption.ResponseHeadersRead, cancellationToken).ConfigureAwait(false);
                if (!response.IsSuccessStatusCode) continue;
                await using var stream = await response.Content.ReadAsStreamAsync(cancellationToken).ConfigureAwait(false);
                var manifest = await JsonSerializer.DeserializeAsync<UpdateManifest>(stream, JsonOptions, cancellationToken).ConfigureAwait(false);
                if (!IsValid(manifest) || !IsNewer(manifest!.Version, CurrentVersion)) continue;

                var runtime = RuntimeInformation.ProcessArchitecture == Architecture.Arm64 ? "win-arm64" : "win-x64";
                var asset = manifest.Assets.FirstOrDefault(item => string.Equals(item.Runtime, runtime, StringComparison.OrdinalIgnoreCase));
                if (asset is null || string.IsNullOrWhiteSpace(asset.File) || string.IsNullOrWhiteSpace(asset.Sha256)) continue;
                return new UpdateCheckResult { Manifest = manifest, Asset = asset, ManifestUri = candidate };
            }
            catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested) { throw; }
            catch (HttpRequestException) { }
            catch (JsonException) { }
        }

        return null;
    }

    public async Task<string> DownloadAndInstallAsync(UpdateCheckResult update, CancellationToken cancellationToken = default)
    {
        var fileName = Path.GetFileName(update.Asset.File);
        if (string.IsNullOrWhiteSpace(fileName) || !fileName.EndsWith(".msi", StringComparison.OrdinalIgnoreCase))
            throw new InvalidOperationException("The update asset is not an MSI file.");

        var downloadPath = await DownloadAsync(update, cancellationToken).ConfigureAwait(false);
        InstallMsi(downloadPath);
        return downloadPath;
    }

    public async Task<string> DownloadAsync(UpdateCheckResult update, CancellationToken cancellationToken = default)
    {
        var fileName = Path.GetFileName(update.Asset.File);
        if (string.IsNullOrWhiteSpace(fileName) || !fileName.EndsWith(".msi", StringComparison.OrdinalIgnoreCase))
            throw new InvalidOperationException("The update asset is not an MSI file.");

        var downloadPath = Path.Combine(Path.GetTempPath(), "Filespace", fileName);
        Directory.CreateDirectory(Path.GetDirectoryName(downloadPath)!);
        var temporaryPath = downloadPath + ".download";
        try
        {
            using var response = await HttpClient.GetAsync(update.DownloadUri, HttpCompletionOption.ResponseHeadersRead, cancellationToken).ConfigureAwait(false);
            response.EnsureSuccessStatusCode();
            await using (var source = await response.Content.ReadAsStreamAsync(cancellationToken).ConfigureAwait(false))
            await using (var destination = File.Create(temporaryPath))
            {
                await source.CopyToAsync(destination, cancellationToken).ConfigureAwait(false);
            }

            var fileInfo = new FileInfo(temporaryPath);
            if (update.Asset.Size > 0 && fileInfo.Length != update.Asset.Size)
                throw new InvalidDataException("The downloaded update size does not match the release manifest.");

            await using var file = File.OpenRead(temporaryPath);
            var hash = Convert.ToHexString(await SHA256.HashDataAsync(file, cancellationToken)).ToLowerInvariant();
            if (!string.Equals(hash, update.Asset.Sha256, StringComparison.OrdinalIgnoreCase))
                throw new InvalidDataException("The downloaded update checksum does not match the release manifest.");

            File.Move(temporaryPath, downloadPath, overwrite: true);
            return downloadPath;
        }
        finally
        {
            if (File.Exists(temporaryPath)) File.Delete(temporaryPath);
        }
    }

    public static void InstallMsi(string path)
    {
        if (!File.Exists(path) || !path.EndsWith(".msi", StringComparison.OrdinalIgnoreCase))
            throw new FileNotFoundException("The MSI update file was not found.", path);

        Process.Start(new ProcessStartInfo
        {
            FileName = "msiexec.exe",
            Arguments = $"/i \"{path}\" /passive /norestart",
            UseShellExecute = true
        });
    }

    private static IReadOnlyList<Uri> BuildManifestCandidates(SettingsService settings)
    {
        var candidates = new List<Uri>();
        if (!string.IsNullOrWhiteSpace(settings.UpdateMirrorPrefix))
        {
            var prefix = settings.UpdateMirrorPrefix.Trim();
            var manifestUrl = prefix.Contains("{file}", StringComparison.OrdinalIgnoreCase)
                ? prefix.Replace("{file}", "latest.json", StringComparison.OrdinalIgnoreCase)
                : prefix.EndsWith(".json", StringComparison.OrdinalIgnoreCase) ? prefix : prefix.TrimEnd('/') + "/latest.json";
            AddUri(candidates, manifestUrl);
        }
        AddUri(candidates, settings.UpdateManifestUrl);
        AddUri(candidates, DefaultManifestUrl);
        return candidates;
    }

    private static void AddUri(List<Uri> candidates, string? value)
    {
        if (Uri.TryCreate(value, UriKind.Absolute, out var uri) && uri.Scheme.Equals("https", StringComparison.OrdinalIgnoreCase) && !candidates.Contains(uri))
            candidates.Add(uri);
    }

    private static bool IsValid(UpdateManifest? manifest)
    {
        return manifest is not null && manifest.SchemaVersion == 1 && Version.TryParse(NormalizeVersion(manifest.Version), out _) && manifest.Assets is { Count: > 0 };
    }

    private static bool IsNewer(string candidate, string current)
    {
        return Version.TryParse(NormalizeVersion(candidate), out var candidateVersion)
            && Version.TryParse(NormalizeVersion(current), out var currentVersion)
            && candidateVersion > currentVersion;
    }

    private static string NormalizeVersion(string version)
    {
        var normalized = version.Trim().TrimStart('v', 'V');
        var separator = normalized.IndexOf('-');
        return separator >= 0 ? normalized[..separator] : normalized;
    }

    private static HttpClient CreateHttpClient()
    {
        var client = new HttpClient { Timeout = TimeSpan.FromSeconds(20) };
        client.DefaultRequestHeaders.UserAgent.Add(new ProductInfoHeaderValue("FilespaceUpdater", "0.1"));
        return client;
    }
}
