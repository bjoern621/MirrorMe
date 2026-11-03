using System.Collections.ObjectModel;
using ScreenShare.Api.V1;
using ScreenShare.App.Contracts;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Insights.Model;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Viewer.Tile.Model;

/// <summary>
/// The panel, composed from one sample of a decode and the report of the window drawing it.
///
/// <see cref="Of"/> holds nothing: a panel is the whole of what two readings say, built again per pass,
/// so no figure is accumulated here and none can disagree with the backend.
/// What a tile keeps between passes is <see cref="Merge"/>'s, a row's last measurement and not a figure of its own.
///
/// Blocks run in the order the frames do, the order a reader walks when a tile looks wrong:
/// the first stage that reads badly is the one to act on and everything after it is downstream.
///
/// Nothing is computed here that the backend measured.
/// The rates are the sample's, taken against the backend's own interval,
/// so two windows on one decode read one figure, and a rate is absent rather than zero on the first sample of a run
/// (<c>api/proto/screenshare/v1/events.proto</c>, ReceiveStreamStats).
/// Added here is the window's own block, the one thing a backend cannot see.
/// </summary>
public static class TileStats
{
    /// <summary>Row that a closed block prints beside its heading, keyed on the heading's id or the element's factory.</summary>
    private static readonly Dictionary<string, string> Leads = new()
    {
        ["section.stream"] = "video_mbps",
        ["section.picture"] = "picture_size",
        ["section.decode"] = "decoder",
        ["section.render"] = "render_fps",
        ["section.delay"] = "delay.total",
        ["section.clock"] = "uptime_sec",
        ["section.audio"] = "audio_kbps",
        ["section.window"] = "window.dropped",
        ["srtsrc"] = "packets-received-lost",
        ["rtpjitterbuffer"] = "num-lost",
    };

    /// <summary>Rows where any reading above zero means lost picture or a slow stage, so the row stands out.</summary>
    private static readonly HashSet<string> Alarms =
    [
        "discarded_fps", "sink_dropped", "window.dropped",
        "packets-received-lost", "packets-received-dropped", "num-lost", "num-late",
    ];

    /// <summary>
    /// One tile's panel.
    /// <paramref name="sample"/> is null on an unsampled decode and on a tile drawing no relay decode, both
    /// leaving the window's own block alone, whose counters hold whatever produces the frames.
    /// </summary>
    public static IReadOnlyList<StatSection> Of(ReceiveStreamStats? sample, TileReport report)
    {
        var sections = new List<StatSection>(8);

        if (sample is not { } s)
        {
            Add(sections, "section.window", Window(report));
            return sections;
        }

        Add(sections, "section.stream", Arriving(s));
        Add(sections, "section.picture", Picture(s));
        Add(sections, "section.decode", Decode(s));
        Add(sections, "section.render", Render(s));
        Add(sections, "section.delay", Delay(s));
        Add(sections, "section.clock", Clock(s));
        Add(sections, "section.audio", Audio(s));
        Add(sections, "section.window", Window(report));

        // Last, being the deepest figures and the only ones depending on the leg the decode was opened on.
        foreach (var group in s.Groups)
        {
            sections.Add(Transport(group));
        }

        Assert.That(sections.Count > 0, "a panel prints at least what this window knows");
        return sections;
    }


    /// <summary>
    /// Converges the panel a tile holds onto the one its newest readings describe.
    /// Readings get written into the rows that are there, so a row under the pointer keeps its tooltip.
    ///
    /// The headings and labels decide between writing and replacing.
    /// They move when a pipeline negotiates something new, such as an audio branch coming up
    /// or a transport element starting to count.
    /// A changed heading list replaces the panel, a changed label list replaces that block.
    ///
    /// A row that the newest reading measured nothing for keeps the last measurement.
    /// A second is short enough for a healthy decode to measure none of a per-interval figure,
    /// and a column flipping between a number and an ellipsis cannot be read.
    /// A held value ends once the shape moves, as it does when a decode stops or gets rebuilt.
    /// </summary>
    public static void Merge(ObservableCollection<StatSection> held, IReadOnlyList<StatSection> built)
    {
        Assert.NotNull(held, "a merge converges the panel a tile is holding");
        Assert.NotNull(built, "a merge converges onto the panel the readings describe");

        if (!held.Select(section => section.Heading).SequenceEqual(built.Select(section => section.Heading)))
        {
            held.Clear();
            foreach (var section in built)
            {
                held.Add(section);
            }

            return;
        }

        for (var i = 0; i < held.Count; i++)
        {
            if (!held[i].Lines.Select(line => line.Label).SequenceEqual(built[i].Lines.Select(line => line.Label)))
            {
                held[i] = built[i];
                continue;
            }

            MergeLines(held[i].Lines, built[i].Lines);
        }

        Assert.That(held.Count == built.Count, "a block per block the readings describe", held.Count, built.Count);
    }

