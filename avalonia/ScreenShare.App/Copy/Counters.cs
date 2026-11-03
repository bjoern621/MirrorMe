namespace ScreenShare.App.Copy;

/// <summary>
/// One figure of the stats panel.
/// A label with no tip names a number nobody can act on, so the two get written together.
/// </summary>
/// <param name="Label">Name of the figure, as the row prints it.</param>
/// <param name="Tip">What a reading tells the reader, beyond the unit beside it (<c>docs/tooltips.md</c>).</param>
public readonly record struct Counter(string Label, string Tip);

/// <summary>
/// Every word of the stats panel: its headings, its rows and the counters of the transport's elements.
/// Keyed on the identifiers that both sides share:
/// the contract's field names for a decode's sample, and an element's own names for its counters
/// (<c>api/proto/screenshare/v1/text.proto</c>).
/// The leg a decode opened on decides its transport rows: SRT counts a link, RTSP a jitter buffer per track.
/// </summary>
public static class Counters
{
    /// <summary>Panel's headings, in the order that frames pass through the stages.</summary>
    private static readonly Dictionary<string, Counter> Headings = new()
    {
        ["section.stream"] = new(
            "Arriving",
            "The encoded stream as it reaches this computer. Shows what the publisher sent and the relay carried."),
        ["section.picture"] = new(
            "Picture",
            "The decoded picture, at the size the publisher encoded."),
        ["section.decode"] = new(
            "Decode",
            "Which decoder runs, where its frames go, and how many it drops to stay on time."),
        ["section.render"] = new(
            "Render",
            "The steps between the decoder and this window."),
        ["section.delay"] = new(
            "Delay",
            "What each stage adds, from the publisher's screen to this window. The relay's own time is not measured, so the real delay can be higher."),
        ["section.clock"] = new(
            "Clock",
            "How the decode keeps time. A live stream cannot pause, so late frames get dropped."),
        ["section.audio"] = new(
            "Audio",
            "The stream's sound track."),
        ["section.window"] = new(
            "This window",
            "What this window received and drew. A slow compositor shows up only here."),
    };

    /// <summary>
    /// What each element of the transport does, keyed on its factory name.
    /// The counters under a block mean different things depending on which element keeps them.
    /// </summary>
    private static readonly Dictionary<string, Counter> Elements = new()
    {
        ["srtsrc"] = new(
            "SRT link",
            "The SRT connection from the relay to this computer. These counters cover that hop only."),
        ["rtpjitterbuffer"] = new(
            "RTP jitter buffer",
            "Puts RTP packets back in order and asks for missing ones. There is one per track."),
    };

