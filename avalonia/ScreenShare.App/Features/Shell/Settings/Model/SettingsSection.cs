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

/// <summary>One entry of the dialog's section list.</summary>
public sealed record SettingsTab(SettingsSection Section, string Label);

/// <summary>Sections in the order the list draws them, the first being where a plain open lands.</summary>
public static class SettingsSections
{
    public static readonly IReadOnlyList<SettingsTab> All =
    [
        new(SettingsSection.Window, AppSettingsCopy.WindowHeading),
        new(SettingsSection.Updates, AppSettingsCopy.UpdatesHeading),
        new(SettingsSection.Discord, AppSettingsCopy.DiscordHeading),
        new(SettingsSection.Logs, AppSettingsCopy.LogsHeading),
        new(SettingsSection.About, AppSettingsCopy.AboutHeading),
        new(SettingsSection.Development, AppSettingsCopy.DevelopmentHeading),
    ];
}
