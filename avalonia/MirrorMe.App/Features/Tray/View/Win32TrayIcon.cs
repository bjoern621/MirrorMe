using System.Runtime.InteropServices;
using Avalonia.Threading;

namespace MirrorMe.App.Features.Tray.View;

/// <summary>
/// The tray icon on Windows, registered with the shell directly so its right-click menu is the system's own.
///
/// Avalonia's Windows tray draws its menu as an Avalonia window wearing the app's theme
/// (<c>Avalonia.Win32.TrayIconImpl.OnRightClicked</c>), which in this app is the dark design palette.
/// A tray menu belongs to the desktop, so this one is a Win32 popup menu
/// (<c>TrackPopupMenuEx</c>), drawn by Windows in its own style and following the system's light or dark mode.
///
/// Lives on the UI thread: the hidden owner window is created there,
/// so Avalonia's message pump dispatches its messages.
/// </summary>
internal sealed class Win32TrayIcon : IDisposable
{
    private const uint IconId = 1;
    private const uint CallbackMessage = WM_APP + 1;

    private static readonly WndProc s_wndProc = Dispatch;
    private static readonly Dictionary<IntPtr, Win32TrayIcon> s_icons = new();
    private static readonly uint s_taskbarCreated = RegisterWindowMessageW("TaskbarCreated");
    private static string? s_className;

    private readonly IntPtr _window;
    private readonly string _tip;
    private readonly Action _clicked;
    private readonly Func<IReadOnlyList<TrayRow>> _menu;
    private IntPtr _icon;
    private bool _disposed;

    private Win32TrayIcon(IntPtr window, string tip, Action clicked, Func<IReadOnlyList<TrayRow>> menu)
    {
        _window = window;
        _tip = tip.Length > 127 ? tip[..127] : tip;
        _clicked = clicked;
        _menu = menu;
    }

    /// <summary>
    /// Registers an icon with the shell. Throws where no owner window can be made;
    /// a shell not running yet is not a failure, the icon arriving with its <c>TaskbarCreated</c>.
    /// </summary>
    public static Win32TrayIcon Create(string tip, byte[] png, Action clicked, Func<IReadOnlyList<TrayRow>> menu)
    {
        var instance = GetModuleHandleW(null);
        if (s_className is null)
        {
            var name = "MirrorMe.TrayOwner";
            var type = new WNDCLASSEXW
            {
                cbSize = (uint)Marshal.SizeOf<WNDCLASSEXW>(),
                lpfnWndProc = Marshal.GetFunctionPointerForDelegate(s_wndProc),
                hInstance = instance,
                lpszClassName = name,
            };
            if (RegisterClassExW(ref type) == 0)
            {
                throw new InvalidOperationException($"RegisterClassEx failed ({Marshal.GetLastWin32Error()})");
            }

            s_className = name;
        }

        // A top-level window rather than a message-only one: TaskbarCreated is a broadcast,
        // and broadcasts reach top-level windows only. It is never shown.
        var window = CreateWindowExW(0, s_className, "", 0, 0, 0, 0, 0, IntPtr.Zero, IntPtr.Zero, instance, IntPtr.Zero);
        if (window == IntPtr.Zero)
        {
            throw new InvalidOperationException($"CreateWindowEx failed ({Marshal.GetLastWin32Error()})");
        }

        FollowSystemTheme(window);

        var icon = new Win32TrayIcon(window, tip, clicked, menu);
        s_icons[window] = icon;
        icon.SetIcon(png);
        return icon;
    }

    /// <summary>Swaps the image the shell draws.</summary>
    public void SetIcon(byte[] png)
    {
        if (_disposed)
        {
            return;
        }

        var next = CreateIconFromResourceEx(
            png, (uint)png.Length, true, 0x00030000,
            GetSystemMetrics(SM_CXSMICON), GetSystemMetrics(SM_CYSMICON), 0);

        var previous = _icon;
        _icon = next;
        Notify(previous == IntPtr.Zero ? NIM_ADD : NIM_MODIFY);

        if (previous != IntPtr.Zero)
        {
            DestroyIcon(previous);
        }
    }

