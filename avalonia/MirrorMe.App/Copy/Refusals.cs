using System.Globalization;
using MirrorMe.Api.V1;
using MirrorMe.App.Contracts;

namespace MirrorMe.App.Copy;

/// <summary>
/// Sentence for each refusal a served status carries (docs/ipc-api.md, "Errors").
/// The coded reason behind one follows it as a sentence of its own.
/// </summary>
public static class Refusals
{
    /// <summary>A refusal as one string: the statement, then a third party's words on a line of their own.</summary>
    public static string Of(Refusal refusal)
    {
        Assert.NotNull(refusal, "a refusal is read off a served status");

        var statement = Statements.Of(refusal.Text);
        return refusal.Message.Length == 0 ? statement : $"{statement}\n{refusal.Message}";
    }

    /// <summary>The sentence for a refusal code, null for every other code.</summary>
    internal static string? Of(Text text)
    {
        var a = new Statements.Args(text);
        var cause = Statements.Of(a.Cause);
        string Because(string lead) => cause.Length == 0 ? lead : $"{lead} {cause}";

        return text.Code switch
        {
            TextCode.RequestNotInThisBuild =>
                "This copy of MirrorMe does not support that request. Reinstall MirrorMe so all its parts share one version.",
            TextCode.FrameSourceUnnamed => "The picture request named nothing to draw.",
            TextCode.StreamUnnamed => "No stream was named.",
            TextCode.TransportUnnamed => $"No protocol was named for the stream {a.Stream}.",
            TextCode.SettingsAbsent => "The request carried no settings.",
            TextCode.PresetUnnamed => "The preset has no name. Name it to save it.",
            TextCode.ReadNamesNothing => "The read named nothing to read.",
            TextCode.KindNotReadable => $"{a.Kind} arrives as events, so it is watched and cannot be read.",
            TextCode.KindNotWatchable => $"{a.Kind} changes without events, so it is read and cannot be watched.",
            TextCode.KindUnknown =>
                $"This copy of MirrorMe has no event kind {a.Kind}. It carries {Words.List(a.Kinds)}.",
            TextCode.LinkEmpty => "The link is empty.",
            TextCode.TestStreamCountNegative => $"{Plain(a.Count)} test streams cannot run. Ask for zero or more.",
            TextCode.RunKeepsNoLog => "This run kept no log.",

            TextCode.TestStreamsAboveLimit =>
                $"This computer runs up to {Plain(a.CountLimit)} test streams at once. Ask for {Plain(a.CountLimit)} or fewer.",
            TextCode.LogGone => $"The log file is gone ({a.Path}). Older run logs are rotated away.",
            TextCode.ReceiveAudioOfNoDecode =>
                Because($"{Leg(a)} is not playing, so its sound cannot be set."),

            TextCode.FramesUndrawable => Because(Source(a)),
            TextCode.WatchRefused => Because($"Could not open {Leg(a)}."),
            TextCode.BrowserOpenRefused => Because($"Could not open {Leg(a)} in the browser."),
            TextCode.ReceiveRefused => Because($"Could not play {Leg(a)}."),
            TextCode.MonitorPreviewRefused => a.Cause?.Code == TextCode.MonitorNotEnumerated
                ? $"Screen {Plain(a.Monitor)} is not connected."
                : Because($"Could not preview screen {Plain(a.Monitor)}."),
            TextCode.ReportRefused => Because("Could not send the report."),
            TextCode.UpdateCheckRefused => Because("Could not check for updates."),
            TextCode.UpdateInstallRefused => Because("Could not install the update."),
            TextCode.UplinkMeasureWhileLive =>
                "Stop sharing to measure the upload speed. The stream would compete with the measurement for the line.",
            TextCode.EncodeRateMeasureWhileLive =>
                "Stop sharing to time the encoder. The stream would compete with the test for the encoder.",
            TextCode.ProtocolMajorMismatch =>
                $"The parts of MirrorMe come from different releases (protocol {Plain(a.AskedProtocolMajor)} "
                + $"and {Plain(a.ProtocolMajor)}). Reinstall MirrorMe.",
            TextCode.LinkUnresolved => Because("Could not open the link."),

            TextCode.SettingsUnsaved => Because("Could not save the settings."),
            TextCode.PresetUnsaved => Because($"Could not save the preset {a.Preset}."),
            TextCode.PresetsUnread => Because("Could not read the saved presets."),
            TextCode.PresetUndeleted => Because($"Could not delete the preset {a.Preset}."),
            TextCode.PublishUnstarted => Because("Could not start sharing."),
            TextCode.PublishUnapplied => Because("Could not apply the settings to the running stream."),
            TextCode.TestStreamsUnstarted => Because($"Could not start {Plain(a.Count)} test streams."),
            TextCode.PortalConsentKept => Because("Could not forget the screen-sharing permission."),
            TextCode.GroupKeyUndrawn => Because("Could not create a group."),
            TextCode.DiscordUnlinkFailed => Because("Could not unlink Discord."),
            TextCode.DiscordLinkFailed => Because("Could not link Discord."),
            TextCode.LogUnopened => Because($"Could not open the log {a.Path}."),
            TextCode.LogsFolderUnopened => Because("Could not open the logs folder."),
            TextCode.UplinkUnmeasured => Because("Could not measure the upload speed."),
            TextCode.EncodeRateUnmeasured => Because("Could not time the encoder."),

            _ => null,
        };
    }

    /// <summary>One stream on one protocol: "desk over SRT".</summary>
    private static string Leg(Statements.Args a) => $"{a.Stream} over {Words.Transport(a.Transport)}";

    /// <summary>What a frame stream draws from: a relay decode by its pair, a screen by its index, else the preview.</summary>
    private static string Source(Statements.Args a)
    {
        if (a.Stream.Length > 0)
        {
            return $"The picture of {Leg(a)} is not available yet.";
        }
        return a.NamesMonitor
            ? $"The picture of screen {Plain(a.Monitor)} is not available yet."
            : "The preview of the shared stream is not available yet.";
    }

    private static string Plain(long value) => value.ToString(CultureInfo.InvariantCulture);
}
