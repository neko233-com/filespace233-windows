using System.Runtime.InteropServices;

namespace Filespace233.Services;

internal sealed class GlobalHotkeyService : IDisposable
{
    private const int WmHotkey = 0x0312;
    private const uint ModWin = 0x0008;
    private const uint VkF = 0x46;
    private const int HotkeyId = 0x233F;
    private const int GwlpWndProc = -4;
    private readonly nint _hwnd;
    private readonly Action _onPressed;
    private readonly nint _previousWndProc;
    private readonly WndProc _wndProc;
    private bool _disposed;

    public bool IsRegistered { get; }

    public GlobalHotkeyService(nint hwnd, Action onPressed)
    {
        _hwnd = hwnd;
        _onPressed = onPressed;
        _wndProc = WindowProcedure;
        _previousWndProc = SetWindowLongPtr(hwnd, GwlpWndProc, Marshal.GetFunctionPointerForDelegate(_wndProc));
        IsRegistered = RegisterHotKey(hwnd, HotkeyId, ModWin, VkF);
    }

    private nint WindowProcedure(nint hwnd, uint message, nint wParam, nint lParam)
    {
        if (message == WmHotkey && wParam.ToInt32() == HotkeyId)
        {
            _onPressed();
            return 0;
        }

        return CallWindowProc(_previousWndProc, hwnd, message, wParam, lParam);
    }

    public void Dispose()
    {
        if (_disposed) return;
        _disposed = true;
        if (IsRegistered) UnregisterHotKey(_hwnd, HotkeyId);
        if (_previousWndProc != 0) SetWindowLongPtr(_hwnd, GwlpWndProc, _previousWndProc);
        GC.KeepAlive(_wndProc);
    }

    [DllImport("user32.dll", SetLastError = true)]
    private static extern bool RegisterHotKey(nint hWnd, int id, uint fsModifiers, uint vk);

    [DllImport("user32.dll", SetLastError = true)]
    private static extern bool UnregisterHotKey(nint hWnd, int id);

    [DllImport("user32.dll", EntryPoint = "SetWindowLongPtrW", SetLastError = true)]
    private static extern nint SetWindowLongPtr(nint hWnd, int nIndex, nint newLong);

    [DllImport("user32.dll", SetLastError = true)]
    private static extern nint CallWindowProc(nint previousWndProc, nint hWnd, uint message, nint wParam, nint lParam);

    [UnmanagedFunctionPointer(CallingConvention.Winapi)]
    private delegate nint WndProc(nint hWnd, uint message, nint wParam, nint lParam);
}
