using Avalonia;
using Avalonia.Controls;
using Avalonia.Platform;
using Avalonia.Threading;
using MirrorMe.App.Contracts;
using MirrorMe.App.Features.Tray.Model;
using MirrorMe.App.Features.Tray.ViewModel;

namespace MirrorMe.App.Features.Tray.View;

/// <summary>
/// The tray icon as the platform draws it, rendered whole from <see cref="TrayViewModel.Menu"/>.
/// Code-behind for the reason the pop-out pass is: nothing binds a tray icon into existence.
///
/// <b>The icon comes and goes with the setting.</b> <see cref="Render"/> registers one where
/// <see cref="IsWanted"/> asks for it and takes it out where it does not, so a toggle in the settings
/// dialog lands without a restart. Both directions are idempotent, a second pass over an unchanged menu
/// registering and removing nothing.
///
/// No icon stands until the settings have been read, an icon that flashes up and goes being worse than
/// one arriving late.
///
/// A platform serving no tray is an Umgebungsfehler,
/// and leaves <see cref="IsUp"/> false the way the setting does:
/// quit-on-close stands, a hidden window with no icon to come back through being gone.
/// </summary>
public sealed class TrayIconHost : IDisposable
{
    /// <summary>
    /// <c>0</c> keeps the icon out of the tray for a whole run, whatever the setting holds.
    ///
    /// For a desktop whose panel draws no tray, and for a checkout run beside an installed app.
    /// Everything the menu offers is the window's own, so a run without the icon loses no control.
    ///
    /// The backend reads it too and greys the app setting behind it
    /// (<c>internal/control</c>, <c>EnvTray</c>).
    /// </summary>
    internal const string EnvTray = "MIRRORME_TRAY";

    private readonly TrayViewModel _tray;
    private readonly string? _env;
    private readonly WindowIcon _idle;
    private readonly WindowIcon _live;
    private readonly byte[] _idlePng;
    private readonly byte[] _livePng;

    /// <summary>The registered icon while one stands, null while none does. Every platform but Windows.</summary>
    private TrayIcon? _icon;

    /// <summary>
    /// The registered icon on Windows, whose menu Avalonia would draw in the app's own dark theme
    /// rather than the system's (<see cref="Win32TrayIcon"/>).
    /// </summary>
    private Win32TrayIcon? _native;

    /// <summary>Which image <see cref="_native"/> shows, so an unchanged state swaps nothing.</summary>
    private bool _nativeLive;

    private TrayIconHost(TrayViewModel tray)
    {
        _tray = tray;
        _env = Environment.GetEnvironmentVariable(EnvTray);
        _idlePng = Asset("avares://mirrorme/Assets/tray.png");
        _livePng = Asset("avares://mirrorme/Assets/tray-live.png");
        _idle = new WindowIcon(new MemoryStream(_idlePng));
        _live = new WindowIcon(new MemoryStream(_livePng));

        // Disposing an icon takes down the process it is registered against (AvaloniaUI/Avalonia#21979):
        // the DBus watch loop's cancellation escapes an async void into the dispatcher.
        // Marked handled here, at the host owning the icon whose dispose arms it, until upstream catches it.
        Dispatcher.UIThread.UnhandledException += (_, thrown) =>
        {
            if (IsWatchDisposeCancellation(thrown.Exception))
            {
                thrown.Handled = true;
            }
        };

        // The menu compares by content, so each notification is a real change and each change one rebuild.
        tray.PropertyChanged += (_, changed) =>
        {
            if (changed.PropertyName == nameof(TrayViewModel.Menu))
            {
                Render();
            }
        };

        Render();
    }

    /// <summary>Builds the host. The icon itself follows the menu.</summary>
    public static TrayIconHost Create(TrayViewModel tray)
    {
        Assert.NotNull(tray, "a tray icon draws the tray's state");

        return new TrayIconHost(tray);
    }

    /// <summary>
    /// Whether an icon stands in the tray right now, which is what makes a window close a hide.
    /// </summary>
    public bool IsUp => _icon is not null || _native is not null;

    /// <summary>Takes the icon out of the tray. A disposed host stays disposed.</summary>
    public void Dispose() => Remove();

    /// <summary>
    /// Whether a run wants an icon.
    /// <see cref="EnvTray"/> refuses one at <c>0</c>, and the app setting decides everywhere else.
    /// </summary>
    internal static bool IsWanted(string? environment, bool wanted) => environment != "0" && wanted;

    /// <summary>
    /// Whether a dispatcher exception is the tray watch dying of its own disposal,
    /// matched by type and by the frame that threw it.
    /// Every other exception passes through and crashes, as a broken contract should.
    /// </summary>
    internal static bool IsWatchDisposeCancellation(Exception thrown) =>
        thrown is OperationCanceledException
        && thrown.StackTrace?.Contains("DBusTrayIconImpl.WatchAsync") == true;

