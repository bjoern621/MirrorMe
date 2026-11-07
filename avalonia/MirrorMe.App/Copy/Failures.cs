using System.Globalization;
using MirrorMe.Api.V1;

namespace MirrorMe.App.Copy;

/// <summary>
/// Sentence for each failure inside the app that a refusal or a state names as its cause
/// (api/proto/mirrorme/v1/text.proto, "Failures inside the app").
/// </summary>
public static class Failures
{
    /// <summary>A failure as one string: the statement, then a third party's words on a line of their own.</summary>
    public static string Of(Text? statement, string words)
    {
        var sentence = Statements.Of(statement);
        if (words.Length == 0)
        {
            return sentence;
        }
        return sentence.Length == 0 ? words : $"{sentence}\n{words}";
    }

    /// <summary>The sentence for a failure code, then the sentence for its cause; null for every other code.</summary>
    internal static string? Of(Text text)
    {
        var a = new Statements.Args(text);
        var sentence = Sentence(text, a);
        var cause = Statements.Of(a.Cause);
        return sentence is null || cause.Length == 0 ? sentence : $"{sentence} {cause}";
    }

    private static string? Sentence(Text text, Statements.Args a)
    {
        return text.Code switch
        {
            TextCode.IdentifierUnknown =>
                $"This copy of MirrorMe does not know \"{FirstId(text)}\". The settings may come from a newer release.",
            TextCode.EngineHasNoForm =>
                $"{Words.Engine(a.Engine)} cannot handle \"{FirstId(text, TextArgName.Engine)}\". Change the capture method to reach the other engine.",
            TextCode.FpsNotPositive => $"The frame rate is {Plain(a.Fps)}. Set it above zero.",
            TextCode.ProgramMissing => $"{a.Program} is not installed. Reinstall MirrorMe to bring it back.",
            TextCode.ProgramUnstarted => $"{a.Program} did not start.",
            TextCode.ProgramExited => $"{a.Program} stopped unexpectedly.",
            TextCode.FileUnwritable => $"Could not write {a.Path}.",
            TextCode.FileUnreadable => $"Could not read {a.Path}.",
            TextCode.FileCorrupt => $"The file {a.Path} is damaged and cannot be read.",
            TextCode.DisplayUnset =>
                $"{Words.Capture(a.Capture)} needs an X display, and none is set (DISPLAY).",
            TextCode.NoRenderNode =>
                $"This computer has no GPU that can take the frames (no render node under /dev/dri). "
                + $"Set frame memory to \"{Words.Memory(MemorySystem)}\".",
            TextCode.NoScanoutCard => "No graphics card here drives a screen, so there is nothing to capture.",
            TextCode.LoopbackPortUnavailable => "Could not open a local port on this computer.",
            TextCode.AddressUnanswered => $"No answer from {a.Address}.",
            TextCode.AddressAnswered => $"{a.Address} turned the request down.",
            TextCode.AddressAnswerUnreadable => $"Could not read the answer from {a.Address}.",
            TextCode.AddressUnreadable => $"{a.Address} is not an address that can be reached.",
            TextCode.PlaylistNoVideo => $"The HLS playlist at {a.Address} lists no video.",
            TextCode.PlaylistNoSegment =>
                $"The HLS playlist at {a.Address} offered no video within {Statements.Seconds(a.TimeoutMs)} seconds.",
            TextCode.EncoderPropertyBound =>
                $"The {Words.Family(a.Family)} encoder's {a.Property} setting stops at {Plain(a.CountLimit)}, "
                + $"and these settings ask for {Plain(a.Count)}. Lower the rate.",
            TextCode.VaVbrTargetBelowFloor =>
                $"The VA-API encoder needs the target to be at least {Plain(a.CountLimit)}% of the ceiling, "
                + $"and {Plain(a.BitrateMbps)} Mbit/s under {Plain(a.MaxrateMbps)} Mbit/s is lower. "
                + "Raise the target or lower the ceiling.",
            TextCode.StreamIdTooLong =>
                $"The SRT stream ID is {Plain(a.Count)} bytes, and SRT carries {Plain(a.CountLimit)} at most. "
                + "Shorten the stream name.",
            TextCode.PortalUnanswered => "The screen-sharing portal did not hand over a stream.",
            TextCode.PortalDismissed => "The screen picker was closed without a choice. Start again to pick what to share.",
            TextCode.PortalCallFailed => $"The screen-sharing portal turned down {a.Value}.",
            TextCode.PortalNoSourceKind => "The screen-sharing portal offers nothing to share.",
            TextCode.SessionBusUnreachable => "Could not reach the desktop session bus (D-Bus).",
            TextCode.HdrCaptureEightBit =>
                $"The capture is HDR ({Words.Transfer(a.Transfer)}), and {Words.Chroma(a.Chroma)} carries 8 bits. "
                + $"Encode as {Words.Chroma(a.Value)}, or capture in standard range.",
            TextCode.DrmMapUnknown => $"KMSGRAB_DRM_MAP is \"{a.Value}\", which names no download method.",
            TextCode.AudioDeviceUnnamed => $"The {Words.AudioSource(a.Audio)} source names no device. Pick one.",
            TextCode.PipelineUnreachable => "The running stream did not take the change.",
            TextCode.PreviewHasNoLeg =>
                $"{a.Codec} produces {Words.Format(a.Format)}, which the local preview cannot show.",
            TextCode.ConfigDirUnknown => "This system names no folder for app settings.",
            TextCode.CaptureRecordsNoAudio =>
                $"{Words.Capture(a.Capture)} cannot record {Words.AudioSource(a.Audio)}.",
            TextCode.NoElementRecordsAudio =>
                $"Nothing on {Words.OperatingSystem(a.Os)} records {Words.AudioSource(a.Audio)} here.",
            TextCode.TeeSlaveUnspellable => a.Option.Length > 0
                ? $"The value of {a.Option} holds a character the output cannot carry (| [ or ])."
                : "The stream address holds a character the output cannot carry (| [ or ]).",
            TextCode.FrameMemoryNoDevicePath =>
                $"{Words.Capture(a.Capture)} and the {Words.Family(a.Family)} encoder share no GPU memory "
                + $"on {Words.Engine(a.Engine)}. Set frame memory to \"{Words.Memory(MemorySystem)}\".",
            TextCode.FrameMemoryTradesColour =>
                $"{Words.Capture(a.Capture)} reaches the {Words.Family(a.Family)} encoder on the GPU only "
                + $"with the encoder converting the color. Pick \"{Words.Memory(MemoryGpuEncoderColor)}\" "
                + $"to accept that, or \"{Words.Memory(MemorySystem)}\" to keep the color.",
            TextCode.GpuUndetermined =>
                $"This computer has several GPUs ({Words.List(a.Devices)}), and the portal does not say "
                + $"which one captures. Set frame memory to \"{Words.Memory(MemorySystem)}\".",
            TextCode.RelayUnnamed => "No relay is set. Enter one under Relay.",
            TextCode.GroupServiceUnnamed => "This relay offers no group service, so a group key cannot be created here.",
            TextCode.RelayTokenUndrawn => "Could not get access to the relay for this group.",
            TextCode.RandomUnavailable => "This system's random number source did not answer.",
            TextCode.DecodeHostUnanswered => "The video decoder process did not answer.",
            TextCode.DecodeHostDown => "The video decoder process is not running.",
            TextCode.ReceivePipelineUnbuilt => $"Could not set up playback of {a.Stream}.",
            TextCode.FramesNeedPlayer =>
                $"MirrorMe cannot show video in its own window on {Words.OperatingSystem(a.Os)}. "
                + "Watch this stream in a player.",
            TextCode.FramesFormatUnimportable => $"The video arrived as {a.Format}, which the window cannot show.",
            TextCode.FramesNoDmabuf => "The GPU did not share the decoded frames with the window.",
            TextCode.FramesConsumerGone => "The window stopped reading the video.",
            TextCode.FramesUnshared => "Could not share the decoded frames with the window.",
            TextCode.LinkUnreadable => "This is not a MirrorMe link.",
            TextCode.LinkNotOurs => $"MirrorMe does not open links that start with \"{a.Value}\".",
            TextCode.LinkIncomplete => "The link names no stream.",
            TextCode.StreamNameTooDeep => $"The stream name {a.Stream} has too many slashes.",
            TextCode.GroupKeyUnreadable => "MirrorMe cannot read this group key. Paste it again.",
            TextCode.MemberSecretUnreadable =>
                "This computer's membership in the group is damaged. Leave the group and join again.",
            TextCode.UpdateNotStaged => "No update is downloaded yet.",
            TextCode.ArchiveUnreadable => $"{a.Path} is not an archive MirrorMe unpacks.",
            TextCode.EncodeProbeTimedOut =>
                $"The encoder test ran past {Statements.Seconds(a.TimeoutMs)} seconds. "
                + "These settings encode slower than the test allows.",
            TextCode.MeasureHasNoTime => "The measurement finished too fast to time.",
            TextCode.GroupMoved => "The voice channel's group changed. Try again in a moment.",
            TextCode.DiscordLinkTimedOut =>
                $"Nothing came back from the browser within {Plain(a.TimeoutMs / 60_000)} minutes. Start linking again.",
            TextCode.BrowserUnopened => "The browser did not open.",
            TextCode.AlreadyPublishing => "Already sharing. Stop the stream to start it with other settings.",
            TextCode.NothingPublishing => "Nothing is being shared.",
            TextCode.NothingDecoding => $"{a.Stream} over {Words.Transport(a.Transport)} is not playing.",
            TextCode.MonitorNotPreviewing => $"Screen {Plain(a.Monitor)} has no preview running.",
            TextCode.PreviewNotRunning => "The stream has no local preview running.",
            TextCode.LinkOtherVoiceChannel =>
                "This stream is shared in another voice channel. Join that channel in Discord to watch it.",
            TextCode.LinkOtherGroup =>
                "This stream is in another group. Paste that group's key under Relay to watch it.",
            TextCode.StreamNotOnRelay =>
                "Nobody is sharing that stream right now. Ask whoever sent the link to start sharing again.",
            TextCode.OutputSizeUnreadable => $"The output resolution \"{a.Value}\" is not in WIDTHxHEIGHT form.",
            TextCode.OutputSizeBelowFloor =>
                $"The output resolution {a.Value} is below {Plain(a.CountLimit)} pixels on a side.",
            TextCode.OutputSizeOdd =>
                $"The output resolution {a.Value} is odd on one side. Pick an even width and height.",
            _ => null,
        };
    }

    // Frame memory values as the settings spell them (backend/internal/gpupath).
    private const string MemorySystem = "system";
    private const string MemoryGpuEncoderColor = "gpu-encoder-color";

    /// <summary>
    /// The identifier a statement carries on whichever axis it names, past the ones listed:
    /// an unknown identifier names its axis by the argument it rides on.
    /// </summary>
    private static string FirstId(Text text, params TextArgName[] past)
    {
        foreach (var arg in text.Args)
        {
            if (!past.Contains(arg.Name) && arg.Id.Length > 0)
            {
                return arg.Id;
            }
        }
        return "";
    }

    private static string Plain(long value) => value.ToString(CultureInfo.InvariantCulture);

    private static string Plain(double value) => value.ToString("0.#", CultureInfo.InvariantCulture);
}
