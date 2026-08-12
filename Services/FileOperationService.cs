using System.Threading.Channels;
using Filespace233.Models;

namespace Filespace233.Services;

public sealed class FileOperationService : IAsyncDisposable
{
    private readonly Channel<OperationRequest> _queue = Channel.CreateUnbounded<OperationRequest>(new UnboundedChannelOptions { SingleReader = true });
    private readonly Task _worker;

    public FileOperationService()
    {
        _worker = Task.Run(ProcessQueueAsync);
    }

    public Task EnqueueCopyAsync(IEnumerable<FileItem> items, string destinationFolder, CancellationToken cancellationToken = default)
    {
        return EnqueueAsync(async token =>
        {
            Directory.CreateDirectory(destinationFolder);
            foreach (var item in items)
            {
                token.ThrowIfCancellationRequested();
                var destination = GetAvailablePath(Path.Combine(destinationFolder, item.Name));
                if (item.IsDirectory) await CopyDirectoryAsync(item.FullPath, destination, token).ConfigureAwait(false);
                else await CopyFileAsync(item.FullPath, destination, token).ConfigureAwait(false);
            }
        }, cancellationToken);
    }

    public Task EnqueueDeleteAsync(IEnumerable<FileItem> items, CancellationToken cancellationToken = default)
    {
        return EnqueueAsync(token =>
        {
            foreach (var item in items)
            {
                token.ThrowIfCancellationRequested();
                if (item.IsDirectory) Directory.Delete(item.FullPath, recursive: true);
                else File.Delete(item.FullPath);
            }
            return Task.CompletedTask;
        }, cancellationToken);
    }

    public async ValueTask DisposeAsync()
    {
        _queue.Writer.TryComplete();
        await _worker.ConfigureAwait(false);
    }

    private async Task EnqueueAsync(Func<CancellationToken, Task> operation, CancellationToken cancellationToken)
    {
        var completion = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        await _queue.Writer.WriteAsync(new OperationRequest(operation, completion, cancellationToken), cancellationToken).ConfigureAwait(false);
        await completion.Task.ConfigureAwait(false);
    }

    private async Task ProcessQueueAsync()
    {
        await foreach (var request in _queue.Reader.ReadAllAsync())
        {
            try
            {
                await request.Operation(request.CancellationToken).ConfigureAwait(false);
                request.Completion.TrySetResult();
            }
            catch (OperationCanceledException exception) { request.Completion.TrySetCanceled(exception.CancellationToken); }
            catch (Exception exception) { request.Completion.TrySetException(exception); }
        }
    }

    private static async Task CopyFileAsync(string source, string destination, CancellationToken cancellationToken)
    {
        await using var input = new FileStream(source, FileMode.Open, FileAccess.Read, FileShare.Read, 128 * 1024, useAsync: true);
        await using var output = new FileStream(destination, FileMode.CreateNew, FileAccess.Write, FileShare.None, 128 * 1024, useAsync: true);
        await input.CopyToAsync(output, 128 * 1024, cancellationToken).ConfigureAwait(false);
    }

    private static async Task CopyDirectoryAsync(string source, string destination, CancellationToken cancellationToken)
    {
        Directory.CreateDirectory(destination);
        foreach (var directory in Directory.EnumerateDirectories(source))
        {
            cancellationToken.ThrowIfCancellationRequested();
            await CopyDirectoryAsync(directory, Path.Combine(destination, Path.GetFileName(directory)), cancellationToken).ConfigureAwait(false);
        }

        foreach (var file in Directory.EnumerateFiles(source))
        {
            cancellationToken.ThrowIfCancellationRequested();
            await CopyFileAsync(file, Path.Combine(destination, Path.GetFileName(file)), cancellationToken).ConfigureAwait(false);
        }
    }

    private static string GetAvailablePath(string path)
    {
        if (!File.Exists(path) && !Directory.Exists(path)) return path;
        var directory = Path.GetDirectoryName(path) ?? string.Empty;
        var name = Path.GetFileNameWithoutExtension(path);
        var extension = Path.GetExtension(path);
        for (var index = 2; ; index++)
        {
            var candidate = Path.Combine(directory, $"{name} ({index}){extension}");
            if (!File.Exists(candidate) && !Directory.Exists(candidate)) return candidate;
        }
    }

    private sealed record OperationRequest(Func<CancellationToken, Task> Operation, TaskCompletionSource Completion, CancellationToken CancellationToken);
}
