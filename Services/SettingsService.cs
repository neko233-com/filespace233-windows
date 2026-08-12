using Windows.Storage;

namespace Filespace233.Services;

public enum SearchProvider
{
    LocalIndex,
    Everything
}

public sealed class SettingsService
{
    private const string WinFEnabledKey = "WinFEnabled";
    private const string SearchProviderKey = "SearchProvider";
    private const string SearchRootsKey = "SearchRoots";
    private const string StartupConfiguredKey = "StartupConfigured";
    private const string StartupEnabledKey = "StartupEnabled";
    private readonly ApplicationDataContainer _settings = ApplicationData.Current.LocalSettings;

    public bool WinFEnabled
    {
        get => ReadBool(WinFEnabledKey, true);
        set => _settings.Values[WinFEnabledKey] = value;
    }

    public bool StartupConfigured => ReadBool(StartupConfiguredKey, false);

    public bool StartupEnabled
    {
        get => ReadBool(StartupEnabledKey, true);
        set
        {
            _settings.Values[StartupEnabledKey] = value;
            _settings.Values[StartupConfiguredKey] = true;
        }
    }

    public SearchProvider SearchProvider
    {
        get => Enum.TryParse<SearchProvider>(ReadString(SearchProviderKey, nameof(Services.SearchProvider.LocalIndex)), out var value)
            ? value
            : Services.SearchProvider.LocalIndex;
        set => _settings.Values[SearchProviderKey] = value.ToString();
    }

    public IReadOnlyList<string> SearchRoots
    {
        get
        {
            var stored = ReadString(SearchRootsKey, string.Empty);
            if (!string.IsNullOrWhiteSpace(stored))
            {
                return stored.Split('|', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries)
                    .Where(Directory.Exists)
                    .Distinct(StringComparer.OrdinalIgnoreCase)
                    .ToArray();
            }

            return DefaultSearchRoots();
        }
    }

    public void SetSearchRoots(IEnumerable<string> roots)
    {
        _settings.Values[SearchRootsKey] = string.Join('|', roots.Where(Directory.Exists).Distinct(StringComparer.OrdinalIgnoreCase));
    }

    private bool ReadBool(string key, bool fallback)
    {
        return _settings.Values.TryGetValue(key, out var value) && value is bool boolean ? boolean : fallback;
    }

    private string ReadString(string key, string fallback)
    {
        return _settings.Values.TryGetValue(key, out var value) && value is string text ? text : fallback;
    }

    private static IReadOnlyList<string> DefaultSearchRoots()
    {
        var roots = new[]
        {
            Environment.GetFolderPath(Environment.SpecialFolder.UserProfile),
            Environment.GetFolderPath(Environment.SpecialFolder.DesktopDirectory),
            Environment.GetFolderPath(Environment.SpecialFolder.MyDocuments),
            Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.UserProfile), "Downloads")
        };

        return roots.Where(Directory.Exists).Distinct(StringComparer.OrdinalIgnoreCase).ToArray();
    }
}