    private static void MergeLines(IReadOnlyList<StatLine> held, IReadOnlyList<StatLine> built)
    {
        for (var i = 0; i < held.Count; i++)
        {
            if (built[i].Value != Figure.NoValue)
            {
                held[i].Value = built[i].Value;
                held[i].Alarm = built[i].Alarm;
            }
        }
    }

    /// <summary>
    /// Adds one block, and nothing where the stage reported no figure at all.
    /// A heading over an empty block would name a stage and say nothing about it,
    /// so a stream without a track has no audio block.
    /// </summary>
    private static void Add(List<StatSection> sections, string id, IReadOnlyList<StatLine> lines)
    {
        if (lines.Count == 0)
        {
            return;
        }

        var heading = Counters.Heading(id);
        sections.Add(new StatSection(heading.Label, heading.Tip, lines, Leads.GetValueOrDefault(id, "")));
    }

    private static IReadOnlyList<StatLine> Arriving(ReceiveStreamStats s) =>
    [
        Line("codec_description", s.CodecDescription),
        Line("profile", s.Profile),
        Line("level", s.Level),
        Line("video_mbps", s.HasVideoMbps ? $"{s.VideoMbps:0.00} Mb/s" : ""),
        Line("video_fps", s.HasVideoFps ? $"{s.VideoFps:0.0} /s" : ""),
        Line("declared_fps", Declared(s.FpsNum, s.FpsDen)),
        Line("keyframes", Count(s.Keyframes)),
        Line("since_keyframe_sec", s.HasSinceKeyframeSec ? $"{s.SinceKeyframeSec:0.0} s ago" : ""),
        Line("video_bytes", Bytes(s.VideoBytes)),
    ];

    private static IReadOnlyList<StatLine> Picture(ReceiveStreamStats s) =>
    [
        Line("picture_size", Pixels(s.Width, s.Height)),
        Line("pixel_format", s.PixelFormat),
        Line("depth", s.Depth > 0 ? $"{s.Depth} bits" : ""),
        Line("subsampling", s.Subsampling),
        Line("colorimetry", s.Colorimetry),
        Line("transfer", s.Transfer.Length > 0 ? Words.Transfer(s.Transfer) : ""),
        Line("chroma_site", s.ChromaSite),
        Line("pixel_aspect", s.PixelAspect),
        Line("interlace", s.Interlace),
    ];

    private static IReadOnlyList<StatLine> Decode(ReceiveStreamStats s) =>
    [
        Line("decoder", s.Decoder.Length > 0
            ? $"{s.Decoder}, {(s.Hardware ? "on the GPU" : "on the CPU")}"
            : ""),
        Line("decode_memory", s.DecodeMemory.Length > 0 ? Words.FrameMemory(s.DecodeMemory) : ""),

        // In the block naming the stage that does it, ahead of Render,
        // so the panel reads as the funnel it is: arriving, shed here, drawn.
        Line("discarded_fps", s.HasDiscardedFps ? $"{s.DiscardedFps:0.0} /s" : "", s.HasDiscardedFps ? s.DiscardedFps : 0),

        // What the pipeline was built with, never what was asked for.
        // Printed in both directions: a row that vanished on "off" would leave two tiles of one HDR stream
        // indistinguishable.
        Line("tone_map", s.ToneMap ? "on" : "off"),
    ];

    private static IReadOnlyList<StatLine> Render(ReceiveStreamStats s) =>
    [
        Line("chain", s.Chain.Length > 0 ? Words.RenderChain(s.Chain) : ""),
        Line("render_memory", s.RenderMemory.Length > 0 ? Words.FrameMemory(s.RenderMemory) : ""),
        Line("render_format", Joined(s.RenderFormat, s.RenderColorimetry)),
        Line("render_size", Pixels(s.RenderWidth, s.RenderHeight)),
        Line("render_fps", s.HasRenderFps ? $"{s.RenderFps:0.0} /s" : ""),
        Line("rendered", Count(s.Rendered)),
        Line("sink_dropped", Count(s.Dropped), s.Dropped),
    ];