    public void Dispose()
    {
        if (_disposed)
        {
            return;
        }

        _disposed = true;
        Notify(NIM_DELETE);
        s_icons.Remove(_window);
        DestroyWindow(_window);
        if (_icon != IntPtr.Zero)
        {
            DestroyIcon(_icon);
            _icon = IntPtr.Zero;
        }
    }

    private void Notify(uint message)
    {
        var data = new NOTIFYICONDATAW
        {
            cbSize = (uint)Marshal.SizeOf<NOTIFYICONDATAW>(),
            hWnd = _window,
            uID = IconId,
            uFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP,
            uCallbackMessage = CallbackMessage,
            hIcon = _icon,
            szTip = _tip,
            szInfo = "",
            szInfoTitle = "",
        };

        // False where the shell is not up; TaskbarCreated adds the icon once it is.
        Shell_NotifyIconW(message, ref data);
    }

    private static IntPtr Dispatch(IntPtr window, uint message, IntPtr wParam, IntPtr lParam)
    {
        if (s_icons.TryGetValue(window, out var icon))
        {
            if (message == CallbackMessage)
            {
                switch ((uint)lParam.ToInt64() & 0xFFFF)
                {
                    case WM_LBUTTONUP:
                        Dispatcher.UIThread.Post(icon._clicked);
                        break;
                    case WM_RBUTTONUP:
                        icon.ShowMenu();
                        break;
                }

                return IntPtr.Zero;
            }

            if (message == s_taskbarCreated)
            {
                // Explorer restarted: the old registration went with it.
                icon.Notify(NIM_ADD);
                return IntPtr.Zero;
            }

            if (message == WM_SETTINGCHANGE)
            {
                // The system's light or dark switch; the next menu draws in the new one.
                FlushMenuThemesSafe();
            }
        }

        return DefWindowProcW(window, message, wParam, lParam);
    }

    private void ShowMenu()
    {
        var actions = new List<TrayRow>();
        var menu = Build(_menu(), actions);
        try
        {
            GetCursorPos(out var at);

            // Without the owner in the foreground the menu stays open after a click elsewhere,
            // and without the WM_NULL after it the next right-click opens nothing (KB135788).
            SetForegroundWindow(_window);
            var flags = TPM_RETURNCMD | TPM_NONOTIFY | TPM_RIGHTBUTTON | TPM_BOTTOMALIGN
                | (GetSystemMetrics(SM_MENUDROPALIGNMENT) != 0 ? TPM_RIGHTALIGN : TPM_LEFTALIGN);
            var picked = TrackPopupMenuEx(menu, flags, at.X, at.Y, _window, IntPtr.Zero);
            PostMessageW(_window, WM_NULL, IntPtr.Zero, IntPtr.Zero);

            if (picked > 0 && picked <= actions.Count)
            {
                Dispatcher.UIThread.Post(actions[picked - 1].Invoke);
            }
        }
        finally
        {
            // Takes the submenus with it.
            DestroyMenu(menu);
        }
    }

