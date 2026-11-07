using System.Windows.Input;

namespace MirrorMe.App.Features.Tray.View;

/// <summary>
/// One row of the tray menu as both renderers read it: Avalonia's <c>NativeMenu</c>
/// where the desktop draws the menu (DBus, macOS), and the Win32 popup menu on Windows.
///
/// A null <see cref="Label"/> is a separator.
/// A row with <see cref="Rows"/> opens a submenu and acts on nothing itself.
/// A row with a <see cref="Command"/> acts through it and is enabled while it can execute,
/// which a renderer reads when it draws, a command in flight moving without the menu being rebuilt.
/// </summary>
internal sealed record TrayRow(
    string? Label,
    bool IsEnabled = true,
    Action? Act = null,
    ICommand? Command = null,
    bool IsRadio = false,
    bool IsChecked = false,
    IReadOnlyList<TrayRow>? Rows = null)
{
    public static TrayRow Separator { get; } = new((string?)null);

    public bool IsSeparator => Label is null;

    /// <summary>Whether the row can be picked right now.</summary>
    public bool CanAct => IsEnabled && (Command?.CanExecute(null) ?? true);

    /// <summary>What picking the row does, the action and the command both where a row carries both.</summary>
    public void Invoke()
    {
        Act?.Invoke();
        if (Command?.CanExecute(null) == true)
        {
            Command.Execute(null);
        }
    }
}
