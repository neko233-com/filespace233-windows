using System.Collections.ObjectModel;
using System.Diagnostics;
using Filespace233.Models;
using Filespace233.Services;
using Microsoft.UI;
using Microsoft.UI.Input;
using Microsoft.UI.Windowing;
using Microsoft.UI.Xaml;
using Microsoft.UI.Xaml.Controls;
using Microsoft.UI.Xaml.Input;
using Microsoft.UI.Xaml.Media;
using WinRT.Interop;
using Windows.Foundation;
using Windows.Graphics;
using Windows.Storage.Pickers;
using Windows.System;

namespace Filespace233;

public sealed partial class MainWindow : Window
{
    private FileSystemService? _fileSystem;
    private SearchService? _search;
    private FileOperationService? _operations;
    private SettingsService? _settings;
    private StartupService? _startup;
    private UpdateService? _updates;
    private readonly ObservableCollection<FileItem> _items = new();
    private readonly ObservableCollection<FileItem> _rightItems = new();
    private readonly ObservableCollection<SearchResult> _searchResults = new();
    private readonly List<string> _history = new();
    private int _historyIndex = -1;
    private CancellationTokenSource? _loadCancellation;
    private CancellationTokenSource? _rightLoadCancellation;
    private CancellationTokenSource? _searchCancellation;
    private string _currentPath = string.Empty;
    private string _rightPath = string.Empty;
    private bool _dualPane;
    private bool _updatesChecked;
    private bool _initializationQueued;
    private GlobalHotkeyService? _globalHotkey;

    private FileSystemService FileSystem => _fileSystem ??= new();
    private SearchService Search => _search ??= new();
    private FileOperationService Operations => _operations ??= new();
    private SettingsService Settings => _settings ??= new();
    private StartupService Startup => _startup ??= new();
    private UpdateService Updates => _updates ??= new();

    public MainWindow()
    {
        InitializeComponent();
        Title = "Filespace";
        ExtendsContentIntoTitleBar = true;
        SetTitleBar(AppTitleBar);
        AppWindow.SetIcon("Assets/FilespaceLogo.png");
        ConfigureWindow();
        FileList.ItemsSource = _items;
        RightFileList.ItemsSource = _rightItems;
        SearchResultsList.ItemsSource = _searchResults;
        TabsList.Items.Add(CreateTab("This PC", Environment.GetFolderPath(Environment.SpecialFolder.UserProfile)));
        RootGrid.KeyDown += RootGrid_KeyDown;
        RootGrid.AddHandler(UIElement.PointerWheelChangedEvent, new PointerEventHandler(RootGrid_PointerWheelChanged), handledEventsToo: true);
        TabsScrollViewer.AddHandler(UIElement.PointerWheelChangedEvent, new PointerEventHandler(TabsScrollViewer_PointerWheelChanged), handledEventsToo: true);
        Activated += MainWindow_Activated;
        Closed += MainWindow_Closed;
        RootGrid.Loaded += RootGrid_Loaded;
    }

    private void ConfigureWindow()
    {
        var hwnd = WindowNative.GetWindowHandle(this);
        var windowId = Win32Interop.GetWindowIdFromWindow(hwnd);
        var appWindow = AppWindow.GetFromWindowId(windowId);
        appWindow.Resize(new SizeInt32(1280, 800));
        if (appWindow.Presenter is OverlappedPresenter presenter)
        {
            presenter.PreferredMinimumWidth = 960;
            presenter.PreferredMinimumHeight = 640;
            presenter.IsResizable = true;
        }
    }

    private void MainWindow_Activated(object sender, WindowActivatedEventArgs args)
    {
        EnsureGlobalHotkey();
    }

    private void RootGrid_Loaded(object sender, RoutedEventArgs e)
    {
        if (_initializationQueued) return;
        _initializationQueued = true;
        _ = InitializeAfterLoadAsync();
    }

