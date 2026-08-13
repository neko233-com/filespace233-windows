using Windows.ApplicationModel;
using Microsoft.Win32;
using System.Diagnostics;
using System.Runtime.InteropServices;

namespace Filespace233.Services;

public sealed class StartupService
{
    public const string TaskId = "FilespaceStartup";
    private const string RunKeyPath = "Software\\Microsoft\\Windows\\CurrentVersion\\Run";

    public async Task<bool> IsEnabledAsync()
    {
        try
        {
            var task = await StartupTask.GetAsync(TaskId);
            return task.State is StartupTaskState.Enabled or StartupTaskState.EnabledByPolicy;
        }
        catch (Exception) when (!IsPackaged())
        {
            return IsUnpackagedStartupEnabled();
        }
        catch (COMException) { return false; }
    }

    public async Task<bool> SetEnabledAsync(bool enabled)
    {
        try
        {
            var task = await StartupTask.GetAsync(TaskId);
            if (enabled)
            {
                var state = await task.RequestEnableAsync();
                return state is StartupTaskState.Enabled or StartupTaskState.EnabledByPolicy;
            }

            task.Disable();
            return true;
        }
        catch (Exception) when (!IsPackaged())
        {
            return SetUnpackagedStartupEnabled(enabled);
        }
        catch (COMException) { return false; }
    }

    private static bool IsUnpackagedStartupEnabled()
    {
        using var key = Registry.CurrentUser.OpenSubKey(RunKeyPath, writable: false);
        return key?.GetValue(TaskId) is string value && !string.IsNullOrWhiteSpace(value);
    }

    private static bool SetUnpackagedStartupEnabled(bool enabled)
    {
        using var key = Registry.CurrentUser.CreateSubKey(RunKeyPath, writable: true);
        if (key is null) return false;

        if (!enabled)
        {
            key.DeleteValue(TaskId, throwOnMissingValue: false);
            return true;
        }

        var executable = Environment.ProcessPath;
        if (string.IsNullOrWhiteSpace(executable) || !File.Exists(executable)) return false;
        key.SetValue(TaskId, $"\"{executable}\" --background", RegistryValueKind.String);
        return true;
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
}
