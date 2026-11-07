namespace MirrorMe.App.Features.Tray.Model;

/// <summary>
/// The tray's fixed words.
/// Keyed on nothing the backend sends, so it sits with the feature rather than in <c>Copy/</c>,
/// the layering <c>Features/Setup/Model/CommitCopy.cs</c> states.
/// The commit row's words are elsewhere on purpose: the start label is <c>CommitCopy</c>'s and the stop
/// label the insights screen's, one place per word.
/// </summary>
public static class TrayCopy
{
    /// <summary>What hovering the icon says. The name alone: the icon itself carries the live state.</summary>
    public const string Tip = "MirrorMe";

    /// <summary>The submenu's heading while nothing is shared.</summary>
    public const string Presets = "Presets";

    /// <summary>The submenu's heading while sharing, a pick there restarting the stream on the preset.</summary>
    public const string SwitchPreset = "Switch preset";

    /// <summary>
    /// The line the menu opens on while sharing: <c>Sharing · 3 watching</c>.
    /// No running time: a figure moving every second would rebuild the desktop's menu under the pointer.
    /// </summary>
    public static string State(int? viewers) => viewers is > 0 ? $"Sharing · {viewers} watching" : "Sharing";

    public const string Open = "Open MirrorMe";

    /// <summary>The one full-shutdown path: window, tray and the backend this shell started.</summary>
    public const string Quit = "Quit MirrorMe";
}
