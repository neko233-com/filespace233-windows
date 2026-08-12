using System.Collections.ObjectModel;
using System.Diagnostics;
using Filespace233.Models;
using Filespace233.Services;
using Microsoft.UI.Input;
using Microsoft.UI.Xaml;
using Microsoft.UI.Xaml.Controls;
using Microsoft.UI.Xaml.Input;
using WinRT.Interop;
using Windows.System;

namespace Filespace233;

public sealed partial class MainWindow : Window
{
    private readonly FileSystemService _fileSystem = new();
    private readonly SearchService _search = new();
    private readonly FileOperationService _operations = new();
    private readonly SettingsService _settings = new();
    private readonly StartupService _startup = new();
    private readonly ObservableCollection<FileItem> _items = new();
    private readonly ObservableCollection<SearchResult> _searchResults = new();
    private readonly List<string> _history = new();
    private int _historyIndex = -1;
    private CancellationTokenSource? _loadCancellation;
    private CancellationTokenSource? _searchCancellation;
    private string _currentPath = string.Empty;
    private bool _dualPane;
    private GlobalHotkeyService? _globalHotkey;

    public MainWindow()
    {
        InitializeComponent();
        Title = "Filespace";
        FileList.ItemsSource = _items;
        SearchResultsList.ItemsSource = _searchResults;
        if (!_settings.StartupConfigured)
        {
            _settings.StartupEnabled = true;
            _startup.SetEnabled(true);
        }
        TabsList.Items.Add(CreateTab("This PC", Environment.GetFolderPath(Environment.SpecialFolder.UserProfile)));
        NavigateTo(Environment.GetFolderPath(Environment.SpecialFolder.UserProfile), addHistory: true);
        RootGrid.KeyDown += RootGrid_KeyDown;
        Activated += MainWindow_Activated;
        Closed += MainWindow_Closed;
    }

    private void MainWindow_Activated(object sender, WindowActivatedEventArgs args)
    {
        if (_globalHotkey is null)
        {
            var hwnd = WindowNative.GetWindowHandle(this);
            _globalHotkey = new GlobalHotkeyService(hwnd, () =>
            {
                Activate();
                SearchBox.Focus(FocusState.Programmatic);
            });
            StatusText.Text = _globalHotkey.IsRegistered
                ? "Win+F is ready. Win+E remains Windows Explorer."
                : "Win+F is unavailable because another app owns it.";
        }
    }

    private void MainWindow_Closed(object sender, WindowEventArgs args)
    {
        _globalHotkey?.Dispose();
        _loadCancellation?.Cancel();
        _searchCancellation?.Cancel();
        _operations.DisposeAsync().AsTask().GetAwaiter().GetResult();
    }

    private void RootGrid_KeyDown(object sender, KeyRoutedEventArgs e)
    {
        var ctrl = InputKeyboardSource.GetKeyStateForCurrentThread(VirtualKey.Control).HasFlag(Windows.UI.Core.CoreVirtualKeyStates.Down);
        if (ctrl && e.Key == VirtualKey.K)
        {
            e.Handled = true;
            OpenSearchOverlay();
        }
        else if (e.Key == VirtualKey.Escape && SearchOverlay.Visibility == Visibility.Visible)
        {
            CloseSearchOverlay();
            e.Handled = true;
        }
    }

    private async void NavigateTo(string path, bool addHistory)
    {
        if (!Directory.Exists(path)) return;
        _currentPath = Path.GetFullPath(path);
        PathBox.Text = _currentPath;
        PaneTitle.Text = new DirectoryInfo(_currentPath).Name;
        if (addHistory)
        {
            if (_historyIndex < _history.Count - 1) _history.RemoveRange(_historyIndex + 1, _history.Count - _historyIndex - 1);
            if (_history.Count == 0 || !string.Equals(_history[^1], _currentPath, StringComparison.OrdinalIgnoreCase)) _history.Add(_currentPath);
            _historyIndex = _history.Count - 1;
        }

        _loadCancellation?.Cancel();
        _loadCancellation?.Dispose();
        _loadCancellation = new CancellationTokenSource();
        var token = _loadCancellation.Token;
        _items.Clear();
        StatusText.Text = "Loading...";
        var stopwatch = Stopwatch.StartNew();

        try
        {
            await foreach (var item in _fileSystem.EnumerateAsync(_currentPath, token))
            {
                _items.Add(item);
                if (_items.Count % 32 == 0) await Task.Yield();
            }
            var ordered = _items.OrderByDescending(item => item.IsDirectory).ThenBy(item => item.Name, StringComparer.OrdinalIgnoreCase).ToList();
            _items.Clear();
            foreach (var item in ordered) _items.Add(item);
            PaneSummary.Text = $"{_items.Count} items";
            StatusText.Text = $"{_items.Count} items";
            PerformanceText.Text = $"Loaded in {stopwatch.ElapsedMilliseconds} ms";
        }
        catch (OperationCanceledException) { }
        catch (Exception exception)
        {
            StatusText.Text = $"Unable to read folder: {exception.Message}";
        }
    }

