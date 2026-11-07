namespace MirrorMe.App.Features.Shell.Settings.Model;

/// <summary>
/// Which of the section list and the section the dialog draws.
/// A wide dialog draws both, and a compact one draws the list until a section is picked from it.
/// </summary>
/// <param name="Compact">Whether the window is too narrow for the list beside a section.</param>
/// <param name="AtList">Whether a compact dialog stands on the list. Ignored while wide.</param>
public readonly record struct SettingsPlace(bool Compact, bool AtList)
{
    public bool ShowsList => !Compact || AtList;

    public bool ShowsSection => !Compact || !AtList;

    public bool ShowsBack => Compact && !AtList;
}
