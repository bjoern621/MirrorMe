namespace ScreenShare.App.Copy;

/// <summary>Words of the insights screen that no backend statement carries (<c>docs/insights.md</c>).</summary>
public static class InsightsCopy
{
    public const string NotSharing = "Not sharing";

    public const string NotSharingDetail = "Start sharing to see how the stream holds up.";

    public const string RunningClean = "Running clean";

    public const string RunningCleanDetail = "The stream holds the frame rate it asks for.";

    public const string UnderTarget = "Frame rate under target";

    public const string Stopped = "Stream stopped";

    public const string OpenFullLog = "Open full log";

    public const string StreamTab = "Stream";

    public const string NoReadings = "Nothing to chart yet. The encoder reports once a second while sharing.";

    public const string ViewersNote = "Latency and frame rate come from each viewer's own app. A browser or an external player sends neither.";

    /// <summary>Card caption for a bitrate held to nothing.</summary>
    public const string NoCap = "no cap";

    public const string YourGroup = "your group";

    public static string Restarting(int attempt, int budget)
        => attempt > 0 && budget > 0 ? $"Restarting, attempt {attempt} of {budget}" : "Restarting";

    public static string ViewersTab(int? count) => count is > 0 ? $"Viewers {count}" : "Viewers";

    /// <summary>Header: what is shared and with whom, "Screen 1 to Friday games".</summary>
    public static string Title(string target, string group) => $"{target} to {group}";
}