    /// <summary>One popup menu from the rows, each actionable row numbered by its place in <paramref name="actions"/>.</summary>
    private static IntPtr Build(IReadOnlyList<TrayRow> rows, List<TrayRow> actions)
    {
        var menu = CreatePopupMenu();

        foreach (var row in rows)
        {
            if (row.IsSeparator)
            {
                AppendMenuW(menu, MF_SEPARATOR, UIntPtr.Zero, null);
                continue;
            }

            // A lone & is a mnemonic marker to Win32; a name may carry one as text.
            var label = row.Label!.Replace("&", "&&");
            var enabled = row.CanAct ? 0u : MF_GRAYED;

            if (row.Rows is { } children)
            {
                var sub = Build(children, actions);
                AppendMenuW(menu, MF_POPUP | enabled, (UIntPtr)(ulong)sub, label);
                continue;
            }

            actions.Add(row);
            var id = (uint)actions.Count;
            AppendMenuW(menu, MF_STRING | enabled | (row.IsChecked ? MF_CHECKED : 0), (UIntPtr)id, label);

            if (row.IsRadio)
            {
                var info = new MENUITEMINFOW
                {
                    cbSize = (uint)Marshal.SizeOf<MENUITEMINFOW>(),
                    fMask = MIIM_FTYPE,
                    fType = MFT_RADIOCHECK,
                };
                SetMenuItemInfoW(menu, id, false, ref info);
            }
        }

        return menu;
    }

    /// <summary>
    /// Lets the menu follow the system's dark mode. Win32 menus draw light unless the process opts in,
    /// which Windows offers only through uxtheme's unnamed exports (Windows 10 1903 and later);
    /// where they are missing the menu stays light, which is still the system's own style.
    /// </summary>
    private static void FollowSystemTheme(IntPtr window)
    {
        if (Environment.OSVersion.Version.Build < 18362)
        {
            return;
        }

        try
        {
            SetPreferredAppMode(AllowDark);
            AllowDarkModeForWindow(window, true);
            FlushMenuThemes();
        }
        catch (EntryPointNotFoundException)
        {
        }
    }

    private static void FlushMenuThemesSafe()
    {
        if (Environment.OSVersion.Version.Build < 18362)
        {
            return;
        }

        try
        {
            FlushMenuThemes();
        }
        catch (EntryPointNotFoundException)
        {
        }
    }

    // ---- Win32 ----

    private delegate IntPtr WndProc(IntPtr window, uint message, IntPtr wParam, IntPtr lParam);

    private const uint WM_NULL = 0x0000;
    private const uint WM_SETTINGCHANGE = 0x001A;
    private const uint WM_LBUTTONUP = 0x0202;
    private const uint WM_RBUTTONUP = 0x0205;
    private const uint WM_APP = 0x8000;

    private const uint NIM_ADD = 0;
    private const uint NIM_MODIFY = 1;
    private const uint NIM_DELETE = 2;
    private const uint NIF_MESSAGE = 0x1;
    private const uint NIF_ICON = 0x2;
    private const uint NIF_TIP = 0x4;

    private const uint MF_STRING = 0x0;
    private const uint MF_GRAYED = 0x1;
    private const uint MF_CHECKED = 0x8;
    private const uint MF_POPUP = 0x10;
    private const uint MF_SEPARATOR = 0x800;
    private const uint MIIM_FTYPE = 0x100;
    private const uint MFT_RADIOCHECK = 0x200;

    private const uint TPM_LEFTALIGN = 0x0;
    private const uint TPM_RIGHTBUTTON = 0x2;
    private const uint TPM_RIGHTALIGN = 0x8;
    private const uint TPM_BOTTOMALIGN = 0x20;
    private const uint TPM_NONOTIFY = 0x80;
    private const uint TPM_RETURNCMD = 0x100;

    private const int SM_MENUDROPALIGNMENT = 40;
    private const int SM_CXSMICON = 49;
    private const int SM_CYSMICON = 50;

    private const int AllowDark = 1;

