using static ScreenShare.App.Features.Shell.Settings.Model.AppSettingsCopy;

namespace ScreenShare.App.Features.Shell.Settings.Model;

/// <summary>
/// One section of the settings dialog, of which one is on screen at a time.
/// A press that comes to fix one thing opens at the section holding the control that fixes it.
/// </summary>
public enum SettingsSection
{
    Window,
    Updates,
    Discord,
    Logs,
    About,
    Development,
}

/// <summary>
/// One entry of the dialog's section list, and the question its section opens on.
/// SetApart: drawn under the others, past a divider.
/// </summary>
public sealed record SettingsTab(SettingsSection Section, string Label, string Question, string Lead, bool SetApart = false);

/// <summary>Sections in the order the list draws them, the first being where a plain open lands.</summary>
public static class SettingsSections
{
    public static readonly IReadOnlyList<SettingsTab> All =
    [
        new(SettingsSection.Window, WindowHeading, WindowQuestion, WindowLead),
        new(SettingsSection.Updates, UpdatesHeading, UpdatesQuestion, UpdatesLead),
        new(SettingsSection.Discord, DiscordHeading, DiscordQuestion, DiscordLead),
        new(SettingsSection.Logs, LogsHeading, LogsQuestion, LogsLead),
        new(SettingsSection.About, AboutHeading, AboutQuestion, AboutLead),
        new(SettingsSection.Development, DevelopmentHeading, DevelopmentQuestion, DevelopmentLead, SetApart: true),
    ];
}