    /// <summary>The one render function: whether an icon stands, and what it draws, from the menu state.</summary>
    private void Render()
    {
        var menu = _tray.Menu;

        if (!IsWanted(_env, menu.IsIconWanted))
        {
            Remove();
            return;
        }

        Register();

        // The Windows menu reads the rows when it opens, so only the image follows the state here.
        if (_native is not null)
        {
            if (_nativeLive != menu.IsLive)
            {
                _nativeLive = menu.IsLive;
                _native.SetIcon(menu.IsLive ? _livePng : _idlePng);
            }

            return;
        }

        if (_icon is null)
        {
            return;
        }

        _icon.Icon = menu.IsLive ? _live : _idle;
        _icon.Menu = MenuOf(RowsOf(menu));
    }

    /// <summary>Puts an icon in the tray, where none stands. A platform serving no tray leaves none.</summary>
    private void Register()
    {
        if (_icon is not null || _native is not null)
        {
            return;
        }

        if (OperatingSystem.IsWindows())
        {
            try
            {
                _native = Win32TrayIcon.Create(TrayCopy.Tip, _idlePng, _tray.Open, () => RowsOf(_tray.Menu));
                _nativeLive = false;
            }
            catch (Exception)
            {
                // Umgebungsfehler: a session with no window station to own an icon.
            }

            return;
        }

        try
        {
            var icon = new TrayIcon { ToolTipText = TrayCopy.Tip, Icon = _idle };

            // Left-click brings the window back; the menu is the platform's right-click.
            icon.Clicked += (_, _) => _tray.Open();

            // Registered against the application, which is what holds the platform icon alive.
            TrayIcon.SetIcons(Application.Current!, new TrayIcons { icon });
            _icon = icon;
        }
        catch (Exception)
        {
            // Umgebungsfehler: a platform or a session with no tray to register against.
        }
    }

    /// <summary>
    /// Takes the icon out of the tray, where one stands.
    /// The empty set and the dispose both: which of the two drops the platform icon is Avalonia's business,
    /// and a second dispose is a no-op.
    /// </summary>
    private void Remove()
    {
        if (_native is not null)
        {
            var native = _native;
            _native = null;
            native.Dispose();
        }

        if (_icon is null)
        {
            return;
        }

        var icon = _icon;
        _icon = null;

        TrayIcon.SetIcons(Application.Current!, new TrayIcons());
        icon.Dispose();
    }

    private static byte[] Asset(string uri)
    {
        using var stream = AssetLoader.Open(new Uri(uri));
        using var copy = new MemoryStream();
        stream.CopyTo(copy);
        return copy.ToArray();
    }

    /// <summary>The menu's rows from its state, which both renderers draw.</summary>
    private List<TrayRow> RowsOf(TrayMenu menu)
    {
        var rows = new List<TrayRow>();

        // A line to read, inert: the desktop draws a disabled row as text.
        if (menu.State.Length > 0)
        {
            rows.Add(new TrayRow(menu.State, IsEnabled: false));
            rows.Add(TrayRow.Separator);
        }

        rows.Add(new TrayRow(menu.CommitLabel, IsEnabled: menu.CanCommit, Act: _tray.Commit));

        if (menu.Presets.Count > 0)
        {
            rows.Add(new TrayRow(
                menu.IsLive ? TrayCopy.SwitchPreset : TrayCopy.Presets,
                Rows: PresetRowsOf(menu.Presets)));
        }

        rows.Add(TrayRow.Separator);
        rows.Add(new TrayRow(TrayCopy.Open, Act: _tray.Open));
        rows.Add(new TrayRow(TrayCopy.Quit, Command: _tray.QuitCommand));

        return rows;
    }

    /// <summary>A radio row per preset, the built-in half parted from the saved one as the card parts them.</summary>
    private List<TrayRow> PresetRowsOf(IReadOnlyList<TrayPresetEntry> entries)
    {
        var rows = new List<TrayRow>();

        for (var at = 0; at < entries.Count; at++)
        {
            var entry = entries[at];

            if (at > 0 && entries[at - 1].Kind != entry.Kind)
            {
                rows.Add(TrayRow.Separator);
            }

            rows.Add(new TrayRow(
                entry.Name,
                IsEnabled: entry.IsReachable,
                Act: () => _tray.UsePreset(entry),
                IsRadio: true,
                IsChecked: entry.IsCurrent));
        }

        return rows;
    }

    /// <summary>The rows as Avalonia's menu, which the desktop draws where it serves a tray menu of its own.</summary>
    private static NativeMenu MenuOf(IReadOnlyList<TrayRow> rows)
    {
        var built = new NativeMenu();

        foreach (var row in rows)
        {
            if (row.IsSeparator)
            {
                built.Items.Add(new NativeMenuItemSeparator());
                continue;
            }

            var item = new NativeMenuItem(row.Label!) { IsEnabled = row.IsEnabled, Command = row.Command };

            if (row.Rows is { } children)
            {
                item.Menu = MenuOf(children);
            }

            if (row.IsRadio)
            {
                item.ToggleType = MenuItemToggleType.Radio;
                item.IsChecked = row.IsChecked;
            }

            if (row.Act is { } act)
            {
                item.Click += (_, _) => act();
            }

            built.Items.Add(item);
        }

        return built;
    }
}
