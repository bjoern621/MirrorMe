using MirrorMe.Api.V1;

namespace MirrorMe.App.Features.Shell.Settings.Model;

/// <summary>
/// What the settings dialog says in its own right, around the controls the fields own.
///
/// Keyed on nothing, like <see cref="Copy.Cards"/>: a control under a heading is keyed by field and takes
/// its words from <see cref="Copy.Fields"/>.
/// </summary>
public static class AppSettingsCopy
{
    public const string Title = "Settings";

    public const string CloseTip = "Closes this dialog. Every setting here is already kept.";

    public const string BackTip = "Back to the list of settings";

    public const string OpenTip =
        "Opens settings for the app as a whole. Set sharing and watching in Setup and Viewer.";

    public const string ResetLabel = "Reset to defaults";

    public const string ResetHelp =
        "Puts every setting here back to the value a fresh installation starts with.";

    public const string WindowHeading = "Window";

    public const string WindowQuestion = "What happens when you close the window?";

    public const string WindowLead = "With the tray icon on, sharing goes on after the window closes.";

    public const string LogsHeading = "Logs and reports";

    public const string LogsQuestion = "Something went wrong?";

    public const string LogsLead = "Send a report and quote its name when you describe the problem.";

    public const string OpenLogsFolder = "Open logs folder";

    public const string OpenLogsFolderTip = "Opens the folder holding the run logs in the file manager.";

    public const string SendReport = "Send a report";

    public const string SendReportTip =
        "Sends the newest run logs, this computer's details and your settings with every secret removed to the relay.";

    /// <summary>
    /// Carries the name the report was stored under, which is what a reader quotes.
    /// Selectable where it is drawn,
    /// for the reason an error text is (<c>CLAUDE.md</c>, "Every error message is selectable and copyable").
    /// </summary>
    public static string ReportSent(string id) => $"Report sent ({id}). Quote that name when you report the problem.";

    public const string UpdatesHeading = "Updates";

    public const string UpdatesQuestion = "Is this the newest version?";

    public const string UpdatesLead = "A check reads the published release and says whether a newer build exists.";

    public const string CheckNow = "Check for updates";

    public const string DiscordHeading = "Discord";

    public const string DiscordQuestion = "Show what you share on Discord?";

    public const string DiscordLead = "Your voice channel becomes the group, and your profile says what you share.";

    /// <summary>
    /// Press on a check line about the Discord link, which is fixed here rather than on a wizard step
    /// (<c>Features/Setup/Model/CheckAnchor.cs</c>).
    /// Composed from the two names it leads to, so a rename of either carries.
    /// </summary>
    public static readonly string ToDiscord = $"Go to {Title} · {DiscordHeading}";

    public const string DevelopmentHeading = "Development";

    public const string DevelopmentQuestion = "Testing MirrorMe itself?";

    public const string DevelopmentLead = "Synthetic streams give a group something to watch with no screen shared.";

    public const string AboutHeading = "About";

    public const string AboutQuestion = "Which build is this?";

    public const string AboutLead = "Quote this version in a bug report.";

    /// <summary>
    /// Where this install stands with Discord, in the sentence the relay step states it in as well
    /// (<see cref="Copy.Links.State"/>).
    /// </summary>
    public static string DiscordLine(DiscordState? state) => Copy.Links.State(state);
}