    // Interop layouts: fields the marshaler fills or Windows reads, which the compiler sees no code touch.
#pragma warning disable CS0169, CS0649

    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    private struct WNDCLASSEXW
    {
        public uint cbSize;
        public uint style;
        public IntPtr lpfnWndProc;
        public int cbClsExtra;
        public int cbWndExtra;
        public IntPtr hInstance;
        public IntPtr hIcon;
        public IntPtr hCursor;
        public IntPtr hbrBackground;
        public string? lpszMenuName;
        public string lpszClassName;
        public IntPtr hIconSm;
    }

    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    private struct NOTIFYICONDATAW
    {
        public uint cbSize;
        public IntPtr hWnd;
        public uint uID;
        public uint uFlags;
        public uint uCallbackMessage;
        public IntPtr hIcon;
        [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 128)]
        public string szTip;
        public uint dwState;
        public uint dwStateMask;
        [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 256)]
        public string szInfo;
        public uint uVersion;
        [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 64)]
        public string szInfoTitle;
        public uint dwInfoFlags;
        public Guid guidItem;
        public IntPtr hBalloonIcon;
    }

    [StructLayout(LayoutKind.Sequential)]
    private struct MENUITEMINFOW
    {
        public uint cbSize;
        public uint fMask;
        public uint fType;
        public uint fState;
        public uint wID;
        public IntPtr hSubMenu;
        public IntPtr hbmpChecked;
        public IntPtr hbmpUnchecked;
        public UIntPtr dwItemData;
        public IntPtr dwTypeData;
        public uint cch;
        public IntPtr hbmpItem;
    }

    [StructLayout(LayoutKind.Sequential)]
    private struct POINT
    {
        public int X;
        public int Y;
    }

#pragma warning restore CS0169, CS0649

    [DllImport("kernel32.dll", CharSet = CharSet.Unicode)]
    private static extern IntPtr GetModuleHandleW(string? name);

    [DllImport("user32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern ushort RegisterClassExW(ref WNDCLASSEXW type);

    [DllImport("user32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern IntPtr CreateWindowExW(
        uint exStyle, string className, string windowName, uint style,
        int x, int y, int width, int height, IntPtr parent, IntPtr menu, IntPtr instance, IntPtr param);

    [DllImport("user32.dll")]
    private static extern bool DestroyWindow(IntPtr window);

    [DllImport("user32.dll")]
    private static extern IntPtr DefWindowProcW(IntPtr window, uint message, IntPtr wParam, IntPtr lParam);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    private static extern uint RegisterWindowMessageW(string name);

    [DllImport("user32.dll")]
    private static extern bool PostMessageW(IntPtr window, uint message, IntPtr wParam, IntPtr lParam);

    [DllImport("user32.dll")]
    private static extern bool SetForegroundWindow(IntPtr window);

    [DllImport("user32.dll")]
    private static extern bool GetCursorPos(out POINT point);

    [DllImport("user32.dll")]
    private static extern int GetSystemMetrics(int index);

    [DllImport("user32.dll")]
    private static extern IntPtr CreateIconFromResourceEx(
        byte[] bits, uint size, bool isIcon, uint version, int width, int height, uint flags);

    [DllImport("user32.dll")]
    private static extern bool DestroyIcon(IntPtr icon);

    [DllImport("user32.dll")]
    private static extern IntPtr CreatePopupMenu();

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    private static extern bool AppendMenuW(IntPtr menu, uint flags, UIntPtr id, string? text);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    private static extern bool SetMenuItemInfoW(IntPtr menu, uint item, bool byPosition, ref MENUITEMINFOW info);

    [DllImport("user32.dll")]
    private static extern int TrackPopupMenuEx(IntPtr menu, uint flags, int x, int y, IntPtr window, IntPtr parameters);

    [DllImport("user32.dll")]
    private static extern bool DestroyMenu(IntPtr menu);

    [DllImport("shell32.dll", CharSet = CharSet.Unicode)]
    private static extern bool Shell_NotifyIconW(uint message, ref NOTIFYICONDATAW data);

    [DllImport("uxtheme.dll", EntryPoint = "#135")]
    private static extern int SetPreferredAppMode(int mode);

    [DllImport("uxtheme.dll", EntryPoint = "#133")]
    private static extern bool AllowDarkModeForWindow(IntPtr window, bool allow);

    [DllImport("uxtheme.dll", EntryPoint = "#136")]
    private static extern void FlushMenuThemes();
}