    /// <summary>
    /// How the decode is paced and how far it has got.
    /// The window the pacing is scheduled inside is a figure of the path and sits with the path
    /// (<see cref="Delay"/>).
    /// </summary>
    private static IReadOnlyList<StatLine> Clock(ReceiveStreamStats s) =>
    [
        Line("live", s.Live ? "yes" : "no"),
        Line("position_sec", s.HasPositionSec ? Elapsed(s.PositionSec) : ""),
        Line("uptime_sec", Elapsed(s.UptimeSec)),
    ];

    /// <summary>
    /// What the path cost a frame: the whole of it first,
    /// then the stages behind it in the order the frame crossed them.
    ///
    /// Which stages carry a figure is the backend's answer and not this side's:
    /// a viewer measures its own leg and its own pipeline,
    /// and sees the publishing machine's own work only where it is that machine too
    /// (<c>api/proto/screenshare/v1/events.proto</c>, DelayBudget).
    ///
    /// One row is not a stage: the latency window, sitting last under the three rows that fill it.
    ///
    /// A decode reporting no budget at all is drawn as one that measured nothing,
    /// the window being the sample's own figure and there either way.
    /// </summary>
    private static IReadOnlyList<StatLine> Delay(ReceiveStreamStats s)
    {
        var d = s.Delay ?? new DelayBudget();

        return
        [
            Line("delay.total", Ms(d.HasTotalMs, d.TotalMs)),
            Line("delay.publish", Ms(d.HasPublishMs, d.PublishMs)),
            Line("delay.path", Ms(d.HasPathMs, d.PathMs)),
            Line("delay.arrive", Ms(d.HasArriveMs, d.ArriveMs)),
            Line("delay.decode", Ms(d.HasDecodeMs, d.DecodeMs)),
            Line("delay.present", Ms(d.HasPresentMs, d.PresentMs)),
            Line("latency", Latency(s)),
        ];
    }

    /// <summary>
    /// One stage of the path in milliseconds, and nothing at all where nothing measured it.
    /// One decimal under ten milliseconds and whole ones above:
    /// a stage of a fifth of a millisecond and one of half a second sit in one column,
    /// read against each other.
    /// </summary>
    private static string Ms(bool measured, double value)
    {
        if (!measured)
        {
            return "";
        }

        return value < 10 ? $"{value:0.0} ms" : $"{value:0} ms";
    }

    /// <summary>
    /// Sound track, and no block at all where the pipeline built no audio branch.
    /// Emptiness is the test rather than a flag, the branch filling every field here:
    /// a stream published without a track and one whose branch has not come up read alike.
    /// Telling the two apart is the tile's.
    /// </summary>
    private static IReadOnlyList<StatLine> Audio(ReceiveStreamStats s)
    {
        if (s.AudioCodecDescription.Length == 0 && s.AudioRate == 0)
        {
            return [];
        }

        return
        [
            Line("audio_codec_description", s.AudioCodecDescription),
            Line("audio_decoder", s.AudioDecoder),
            Line("audio_format", s.AudioFormat),
            Line("audio_rate", s.AudioRate > 0 ? $"{s.AudioRate} Hz" : ""),
            Line("audio_channels", s.AudioChannels > 0 ? $"{s.AudioChannels}" : ""),
            Line("audio_kbps", s.HasAudioKbps ? $"{s.AudioKbps:0} kb/s" : ""),
            Line("audio_bytes", Bytes(s.AudioBytes)),
        ];
    }

    /// <summary>
    /// What this window got and drew, the counts being totals since the subscription opened.
    /// Drawn on every kind of tile, these being the frame channel's counters,
    /// independent of what produces the pictures.
    /// </summary>
    private static IReadOnlyList<StatLine> Window(TileReport report) =>
    [
        Line("window.size", Pixels(report.Width, report.Height)),
        Line("window.frames", Count(report.Frames)),
        Line("window.dropped", Count(report.Dropped), report.Dropped),
    ];