    private Button CreateTab(string title, string path)
    {
        var button = new Button { Content = title, Tag = path, Padding = new Thickness(14, 7, 14, 7) };
        button.Click += (_, _) => NavigateTo((string)button.Tag, addHistory: true);
        return button;
    }

    private void NewTabButton_Click(object sender, RoutedEventArgs e)
    {
        var tab = CreateTab("New tab", Environment.GetFolderPath(Environment.SpecialFolder.UserProfile));
        TabsList.Items.Add(tab);
        NavigateTo((string)tab.Tag, addHistory: true);
    }

    private void BackButton_Click(object sender, RoutedEventArgs e)
    {
        if (_historyIndex <= 0) return;
        _historyIndex--;
        NavigateTo(_history[_historyIndex], addHistory: false);
    }

    private void ForwardButton_Click(object sender, RoutedEventArgs e)
    {
        if (_historyIndex >= _history.Count - 1) return;
        _historyIndex++;
        NavigateTo(_history[_historyIndex], addHistory: false);
    }

    private void UpButton_Click(object sender, RoutedEventArgs e)
    {
        var parent = Directory.GetParent(_currentPath)?.FullName;
        if (parent is not null) NavigateTo(parent, addHistory: true);
    }

    private void RefreshButton_Click(object sender, RoutedEventArgs e) => NavigateTo(_currentPath, addHistory: false);

    private async void NewFolderButton_Click(object sender, RoutedEventArgs e)
    {
        var nameBox = new TextBox { PlaceholderText = "Folder name" };
        var dialog = new ContentDialog
        {
            Title = "New folder",
            Content = nameBox,
            PrimaryButtonText = "Create",
            CloseButtonText = "Cancel",
            DefaultButton = ContentDialogButton.Primary,
            XamlRoot = RootGrid.XamlRoot
        };
        if (await dialog.ShowAsync() != ContentDialogResult.Primary || string.IsNullOrWhiteSpace(nameBox.Text)) return;
        try
        {
            Directory.CreateDirectory(Path.Combine(_currentPath, nameBox.Text.Trim()));
            RefreshButton_Click(sender, e);
        }
        catch (Exception exception) { StatusText.Text = $"Unable to create folder: {exception.Message}"; }
    }

    private async void CopyButton_Click(object sender, RoutedEventArgs e)
    {
        var selected = FileList.SelectedItems.OfType<FileItem>().ToArray();
        if (selected.Length == 0) { StatusText.Text = "Select one or more items first"; return; }
        try
        {
            StatusText.Text = $"Copying {selected.Length} item(s)...";
            await _operations.EnqueueCopyAsync(selected, _currentPath);
            RefreshButton_Click(sender, e);
            StatusText.Text = "Copy complete";
        }
        catch (Exception exception) { StatusText.Text = $"Copy failed: {exception.Message}"; }
    }

    private async void DeleteButton_Click(object sender, RoutedEventArgs e)
    {
        var selected = FileList.SelectedItems.OfType<FileItem>().ToArray();
        if (selected.Length == 0) { StatusText.Text = "Select one or more items first"; return; }
        var dialog = new ContentDialog
        {
            Title = "Delete selected items?",
            Content = $"This permanently removes {selected.Length} item(s) from disk.",
            PrimaryButtonText = "Delete",
            CloseButtonText = "Cancel",
            DefaultButton = ContentDialogButton.Close,
            XamlRoot = RootGrid.XamlRoot
        };
        if (await dialog.ShowAsync() != ContentDialogResult.Primary) return;
        try
        {
            StatusText.Text = $"Deleting {selected.Length} item(s)...";
            await _operations.EnqueueDeleteAsync(selected);
            RefreshButton_Click(sender, e);
            StatusText.Text = "Delete complete";
        }
        catch (Exception exception) { StatusText.Text = $"Delete failed: {exception.Message}"; }
    }

    private void PathBox_QuerySubmitted(AutoSuggestBox sender, AutoSuggestBoxQuerySubmittedEventArgs args)
    {
        if (!string.IsNullOrWhiteSpace(args.QueryText)) NavigateTo(args.QueryText.Trim(), addHistory: true);
    }

    private void FileList_ItemClick(object sender, ItemClickEventArgs e)
    {
        if (e.ClickedItem is FileItem item) StatusText.Text = item.FullPath;
    }

    private void FileList_DoubleTapped(object sender, DoubleTappedRoutedEventArgs e)
    {
        if (FileList.SelectedItem is not FileItem item) return;
        if (item.IsDirectory) NavigateTo(item.FullPath, addHistory: true);
        else Process.Start(new ProcessStartInfo(item.FullPath) { UseShellExecute = true });
    }

