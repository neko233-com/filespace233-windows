using System.Text.Json;
using Windows.ApplicationModel;
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
    private const string AutoUpdateCheckKey = "AutoUpdateCheck";
    private const string UpdateManifestUrlKey = "UpdateManifestUrl";
    private const string UpdateMirrorPrefixKey = "UpdateMirrorPrefix";
    private readonly ApplicationDataContainer? _packagedSettings;
    private readonly string _settingsPath;
    private readonly Dictionary<string, object?> _unpackagedSettings;

    public SettingsService()
    {
        if (IsPackaged())
        {
            _packagedSettings = ApplicationData.Current.LocalSettings;
            _settingsPath = string.Empty;
            _unpackagedSettings = new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
            return;
        }

        var directory = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData), "Filespace");
        Directory.CreateDirectory(directory);
        _settingsPath = Path.Combine(directory, "settings.json");
        _unpackagedSettings = LoadUnpackagedSettings(_settingsPath);
    }

    public bool WinFEnabled
    {
        get => ReadBool(WinFEnabledKey, true);
        set => Write(WinFEnabledKey, value);
    }

    public bool StartupConfigured => ReadBool(StartupConfiguredKey, false);

    public bool StartupEnabled
    {
        get => ReadBool(StartupEnabledKey, true);
        set { Write(StartupEnabledKey, value); Write(StartupConfiguredKey, true); }
    }

    public bool AutoUpdateCheck
    {
        get => ReadBool(AutoUpdateCheckKey, true);
        set => Write(AutoUpdateCheckKey, value);
    }

    public string UpdateManifestUrl
    {
        get => ReadString(UpdateManifestUrlKey, string.Empty);
        set => Write(UpdateManifestUrlKey, value.Trim());
    }

    public string UpdateMirrorPrefix
    {
        get => ReadString(UpdateMirrorPrefixKey, string.Empty);
        set => Write(UpdateMirrorPrefixKey, value.Trim());
    }

    public SearchProvider SearchProvider
    {
        get => Enum.TryParse<SearchProvider>(ReadString(SearchProviderKey, nameof(Services.SearchProvider.LocalIndex)), out var value)
            ? value
            : Services.SearchProvider.LocalIndex;
        set => Write(SearchProviderKey, value.ToString());
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
        Write(SearchRootsKey, string.Join('|', roots.Where(Directory.Exists).Distinct(StringComparer.OrdinalIgnoreCase)));
    }

    private bool ReadBool(string key, bool fallback)
    {
        var value = ReadValue(key);
        return value switch
        {
            bool boolean => boolean,
            JsonElement element when element.ValueKind is JsonValueKind.True or JsonValueKind.False => element.GetBoolean(),
            _ => fallback
        };
    }

    private string ReadString(string key, string fallback)
    {
        var value = ReadValue(key);
        return value switch
        {
            string text => text,
            JsonElement element when element.ValueKind == JsonValueKind.String => element.GetString() ?? fallback,
            _ => fallback
        };
    }

    private object? ReadValue(string key)
    {
        if (_packagedSettings is not null)
            return _packagedSettings.Values.TryGetValue(key, out var value) ? value : null;
        return _unpackagedSettings.TryGetValue(key, out var localValue) ? localValue : null;
    }

    private void Write(string key, object value)
    {
        if (_packagedSettings is not null)
        {
            _packagedSettings.Values[key] = value;
            return;
        }

        _unpackagedSettings[key] = value;
        File.WriteAllText(_settingsPath, JsonSerializer.Serialize(_unpackagedSettings, new JsonSerializerOptions { WriteIndented = true }));
    }

    private static Dictionary<string, object?> LoadUnpackagedSettings(string path)
    {
        try
        {
            if (File.Exists(path))
            {
                return JsonSerializer.Deserialize<Dictionary<string, object?>>(File.ReadAllText(path))
                    ?? new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
            }
        }
        catch (JsonException) { }
        catch (IOException) { }

        return new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
    }

    private static bool IsPackaged()
    {
        try
        {
            _ = Package.Current.Id;
            return true;
        }
        catch (InvalidOperationException) { return false; }
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
