using System.Runtime.CompilerServices;
using System.Threading.Channels;
using Filespace233.Models;

namespace Filespace233.Services;

public sealed class FileSystemService
{
    public async IAsyncEnumerable<FileItem> EnumerateAsync(
        string path,
        [EnumeratorCancellation] CancellationToken cancellationToken = default)
    {
        var channel = Channel.CreateBounded<FileItem>(new BoundedChannelOptions(128)
        {
            FullMode = BoundedChannelFullMode.Wait,
            SingleWriter = true,
            SingleReader = true
        });

        _ = Task.Run(async () =>
        {
            try
            {
                var options = new EnumerationOptions
                {
                    IgnoreInaccessible = true,
                    RecurseSubdirectories = false,
                    ReturnSpecialDirectories = false,
                    AttributesToSkip = FileAttributes.System
                };

                foreach (var entry in Directory.EnumerateFileSystemEntries(path, "*", options))
                {
                    cancellationToken.ThrowIfCancellationRequested();
                    var item = CreateItem(entry);
                    if (item is not null)
                    {
                        await channel.Writer.WriteAsync(item, cancellationToken).ConfigureAwait(false);
                    }
                }

                channel.Writer.TryComplete();
            }
            catch (Exception exception)
            {
                channel.Writer.TryComplete(exception);
            }
        }, cancellationToken);

        await foreach (var item in channel.Reader.ReadAllAsync(cancellationToken))
        {
            yield return item;
        }
    }

    public static FileItem? CreateItem(string path)
    {
        try
        {
            var attributes = File.GetAttributes(path);
            var isDirectory = attributes.HasFlag(FileAttributes.Directory);
            var modified = File.GetLastWriteTimeUtc(path);
            var size = isDirectory ? 0 : new FileInfo(path).Length;
            return new FileItem
            {
                Name = Path.GetFileName(path),
                FullPath = path,
                IsDirectory = isDirectory,
                Size = size,
                ModifiedUtc = modified
            };
        }
        catch (IOException)
        {
            return null;
        }
        catch (UnauthorizedAccessException)
        {
            return null;
        }
    }
}
