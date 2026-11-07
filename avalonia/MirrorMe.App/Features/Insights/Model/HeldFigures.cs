using MirrorMe.App.Contracts;

namespace MirrorMe.App.Features.Insights.Model;

/// <summary>
/// Last measurement each figure of the running stream stated, so a pass that measures none prints what the last
/// one did instead of <see cref="Figure.NoValue"/>.
///
/// A sample states a figure only where something measured it over that interval, and a second is short enough
/// for that to fail on a healthy stream: an encoder that emitted no frame in one times nothing, and a relay poll
/// that has not landed states no reader.
/// Both are gaps in a figure that is otherwise there, and a card alternating between a number and an ellipsis
/// is one that nobody reads.
///
/// Held for the run and no further.
/// A stream that stopped, one waiting out a retry and one publishing under another name measure nothing at all,
/// so every held figure goes and the screen reads absent.
/// </summary>
public sealed class HeldFigures
{
    private string _stream = "";
    private double? _egress;
    private double? _fps;
    private double? _encode;
    private int? _viewers;

    /// <summary>
    /// The reading with each unmeasured figure filled from the last pass that measured it.
    /// Safe to run twice on one reading: a filled figure is recorded as the value it was filled with.
    /// </summary>
    public InsightsSnapshot Fill(InsightsSnapshot reading)
    {
        Assert.NotNull(reading, "a hold fills the reading it is handed");

        // A stream that stopped and one waiting out a retry measure nothing, so there is nothing to fill from.
        if (!reading.IsLive || reading.IsRetrying)
        {
            Forget(reading.Stream);
            return reading;
        }

        // A stream under another name is another run, and its first pass still measures: forgetting what
        // the last one read is the whole of it.
        if (reading.Stream != _stream)
        {
            Forget(reading.Stream);
        }

        var filled = reading with
        {
            EgressMbps = Held(ref _egress, reading.EgressMbps),
            Fps = Held(ref _fps, reading.Fps),
            EncodeMs = Held(ref _encode, reading.EncodeMs),
            Viewers = Held(ref _viewers, reading.Viewers),
        };

        Assert.That(filled.IsLive && filled.Stream == reading.Stream,
            "a held figure describes the run that measured it", reading.Stream);
        return filled;
    }

    /// <summary>Drops every held figure and takes the run they will be measured on.</summary>
    private void Forget(string stream)
    {
        _stream = stream;
        _egress = _fps = _encode = null;
        _viewers = null;
    }

    /// <summary>What this pass measured, or the last pass that measured anything.</summary>
    private static T? Held<T>(ref T? held, T? measured) where T : struct
    {
        if (measured is not null)
        {
            held = measured;
        }

        return measured ?? held;
    }
}
