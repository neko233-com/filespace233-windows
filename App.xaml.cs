using Microsoft.UI.Xaml;
using WinRT.Interop;

namespace Filespace233;

public partial class App : Application
{
    public static MainWindow? MainWindow { get; private set; }

    public App()
    {
        InitializeComponent();
    }

    protected override void OnLaunched(Microsoft.UI.Xaml.LaunchActivatedEventArgs args)
    {
        MainWindow ??= new MainWindow();
        MainWindow.Activate();
        var backgroundLaunch = args.Arguments.Contains("--background", StringComparison.OrdinalIgnoreCase);
        if (backgroundLaunch)
        {
            var hwnd = WindowNative.GetWindowHandle(MainWindow);
            var windowId = Microsoft.UI.Win32Interop.GetWindowIdFromWindow(hwnd);
            Microsoft.UI.Windowing.AppWindow.GetFromWindowId(windowId)?.Hide();
        }
    }
}