    private async Task InitializeAfterLoadAsync()
    {
        await Task.Delay(500);
        if (string.IsNullOrWhiteSpace(_currentPath))
            NavigateTo(Environment.GetFolderPath(Environment.SpecialFolder.UserProfile), addHistory: true);

        await Task.Delay(1500);
        if (_updatesChecked) return;
        _updatesChecked = true;
        if (Settings.AutoUpdateCheck) _ = CheckForUpdatesAsync(silent: true);
    }

    private void EnsureGlobalHotkey()
    {
        if (!Settings.WinFEnabled || _globalHotkey is not null) return;
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

    private void MainWindow_Closed(object sender, WindowEventArgs args)
    {
        _globalHotkey?.Dispose();
        _loadCancellation?.Cancel();
        _rightLoadCancellation?.Cancel();
        _searchCancellation?.Cancel();
        _operations?.DisposeAsync().AsTask().GetAwaiter().GetResult();
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

    private void RootGrid_PointerWheelChanged(object sender, PointerRoutedEventArgs e)
    {
        var point = e.GetCurrentPoint(RootGrid);
        if (e.Handled) return;

        var scrollViewer = FindScrollViewerAt(point.Position);
        ScrollWithWheel(scrollViewer, point.Properties.MouseWheelDelta, e);
    }

    private void TabsScrollViewer_PointerWheelChanged(object sender, PointerRoutedEventArgs e)
    {
        if (e.Handled || e.GetCurrentPoint(TabsScrollViewer).Properties.MouseWheelDelta == 0) return;

        var delta = e.GetCurrentPoint(TabsScrollViewer).Properties.MouseWheelDelta;
        if (TabsScrollViewer.ExtentWidth <= TabsScrollViewer.ViewportWidth) return;

        var maximumOffset = Math.Max(0, TabsScrollViewer.ExtentWidth - TabsScrollViewer.ViewportWidth);
        var nextOffset = Math.Clamp(TabsScrollViewer.HorizontalOffset - delta * 0.75, 0, maximumOffset);
        TabsScrollViewer.ChangeView(nextOffset, null, null, disableAnimation: true);
        e.Handled = true;
    }

    private static void ScrollWithWheel(ScrollViewer? scrollViewer, int delta, PointerRoutedEventArgs e)
    {
        if (scrollViewer is null || delta == 0 || scrollViewer.ExtentHeight <= scrollViewer.ViewportHeight) return;

        var maximumOffset = Math.Max(0, scrollViewer.ExtentHeight - scrollViewer.ViewportHeight);
        var nextOffset = Math.Clamp(scrollViewer.VerticalOffset - delta * 0.75, 0, maximumOffset);
        scrollViewer.ChangeView(null, nextOffset, null, disableAnimation: true);
        e.Handled = true;
    }

    private ScrollViewer? FindScrollViewerAt(Point position)
    {
        if (SearchOverlay.Visibility == Visibility.Visible && IsPointInside(SearchResultsList, position))
            return FindDescendant<ScrollViewer>(SearchResultsList);

        if (IsPointInside(FileList, position))
            return FindDescendant<ScrollViewer>(FileList);

        if (_dualPane && IsPointInside(RightFileList, position))
            return FindDescendant<ScrollViewer>(RightFileList);

        if (IsPointInside(SidebarScrollViewer, position))
            return SidebarScrollViewer;

        return null;
    }

    private bool IsPointInside(FrameworkElement element, Point point)
    {
        if (element.Visibility != Visibility.Visible || element.ActualWidth <= 0 || element.ActualHeight <= 0)
            return false;

        var origin = element.TransformToVisual(RootGrid).TransformPoint(new Point(0, 0));
        return point.X >= origin.X && point.X <= origin.X + element.ActualWidth
            && point.Y >= origin.Y && point.Y <= origin.Y + element.ActualHeight;
    }

    private static T? FindDescendant<T>(DependencyObject root) where T : DependencyObject
    {
        for (var index = 0; index < VisualTreeHelper.GetChildrenCount(root); index++)
        {
            var child = VisualTreeHelper.GetChild(root, index);
            if (child is T match) return match;

            var descendant = FindDescendant<T>(child);
            if (descendant is not null) return descendant;
        }

        return null;
    }

    private async void NavigateTo(string path, bool addHistory)
    {
        if (!Directory.Exists(path)) return;
        _currentPath = Path.GetFullPath(path);
        PathBox.Text = _currentPath;
        PaneTitle.Text = GetDisplayFolderName(_currentPath);
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
            await foreach (var item in FileSystem.EnumerateAsync(_currentPath, token))
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

    private async void NavigateRightTo(string path)
    {
        if (!Directory.Exists(path)) return;
        _rightPath = Path.GetFullPath(path);
        RightPanePath.Text = _rightPath;
        RightPaneTitle.Text = GetDisplayFolderName(_rightPath);
        _rightLoadCancellation?.Cancel();
        _rightLoadCancellation?.Dispose();
        _rightLoadCancellation = new CancellationTokenSource();
        var token = _rightLoadCancellation.Token;
        _rightItems.Clear();

        try
        {
            await foreach (var item in FileSystem.EnumerateAsync(_rightPath, token))
            {
                _rightItems.Add(item);
                if (_rightItems.Count % 32 == 0) await Task.Yield();
            }

            var ordered = _rightItems.OrderByDescending(item => item.IsDirectory).ThenBy(item => item.Name, StringComparer.OrdinalIgnoreCase).ToList();
            _rightItems.Clear();
            foreach (var item in ordered) _rightItems.Add(item);
            RightPaneSummary.Text = $"{_rightItems.Count} items";
        }
        catch (OperationCanceledException) { }
        catch (Exception exception)
        {
            RightPaneSummary.Text = $"Unable to read folder: {exception.Message}";
        }
    }

    private static string GetDisplayFolderName(string path)
    {
        var directory = new DirectoryInfo(path);
        return string.IsNullOrWhiteSpace(directory.Name) ? directory.Root.Name : directory.Name;
    }

    private Button CreateTab(string title, string path)
    {
        var button = new Button
        {
            Tag = path,
            Padding = new Thickness(14, 7, 14, 7),
            Margin = new Thickness(0, 0, 4, 0),
            Content = new StackPanel
            {
                Orientation = Orientation.Horizontal,
                Spacing = 8,
                Children =
                {
                    new SymbolIcon { Symbol = Symbol.Folder, Width = 16, Height = 16 },
                    new TextBlock { Text = title, VerticalAlignment = VerticalAlignment.Center }
                }
            }
        };
        button.Click += (_, _) => NavigateTo((string)button.Tag, addHistory: true);
        return button;
    }

    private void SidebarLocation_Click(object sender, RoutedEventArgs e)
    {
        if (sender is not Button button || button.Tag is not string location) return;
        var path = location switch
        {
            "home" or "profile" => Environment.GetFolderPath(Environment.SpecialFolder.UserProfile),
            "desktop" => Environment.GetFolderPath(Environment.SpecialFolder.DesktopDirectory),
            "documents" => Environment.GetFolderPath(Environment.SpecialFolder.MyDocuments),
            "downloads" => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.UserProfile), "Downloads"),
            "pictures" => Environment.GetFolderPath(Environment.SpecialFolder.MyPictures),
            _ => string.Empty
        };
        if (!string.IsNullOrWhiteSpace(path) && Directory.Exists(path)) NavigateTo(path, addHistory: true);
    }

    private void SearchButton_Click(object sender, RoutedEventArgs e) => OpenSearchOverlay();

    private void CloseSearchButton_Click(object sender, RoutedEventArgs e) => CloseSearchOverlay();

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
            await Operations.EnqueueCopyAsync(selected, _currentPath);
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
            await Operations.EnqueueDeleteAsync(selected);
            RefreshButton_Click(sender, e);
            StatusText.Text = "Delete complete";
        }
        catch (Exception exception) { StatusText.Text = $"Delete failed: {exception.Message}"; }
    }

    private void PathBox_KeyDown(object sender, KeyRoutedEventArgs e)
    {
        if (e.Key != VirtualKey.Enter || string.IsNullOrWhiteSpace(PathBox.Text)) return;
        NavigateTo(PathBox.Text.Trim(), addHistory: true);
        e.Handled = true;
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
        if (_dualPane && string.IsNullOrWhiteSpace(_rightPath)) NavigateRightTo(_currentPath);
    }

    private void RightFileList_DoubleTapped(object sender, DoubleTappedRoutedEventArgs e)
    {
        if (RightFileList.SelectedItem is not FileItem item) return;
        if (item.IsDirectory) NavigateRightTo(item.FullPath);
        else Process.Start(new ProcessStartInfo(item.FullPath) { UseShellExecute = true });
    }

    private void RightPanePath_KeyDown(object sender, KeyRoutedEventArgs e)
    {
        if (e.Key != VirtualKey.Enter || string.IsNullOrWhiteSpace(RightPanePath.Text)) return;
        NavigateRightTo(RightPanePath.Text.Trim());
        e.Handled = true;
    }

    private void RightPaneOpenButton_Click(object sender, RoutedEventArgs e)
    {
        if (!string.IsNullOrWhiteSpace(_rightPath)) NavigateTo(_rightPath, addHistory: true);
    }

    private void ViewModeBox_SelectionChanged(object sender, SelectionChangedEventArgs e)
    {
        if (FileList is null) return;
        FileList.Padding = ViewModeBox.SelectedIndex == 1 ? new Thickness(0, 4, 0, 4) : new Thickness(0);
    }

    private void SearchBox_KeyDown(object sender, KeyRoutedEventArgs e)
    {
        if (e.Key != VirtualKey.Enter || sender is not TextBox textBox) return;
        OpenSearchOverlay(textBox.Text);
        _ = RunSearchAsync(textBox.Text);
        e.Handled = true;
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

    private void OverlaySearchBox_KeyDown(object sender, KeyRoutedEventArgs e)
    {
        if (e.Key != VirtualKey.Enter || sender is not TextBox textBox) return;
        _ = RunSearchAsync(textBox.Text);
        e.Handled = true;
    }

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
            var results = await Search.SearchAsync(query, Settings.SearchRoots, Settings.SearchProvider, token);
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
        var currentStartupEnabled = await Startup.IsEnabledAsync();
        var startupCheckBox = new CheckBox
        {
            Content = "Start Filespace with Windows for Win+F",
            IsChecked = currentStartupEnabled
        };
        var winFCheckBox = new CheckBox
        {
            Content = "Enable Win+F to open Filespace",
            IsChecked = Settings.WinFEnabled
        };
        var autoUpdateCheckBox = new CheckBox { Content = "Check for updates when Filespace starts", IsChecked = Settings.AutoUpdateCheck };
        var providerBox = new ComboBox { SelectedIndex = Settings.SearchProvider == SearchProvider.Everything ? 1 : 0 };
        providerBox.Items.Add(new ComboBoxItem { Content = "Built-in local search" });
        providerBox.Items.Add(new ComboBoxItem { Content = "Everything (es.exe) with local fallback" });
        var manifestBox = new TextBox { Header = "Trusted update manifest URL (optional)", Text = Settings.UpdateManifestUrl, PlaceholderText = "https://.../latest.json" };
        var mirrorBox = new TextBox { Header = "Trusted mirror prefix (optional)", Text = Settings.UpdateMirrorPrefix, PlaceholderText = "https://mirror.example/filespace/{file}" };
        var checkUpdatesButton = new Button { Content = "Check for updates now" };
        var installMsiButton = new Button { Content = "Install an MSI downloaded manually" };
        ContentDialog? dialog = null;
        checkUpdatesButton.Click += (_, _) =>
        {
            dialog?.Hide();
            _ = CheckForUpdatesAsync(silent: false);
        };
        installMsiButton.Click += async (_, _) =>
        {
            dialog?.Hide();
            await InstallLocalMsiAsync();
        };
        dialog = new ContentDialog
        {
            Title = "Filespace settings",
            Content = new ScrollViewer
            {
                MaxHeight = 520,
                VerticalScrollBarVisibility = ScrollBarVisibility.Auto,
                Content = new StackPanel
                {
                    Spacing = 14,
                    Children =
                    {
                        new TextBlock { Text = "Win+F opens Filespace. Win+E remains owned by Windows Explorer.", TextWrapping = TextWrapping.Wrap },
                        winFCheckBox,
                        startupCheckBox,
                        autoUpdateCheckBox,
                        new TextBlock { Text = "Search provider" },
                        providerBox,
                        new TextBlock { Text = "Search roots use your profile, Desktop, Documents and Downloads by default.", Opacity = 0.65, TextWrapping = TextWrapping.Wrap },
                        manifestBox,
                        mirrorBox,
                        checkUpdatesButton,
                        installMsiButton,
                        new TextBlock { Text = "Updates are never installed silently. The download is verified with SHA-256 before Windows Installer is opened.", Opacity = 0.65, TextWrapping = TextWrapping.Wrap }
                    }
                }
            },
            PrimaryButtonText = "Save",
            CloseButtonText = "Cancel",
            XamlRoot = RootGrid.XamlRoot
        };
        if (await dialog.ShowAsync() != ContentDialogResult.Primary) return;
        Settings.WinFEnabled = winFCheckBox.IsChecked == true;
        _globalHotkey?.Dispose();
        _globalHotkey = null;
        EnsureGlobalHotkey();
        var startupEnabled = startupCheckBox.IsChecked == true;
        Settings.StartupEnabled = startupEnabled;
        if (!await Startup.SetEnabledAsync(startupEnabled))
        {
            StatusText.Text = "Unable to update the Windows startup setting.";
            return;
        }
        Settings.AutoUpdateCheck = autoUpdateCheckBox.IsChecked == true;
        Settings.UpdateManifestUrl = manifestBox.Text;
        Settings.UpdateMirrorPrefix = mirrorBox.Text;
        Settings.SearchProvider = providerBox.SelectedIndex == 1 ? SearchProvider.Everything : SearchProvider.LocalIndex;
        StatusText.Text = "Settings saved";
    }

    private async Task InstallLocalMsiAsync()
    {
        var picker = new FileOpenPicker
        {
            SuggestedStartLocation = PickerLocationId.Downloads,
            ViewMode = PickerViewMode.List
        };
        picker.FileTypeFilter.Add(".msi");
        InitializeWithWindow.Initialize(picker, WindowNative.GetWindowHandle(this));

        var file = await picker.PickSingleFileAsync();
        if (file is null) return;

        try
        {
            UpdateService.InstallMsi(file.Path);
            StatusText.Text = "MSI installer started";
        }
        catch (Exception exception)
        {
            StatusText.Text = $"Unable to start MSI installer: {exception.Message}";
        }
    }

    private async Task CheckForUpdatesAsync(bool silent)
    {
        try
        {
            var update = await Updates.CheckAsync(Settings);
            if (update is null)
            {
                if (!silent) StatusText.Text = "Filespace is up to date";
                return;
            }

            var dialog = new ContentDialog
            {
                Title = $"Filespace {update.Manifest.Version} is available",
                Content = new TextBlock { Text = "Download the verified MSI update now, or open the release page to review it first.", TextWrapping = TextWrapping.Wrap },
                PrimaryButtonText = "Download and install",
                SecondaryButtonText = "Open release page",
                CloseButtonText = "Later",
                XamlRoot = RootGrid.XamlRoot
            };
            var result = await dialog.ShowAsync();
            if (result == ContentDialogResult.Secondary)
            {
                Process.Start(new ProcessStartInfo(update.Manifest.ReleasePage) { UseShellExecute = true });
            }
            else if (result == ContentDialogResult.Primary)
            {
                StatusText.Text = "Downloading update...";
                await Updates.DownloadAndInstallAsync(update);
                StatusText.Text = "Update installer started";
            }
        }
        catch (OperationCanceledException) { }
        catch (Exception exception)
        {
            if (!silent) StatusText.Text = $"Update check failed: {exception.Message}";
        }
    }
}