    /// <summary>One element of the transport and the counters it keeps, in the element's own order.</summary>
    private static StatSection Transport(ReceiveStatGroup group)
    {
        Assert.That(group.Values.Count > 0, "a reported element carries at least one counter", group.Element);

        var element = Counters.Element(group.Factory);

        // The element's own name beside what it is: what tells two jitterbuffers of one muxed stream apart,
        // and the name to look for in a pipeline dump.
        return new StatSection(
            $"{element.Label} · {group.Element}",
            element.Tip,
            group.Values.Select(Line),
            Leads.GetValueOrDefault(group.Factory, ""));
    }

    /// <summary>
    /// One figure's row.
    /// An empty value means nothing measured the figure, printed as absent and never as a zero: the pads negotiate
    /// after the pipeline is built, so every field here is empty for a moment on every decode.
    /// </summary>
    private static StatLine Line(string key, string value, double reading = 0)
    {
        var counter = Counters.Field(key);
        return new StatLine(
            key, counter.Label, value.Length > 0 ? value : Figure.NoValue, counter.Tip, Alarms.Contains(key) && reading > 0);
    }

    /// <summary>One transport counter, keyed as the element names it.</summary>
    private static StatLine Line(ReceiveStatValue value) =>
        Line(value.Key, Counter(value.Key, value.Value), value.Value);

    /// <summary>
    /// One of an element's counters, the unit read off the key:
    /// "-ms" milliseconds, "-mbps" Mb/s, anything else a plain count.
    /// An element names its fields for what they report, so the key is where a unit is stated at all.
    /// </summary>
    private static string Counter(string key, double value)
    {
        if (key.EndsWith("-ms", StringComparison.Ordinal))
        {
            return $"{value:0.0} ms";
        }

        if (key.EndsWith("-mbps", StringComparison.Ordinal))
        {
            return $"{value:0.00} Mb/s";
        }

        return value.ToString("N0");
    }

    /// <summary>A picture size in pixels, and nothing before the pads negotiate one.</summary>
    private static string Pixels(int width, int height) =>
        width > 0 && height > 0 ? $"{width}×{height}" : "";

    /// <summary>
    /// Frame rate the caps declare, per second, a claim rather than a measurement.
    /// A denominator of zero declares a variable rate, a different fact from declaring none.
    /// </summary>
    private static string Declared(int num, int den)
    {
        if (num <= 0)
        {
            return "";
        }

        return den > 0 ? $"{(double)num / den:0.##} /s" : "variable";
    }

    /// <summary>
    /// Latency window in milliseconds as one reading, the two ends being meaningful only together.
    /// Where no upper end was reported, the lower one stands alone rather than beside an invented bound.
    /// </summary>
    private static string Latency(ReceiveStreamStats s)
    {
        if (!s.HasLatencyMinMs)
        {
            return "";
        }

        return s.HasLatencyMaxMs && s.LatencyMaxMs > s.LatencyMinMs
            ? $"{s.LatencyMinMs:0} to {s.LatencyMaxMs:0} ms"
            : $"{s.LatencyMinMs:0} ms";
    }

    /// <summary>Two caps fields that read as one answer, either half possibly absent.</summary>
    private static string Joined(string first, string second) =>
        string.Join(" ", new[] { first, second }.Where(part => part.Length > 0));

    /// <summary>A running total, grouped, these reaching seven digits within the hour.</summary>
    private static string Count(ulong value) => value.ToString("N0");

    /// <summary>
    /// A byte total at a scale worth reading.
    /// Decimal multiples rather than binary, being read beside a bitrate.
    /// </summary>
    private static string Bytes(ulong value) => value switch
    {
        0 => "",
        >= 1_000_000_000 => $"{value / 1_000_000_000d:0.00} GB",
        >= 1_000_000 => $"{value / 1_000_000d:0.0} MB",
        >= 1_000 => $"{value / 1_000d:0.0} kB",
        _ => $"{value} B",
    };

    /// <summary>
    /// Seconds as a clock reads them.
    /// Position and uptime sit beside each other and are compared, a stalled position against an uptime that runs
    /// on, so both are spelled the same way.
    /// </summary>
    private static string Elapsed(double seconds)
    {
        var span = TimeSpan.FromSeconds(seconds);
        return span.TotalHours >= 1
            ? $"{(int)span.TotalHours}:{span.Minutes:00}:{span.Seconds:00}"
            : $"{span.Minutes}:{span.Seconds:00}";
    }
}
