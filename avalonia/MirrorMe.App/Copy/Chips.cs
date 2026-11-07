using MirrorMe.Api.V1;
using MirrorMe.App.Contracts;

namespace MirrorMe.App.Copy;

/// <summary>
/// Short form of each statement the form's per-settings diagnostics raise, as a card prints it.
/// The sentence (<see cref="Statements"/>) stands in the chip's tooltip.
/// A statement with no short form prints its code, as one with no sentence does (docs/tooltips.md).
/// </summary>
public static class Chips
{
    public static string Of(Text text)
    {
        Assert.NotNull(text, "a chip states one statement");

        var a = new Statements.Args(text);
        return text.Code switch
        {
            TextCode.DiscordNotLinked => "Discord not linked",
            TextCode.DiscordLinkRefused => "Discord link not recognized",
            TextCode.DiscordNoVoiceChannel => "Not in a voice channel",
            TextCode.GroupRequired => "Needs a group",
            TextCode.GroupNameMissing => "Needs a computer name",

            TextCode.NoUplinkStated => "Upload speed not set",
            TextCode.UplinkBelowPrediction => "More than the upload carries",
            TextCode.BurstAboveUplink => "Peaks above the upload",
            TextCode.CeilingHoldsQuality => "Held at the ceiling",

            TextCode.FpsAboveRefresh => "Repeats frames",
            TextCode.FpsMatchesRefresh => "Matches the screen",
            TextCode.EngineNotProbed => "Encoders not tested here",

            TextCode.DecodesOnCpu => "Every viewer decodes on CPU",
            TextCode.DecodesInHardwarePartly => "Some viewers decode on CPU",
            TextCode.DecodesInHardwareEverywhere => "Plays on every viewer",
            TextCode.EncodesOnGpu => "Runs on the graphics card",
            TextCode.JoinWait => $"Viewers join within {Statements.Seconds(a.JoinWaitMs)} s",
            TextCode.WatchDelay => a.DelayMs < 100 ? "Under 0.1 s behind" : $"~{Statements.Tenths(a.DelayMs)} s behind",

            TextCode.MonitorNotPriced => "Screen not connected",
            TextCode.NoPictureToPrice => "No prediction",

            _ => text.Code.ToString(),
        };
    }
}