    /// <summary>
    /// Every row, keyed on the contract field that it prints,
    /// or in a transport block on the element's own name for the counter.
    /// </summary>
    private static readonly Dictionary<string, Counter> Fields = new()
    {
        // section.stream
        ["codec_description"] = new(
            "Codec",
            "The format the publisher encodes in. Every viewer gets the same one."),
        ["profile"] = new(
            "Profile",
            "The part of the codec this stream uses. A hardware decoder can support a codec but not every profile."),
        ["level"] = new(
            "Level",
            "The codec's limit on size, frame rate and bitrate. A decoder that refuses a stream it supports usually refuses the level."),
        ["video_mbps"] = new(
            "Bitrate",
            "Video received over the last second. Staying below the publisher's setting means the path from the relay is too slow."),
        ["video_fps"] = new(
            "Frames arriving",
            "Encoded frames received in the last second. Below the declared rate means frames get lost on the way."),
        ["declared_fps"] = new(
            "Declared rate",
            "The frame rate the stream says it runs at. Compare the measured rates against it."),
        ["keyframes"] = new(
            "Keyframes",
            "Frames a decoder can start from, counted since this decode opened. A new viewer waits for the next one."),
        ["since_keyframe_sec"] = new(
            "Last keyframe",
            "Time since the last keyframe. Longer than the publisher's keyframe interval means keyframes stopped arriving."),
        ["video_bytes"] = new(
            "Video received",
            "All video data since this decode opened. A data cap counts this."),

        // section.picture
        ["picture_size"] = new(
            "Size",
            "The picture as the publisher encoded it. The size this window draws is under Render."),
        ["pixel_format"] = new(
            "Pixel format",
            "How the decoded samples are laid out, as GStreamer names it."),
        ["depth"] = new(
            "Depth",
            "Bits per color component. 8 is standard. 10 can carry HDR."),
        ["subsampling"] = new(
            "Chroma sampling",
            "How much color detail survived the encode. 4:4:4 keeps all of it, for sharp text. 4:2:0 keeps a quarter."),
        ["colorimetry"] = new(
            "Color",
            "Range, matrix, primaries and transfer. A missing part gets guessed. A wrong guess draws the picture too bright or too dark."),
        ["transfer"] = new(
            "Transfer",
            "The curve from code values to light. HDR drawn without conversion looks flat and dim."),
        ["chroma_site"] = new(
            "Chroma siting",
            "Where color samples sit against brightness samples. A mismatch puts a colored fringe on thin lines."),
        ["pixel_aspect"] = new(
            "Pixel aspect",
            "The shape of one pixel. Anything but 1:1 needs stretching to look right."),
        ["interlace"] = new(
            "Interlacing",
            "Whole frames or fields. Screen capture is progressive. Fields point to a camera or TV source."),

        // section.decode
        ["decoder"] = new(
            "Decoder",
            "The decoder this computer picked. Two computers watching one stream can use different ones."),
        ["decode_memory"] = new(
            "Decoded into",
            "Where the decoder puts its frames. A GPU decoder writing to system memory copies every frame."),
        ["discarded_fps"] = new(
            "Discarded to keep up",
            "Frames dropped each second so the picture stays current. A steady rate means the publisher could lower its frame rate."),
        ["tone_map"] = new(
            "Tone mapping",
            "Whether this decode converts HDR to this display's range. Without a converter on this computer, it stays off."),

        // section.render
        ["chain"] = new(
            "Render chain",
            "What converts frames between the decoder and this window. When the driver handles color, two computers can show different brightness."),
        ["render_memory"] = new(
            "Window reads from",
            "Where frames are when this window gets them. Different from the decoder's memory means a copy happened."),
        ["render_format"] = new(
            "Draw format",
            "The pixel format and color this window draws. Fixed, because a guessed curve washes out desktop content."),
        ["render_size"] = new(
            "Draw size",
            "The size frames reach this window at. Smaller than the picture means the stream got scaled down to fit this tile."),
        ["render_fps"] = new(
            "Frames drawn",
            "Frames drawn in the last second. Below the arriving rate means this computer cannot keep up."),
        ["rendered"] = new(
            "Total drawn",
            "Frames drawn since this decode opened."),
        ["sink_dropped"] = new(
            "Dropped late",
            "Frames dropped after decoding, right before drawing. A healthy decode keeps this at 0."),

        // section.clock
        ["live"] = new(
            "Live timing",
            "Whether the decode runs on a clock it cannot pause. Every relay stream does. Frames not decoded in time get dropped."),
        ["position_sec"] = new(
            "Position",
            "How far the decode has played. Stuck while uptime climbs means the stream stalled."),
        ["uptime_sec"] = new(
            "Uptime",
            "How long this decode has run. Resets when the decode restarts, as turning on tone mapping does."),

        // section.delay
        ["delay.total"] = new(
            "End to end",
            "All measured stages added up. A stage with no figure is left out, so the real delay can be higher."),
        ["delay.publish"] = new(
            "Capture and encode",
            "Time the publisher spends reading the screen and encoding a frame. A faster effort setting or a shorter lookahead cuts it."),
        ["delay.path"] = new(
            "Publisher to here",
            "Network and relay time before this computer buffers the frame. Measured only on H.264 and H.265."),
        ["delay.arrive"] = new(
            "Buffered here",
            "Time a frame waits in this computer's transport buffer and parser. The leg's latency setting controls it."),
        ["delay.decode"] = new(
            "Decode",
            "Time the decoder takes per frame. Nearing the latency window means frames will start to drop."),
        ["delay.present"] = new(
            "Waiting to show",
            "Time a frame waits for its moment on screen. Buffering, decoding and this wait together fill the latency window."),
        ["latency"] = new(
            "Latency window",
            "How long a frame is held before it plays, to absorb jitter. Larger is steadier but later."),

        // section.audio
        ["audio_codec_description"] = new(
            "Codec",
            "The format the sound track is encoded in."),
        ["audio_decoder"] = new(
            "Decoder",
            "What decodes the sound track."),
        ["audio_format"] = new(
            "Sample format",
            "How the decoded audio samples are laid out."),
        ["audio_rate"] = new(
            "Sample rate",
            "Audio samples per second."),
        ["audio_channels"] = new(
            "Channels",
            "Number of audio channels."),
        ["audio_kbps"] = new(
            "Bitrate",
            "Audio received over the last second."),
        ["audio_bytes"] = new(
            "Audio received",
            "All audio data since this decode opened."),

        // section.window
        ["window.size"] = new(
            "Frame size",
            "The size of the frames this window gets. Markers over the picture line up against it."),
        ["window.frames"] = new(
            "Frames received",
            "Frames this window got since it opened the stream. A window stuck behind the compositor receives frames it never shows."),
        ["window.dropped"] = new(
            "Dropped for this window",
            "Frames thrown away because this window still held earlier ones. Points to drawing as the slow part."),

        // srtsrc
        ["packets-received"] = new(
            "Packets",
            "All packets that arrived, resent ones included."),
        ["packets-received-lost"] = new(
            "Lost",
            "Packets that never arrived, even after resending. Each one shows as blocks or smearing."),
        ["packets-received-retransmitted"] = new(
            "Resent",
            "Packets that arrived only after being asked for again. Rising with nothing lost means SRT covers for a lossy path."),
        ["packets-received-dropped"] = new(
            "Dropped",
            "Packets that arrived too late to play. A higher SRT latency trades delay for fewer of these."),
        ["receive-rate-mbps"] = new(
            "Receive rate",
            "What the link delivers, protocol overhead included. Sits a little above the video bitrate."),
        ["bandwidth-mbps"] = new(
            "Estimated capacity",
            "What SRT thinks the path from the relay can carry. A bitrate close to it starts losing packets."),
        ["rtt-ms"] = new(
            "Round trip",
            "Time for a packet to reach the relay and come back. The latency window needs several times this for resending to work."),
        ["negotiated-latency-ms"] = new(
            "Buffer",
            "How long both ends hold packets to fill gaps. The relay sets a minimum."),

        // rtpjitterbuffer
        ["num-pushed"] = new(
            "Passed on",
            "Packets sent on to the decoder, in order."),
        ["num-lost"] = new(
            "Lost",
            "Packets the buffer stopped waiting for. The decoder has to hide each gap."),
        ["num-late"] = new(
            "Late",
            "Packets that arrived after their play time. A rising count means the buffer is too short for this path."),
        ["num-duplicates"] = new(
            "Duplicates",
            "Packets that arrived twice. A few are normal with resending. Many point to a faulty path."),
        ["rtx-count"] = new(
            "Resends asked",
            "Times the buffer saw a gap and asked for the packet again."),
        ["rtx-success-count"] = new(
            "Resends in time",
            "Requests answered in time to use. The gap to the count above is recovery that came too late."),
    };

    public static Counter Heading(string id) => Look(Headings, id);

    public static Counter Element(string factory) => Look(Elements, factory);

    public static Counter Field(string key) => Look(Fields, key);

    /// <summary>
    /// Entry for an identifier, or the identifier itself with an empty tip.
    /// The backend still sends a counter that this build has no words for,
    /// and a row printing the raw key gives the reader something to search for and report.
    /// </summary>
    private static Counter Look(Dictionary<string, Counter> table, string id) =>
        id.Length > 0 && table.TryGetValue(id, out var entry) ? entry : new Counter(id, "");
}
