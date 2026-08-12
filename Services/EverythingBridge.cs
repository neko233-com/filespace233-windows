using System.Diagnostics;
using Filespace233.Models;

namespace Filespace233.Services;

public sealed class EverythingBridge
{
    public async Task<IReadOnlyList<SearchResult>?> TrySearchAsync(string query, CancellationToken cancellationToken)
    {
        var executable = FindExecutable();
        if (executable is null) return null;

        try
        {
            using var process = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = executable,
                    UseShellExecute = false,
                    CreateNoWindow = true,
                    RedirectStandardOutput = true,
                    RedirectStandardError = true
                }
            };
            process.StartInfo.ArgumentList.Add("-n");
            process.StartInfo.ArgumentList.Add("300");
            process.StartInfo.ArgumentList.Add(query);
            process.Start();

            var results = new List<SearchResult>();
            while (await process.StandardOutput.ReadLineAsync(cancellationToken).ConfigureAwait(false) is { } line)
            {
                if (string.IsNullOrWhiteSpace(line)) continue;
                results.Add(new SearchResult
                {
                    FullPath = line,
                    Name = Path.GetFileName(line),
                    IsDirectory = Directory.Exists(line)
                });
            }

            await process.WaitForExitAsync(cancellationToken).ConfigureAwait(false);
            return results;
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception)
        {
            return null;
        }
    }

    public static string? FindExecutable()
    {
        var candidates = new List<string>
        {
            Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ProgramFiles), "Everything", "es.exe"),
            Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ProgramFilesX86), "Everything", "es.exe")
        };

        var path = Environment.GetEnvironmentVariable("PATH");
        if (!string.IsNullOrWhiteSpace(path))
        {
            candidates.AddRange(path.Split(Path.PathSeparator, StringSplitOptions.RemoveEmptyEntries)
                .Select(folder => Path.Combine(folder, "es.exe")));
        }

        return candidates.FirstOrDefault(File.Exists);
    }
}