    private void ToggleDualPaneButton_Click(object sender, RoutedEventArgs e)
    {
        _dualPane = !_dualPane;
        RightPane.Visibility = _dualPane ? Visibility.Visible : Visibility.Collapsed;
        RightColumn.Width = _dualPane ? new GridLength(1, GridUnitType.Star) : new GridLength(0);
        LeftColumn.Width = _dualPane ? new GridLength(1, GridUnitType.Star) : new GridLength(1, GridUnitType.Star);
    }

    private void ViewModeBox_SelectionChanged(object sender, SelectionChangedEventArgs e)
    {
        if (FileList is null) return;
        FileList.Padding = ViewModeBox.SelectedIndex == 1 ? new Thickness(0, 4, 0, 4) : new Thickness(0);
    }

    private void SearchBox_TextChanged(AutoSuggestBox sender, AutoSuggestBoxTextChangedEventArgs args)
    {
        if (!string.IsNullOrWhiteSpace(sender.Text)) OpenSearchOverlay(sender.Text);
    }

    private void SearchBox_QuerySubmitted(AutoSuggestBox sender, AutoSuggestBoxQuerySubmittedEventArgs args)
    {
        OpenSearchOverlay(args.QueryText);
        _ = RunSearchAsync(args.QueryText);
    }

    private void OpenSearchOverlay(string query = "")
    {
        SearchOverlay.Visibility = Visibility.Visible;
        OverlaySearchBox.Text = query;
        OverlaySearchBox.Focus(FocusState.Programmatic);
        if (!string.IsNullOrWhiteSpace(query)) _ = RunSearchAsync(query);
    }

    private void CloseSearchOverlay()
    {
        _searchCancellation?.Cancel();
        SearchOverlay.Visibility = Visibility.Collapsed;
        SearchBox.Text = string.Empty;
    }

    private void OverlaySearchBox_QuerySubmitted(AutoSuggestBox sender, AutoSuggestBoxQuerySubmittedEventArgs args) => _ = RunSearchAsync(args.QueryText);

    private async Task RunSearchAsync(string query)
    {
        query = query.Trim();
        if (query.Length < 2) return;
        _searchCancellation?.Cancel();
        _searchCancellation?.Dispose();
        _searchCancellation = new CancellationTokenSource();
        var token = _searchCancellation.Token;
        StatusText.Text = $"Searching for {query}...";
        try
        {
            var results = await _search.SearchAsync(query, _settings.SearchRoots, _settings.SearchProvider, token);
            _searchResults.Clear();
            foreach (var result in results) _searchResults.Add(result);
            StatusText.Text = $"{_searchResults.Count} results";
        }
        catch (OperationCanceledException) { }
        catch (Exception exception) { StatusText.Text = $"Search failed: {exception.Message}"; }
    }

    private void SearchResultsList_ItemClick(object sender, ItemClickEventArgs e)
    {
        if (e.ClickedItem is not SearchResult result) return;
        CloseSearchOverlay();
        if (result.IsDirectory) NavigateTo(result.FullPath, addHistory: true);
        else Process.Start(new ProcessStartInfo(result.FullPath) { UseShellExecute = true });
    }

    private async void SettingsButton_Click(object sender, RoutedEventArgs e)
    {
        var startupCheckBox = new CheckBox { Content = "Start Filespace with Windows for Win+F", IsChecked = _settings.StartupEnabled };
        var providerBox = new ComboBox { SelectedIndex = _settings.SearchProvider == SearchProvider.Everything ? 1 : 0 };
        providerBox.Items.Add(new ComboBoxItem { Content = "Built-in local search" });
        providerBox.Items.Add(new ComboBoxItem { Content = "Everything (es.exe) with local fallback" });
        var dialog = new ContentDialog
        {
            Title = "Filespace settings",
            Content = new StackPanel
            {
                Spacing = 14,
                Children =
                {
                    new TextBlock { Text = "Win+F opens Filespace. Win+E remains owned by Windows Explorer.", TextWrapping = TextWrapping.Wrap },
                    startupCheckBox,
                    new TextBlock { Text = "Search provider" },
                    providerBox,
                    new TextBlock { Text = "Search roots use your profile, Desktop, Documents and Downloads by default.", Opacity = 0.65, TextWrapping = TextWrapping.Wrap }
                }
            },
            PrimaryButtonText = "Save",
            CloseButtonText = "Cancel",
            XamlRoot = RootGrid.XamlRoot
        };
        if (await dialog.ShowAsync() != ContentDialogResult.Primary) return;
        var startupEnabled = startupCheckBox.IsChecked == true;
        _settings.StartupEnabled = startupEnabled;
        _startup.SetEnabled(startupEnabled);
        _settings.SearchProvider = providerBox.SelectedIndex == 1 ? SearchProvider.Everything : SearchProvider.LocalIndex;
        StatusText.Text = "Settings saved";
    }
}
