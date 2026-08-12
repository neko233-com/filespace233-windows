using Filespace233.Models;

namespace Filespace233.Services;

public sealed class SearchService
{
    private readonly EverythingBridge _everything = new();

    public async Task<IReadOnlyList<SearchResult>> SearchAsync(
        string query,
        IReadOnlyList<string> roots,
        SearchProvider provider,
        CancellationToken cancellationToken)
    {
        if (provider == SearchProvider.Everything)
        {
            var everythingResults = await _everything.TrySearchAsync(query, cancellationToken).ConfigureAwait(false);
            if (everythingResults is not null) return everythingResults;
        }

        return await SearchLocallyAsync(query, roots, cancellationToken).ConfigureAwait(false);
    }

    private static Task<IReadOnlyList<SearchResult>> SearchLocallyAsync(
        string query,
        IReadOnlyList<string> roots,
        CancellationToken cancellationToken)
    {
        return Task.Run<IReadOnlyList<SearchResult>>(() =>
        {
            var results = new List<SearchResult>(capacity: 64);
            var pending = new Stack<string>(roots.Reverse());
            var comparison = StringComparison.OrdinalIgnoreCase;

            while (pending.Count > 0 && results.Count < 300)
            {
                cancellationToken.ThrowIfCancellationRequested();
                var folder = pending.Pop();
                IEnumerable<string> entries;
                try
                {
                    entries = Directory.EnumerateFileSystemEntries(folder, "*", new EnumerationOptions
                    {
                        IgnoreInaccessible = true,
                        RecurseSubdirectories = false,
                        ReturnSpecialDirectories = false,
                        AttributesToSkip = FileAttributes.System
                    });
                }
                catch (IOException) { continue; }
                catch (UnauthorizedAccessException) { continue; }

                foreach (var entry in entries)
                {
                    cancellationToken.ThrowIfCancellationRequested();
                    var name = Path.GetFileName(entry);
                    var isDirectory = Directory.Exists(entry);
                    if (name.Contains(query, comparison))
                    {
                        results.Add(new SearchResult
                        {
                            Name = name,
                            FullPath = entry,
                            IsDirectory = isDirectory
                        });
                        if (results.Count >= 300) break;
                    }

                    if (isDirectory) pending.Push(entry);
                }
            }

            return results;
        }, cancellationToken);
    }
}
