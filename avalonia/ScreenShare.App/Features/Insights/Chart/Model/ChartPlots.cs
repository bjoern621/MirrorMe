using System.Globalization;
using Avalonia;
using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Contracts;

namespace ScreenShare.App.Features.Insights.Chart.Model;

/// <summary>A timeline mark placed on the chart. At: seconds from the left edge.</summary>
public sealed record ChartMark(double At, string Label);

/// <summary>What the chart header states for the span drawn. Worst: the low of a frame rate, the peak of the rest.</summary>
public sealed record ChartSummary(double? Average, double? Worst, double PastLimitSeconds);

/// <summary>
/// One figure over one span.
/// Points: X in seconds from the left edge, 0..Span; Y the reading itself, 0..Top.
/// </summary>
public sealed record ChartPlot(
    IReadOnlyList<Point> Points,
    double Span,
    double Top,
    double? Limit,
    ChartSummary Summary,
    IReadOnlyList<ChartMark> Marks)
{
    public static readonly ChartPlot Empty = new([], ChartPlots.MinimumSpanSeconds, 1, null, new(null, null, 0), []);
}

/// <summary>Every card's figure at one reading, the one place the three meet.</summary>
public sealed record Readout(string Time, string FrameRate, string EncodeTime, string Bitrate);

/// <summary>
/// Chart geometry and the header's summary, read out of the session's stamped readings.
/// Display arithmetic alone: the backend's timeline judges the stream, and this draws what was measured.
/// </summary>
public static class ChartPlots
{
    /// <summary>Narrowest span drawn, s. A run younger than this fills the right of the chart.</summary>
    public const double MinimumSpanSeconds = 60;

    /// <summary>Gap between readings counted past a limit at most, s. A longer one went unmeasured.</summary>
    private const double MaxIntervalSeconds = 5;

    /// <summary>Share of the height the highest value or limit may take, the rest keeping it off the top edge.</summary>
    private const double Headroom = 0.85;

    public static ChartPlot Of(
        IReadOnlyList<PublishReading> readings,
        Metric metric,
        Limits limits,
        ChartRange range,
        IReadOnlyList<StreamEvent> events)
    {
        Assert.NotNull(readings, "a chart is drawn from the readings the session kept");
        Assert.NotNull(events, "a chart marks the timeline's events");

        if (readings.Count == 0)
        {
            return ChartPlot.Empty;
        }

        var (start, span) = Window(readings, range);
        var limit = metric.Limit(limits);

        var points = new List<Point>(readings.Count);
        var past = 0.0;
        DateTimeOffset? previous = null;
        foreach (var reading in readings)
        {
            if (reading.At < start || metric.Read(reading.Stats) is not { } value)
            {
                continue;
            }

            if (limit is { } bound && metric.IsPast(value, bound) && previous is { } before)
            {
                past += Math.Min((reading.At - before).TotalSeconds, MaxIntervalSeconds);
            }

            previous = reading.At;
            points.Add(new Point((reading.At - start).TotalSeconds, value));
        }

        var values = points.Select(point => point.Y).ToList();
        var summary = new ChartSummary(
            values.Count == 0 ? null : values.Average(),
            values.Count == 0 ? null : metric.LowIsBad ? values.Min() : values.Max(),
            past);
        var top = Nice(Math.Max(values.DefaultIfEmpty(0).Max(), limit ?? 0) / Headroom);

        var plot = new ChartPlot(points, span, top, limit, summary, Marks(events, start, span));

        Assert.That(plot.Points.All(point => point.X >= 0 && point.X <= span), "every point sits inside the span", span);
        Assert.That(plot.Top > 0, "the chart has height to draw in", plot.Top);
        return plot;
    }

    /// <summary>
    /// Every figure of the reading nearest to a moment, null where the span holds none.
    /// seconds: from the left edge, as the chart reports a pointer.
    /// </summary>
    public static Readout? ReadoutAt(IReadOnlyList<PublishReading> readings, ChartRange range, double seconds)
    {
        Assert.NotNull(readings, "a readout reads the readings the session kept");

        if (readings.Count == 0 || double.IsNaN(seconds))
        {
            return null;
        }

        var (start, _) = Window(readings, range);
        var at = start.AddSeconds(seconds);
        var nearest = readings
            .Where(reading => reading.At >= start)
            .MinBy(reading => Math.Abs((reading.At - at).TotalSeconds));
        if (nearest is null)
        {
            return null;
        }

        return new Readout(
            nearest.At.ToLocalTime().ToString("HH:mm:ss", CultureInfo.CurrentCulture),
            Print(MetricKey.FrameRate, nearest.Stats),
            Print(MetricKey.EncodeTime, nearest.Stats),
            Print(MetricKey.Bitrate, nearest.Stats));
    }

    /// <summary>Left edge and span. The newest reading holds the right edge.</summary>
    private static (DateTimeOffset Start, double Span) Window(IReadOnlyList<PublishReading> readings, ChartRange range)
    {
        var end = readings[^1].At;
        var span = ChartRanges.Of(range).Seconds
            ?? Math.Max(MinimumSpanSeconds, (end - readings[0].At).TotalSeconds);
        return (end.AddSeconds(-span), span);
    }

    private static IReadOnlyList<ChartMark> Marks(IReadOnlyList<StreamEvent> events, DateTimeOffset start, double span)
    {
        var marks = new List<ChartMark>();
        foreach (var e in events)
        {
            var at = (DateTimeOffset.FromUnixTimeMilliseconds(e.AtUnixMillis) - start).TotalSeconds;
            if (at < 0 || at > span)
            {
                continue;
            }

            marks.Add(new ChartMark(at, Label(e)));
        }

        return marks;
    }

    private static string Label(StreamEvent e) => e.Kind switch
    {
        StreamEvent.Types.Kind.Join => $"{e.Who} joined",
        StreamEvent.Types.Kind.Leave => $"{e.Who} left",
        StreamEvent.Types.Kind.Dip => $"Fell to {e.Fps:0} fps",
        StreamEvent.Types.Kind.Restart => "Restarted",
        _ => Assert.Never<string>("every timeline event kind has a label", e.Kind),
    };

    private static string Print(MetricKey key, PublishStats stats)
    {
        var metric = Metrics.Of(key);
        return metric.Print(metric.Read(stats));
    }

    /// <summary>Smallest round figure at or above value: 1, 2, 2.5, 5 or 10 times a power of ten.</summary>
    private static double Nice(double value)
    {
        if (value <= 0)
        {
            return 1;
        }

        var power = Math.Pow(10, Math.Floor(Math.Log10(value)));
        foreach (var step in new[] { 1, 2, 2.5, 5, 10 })
        {
            if (step * power >= value)
            {
                return step * power;
            }
        }

        return Assert.Never<double>("ten times the power of ten clears the value", value);
    }
}
