using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Features.Insights.Chart.Model;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// The chart draws one figure over a span, against that figure's limit.
/// Every point, summary and mark comes out of the readings and the timeline handed in.
/// </summary>
public sealed class InsightsChartTests
{
    private static readonly DateTimeOffset T0 = new(2026, 11, 1, 18, 40, 0, TimeSpan.Zero);

    private static readonly Limits AtSixty = new(TargetFps: 60, CeilingMbps: 20);

    private static PublishReading Read(int second, double fps, double transitMs = 10, double mbps = 12) => new(
        new PublishStats { Fps = fps, TransitMs = transitMs, InstMbps = mbps },
        T0.AddSeconds(second));

    private static StreamEvent Joined(string who, int second) => new()
    {
        Kind = StreamEvent.Types.Kind.Join,
        Who = who,
        AtUnixMillis = T0.AddSeconds(second).ToUnixTimeMilliseconds(),
    };

    [Fact]
    public void AReadingOlderThanTheRangeLeavesThePlot()
    {
        var readings = new[] { Read(0, 60), Read(30, 60), Read(70, 60), Read(90, 60) };

        var plot = ChartPlots.Of(readings, Metrics.Of(MetricKey.FrameRate), AtSixty, ChartRange.OneMinute, []);

        Assert.Equal(60, plot.Span);
        Assert.Equal([0.0, 40, 60], plot.Points.Select(point => point.X));
    }

    [Fact]
    public void TheSessionRangeSpansEveryReadingKept()
    {
        var readings = new[] { Read(0, 60), Read(200, 60), Read(400, 60) };

        var plot = ChartPlots.Of(readings, Metrics.Of(MetricKey.FrameRate), AtSixty, ChartRange.Session, []);

        Assert.Equal(400, plot.Span);
        Assert.Equal(3, plot.Points.Count);
    }

    [Fact]
    public void TheSummaryCountsTheTimeUnderTarget()
    {
        var readings = new[] { Read(0, 60), Read(1, 48), Read(2, 50), Read(3, 60) };

        var summary = ChartPlots.Of(readings, Metrics.Of(MetricKey.FrameRate), AtSixty, ChartRange.OneMinute, []).Summary;

        Assert.Equal(54.5, summary.Average);
        Assert.Equal(48, summary.Worst);
        Assert.Equal(2, summary.PastLimitSeconds);
    }

    [Fact]
    public void EncodeTimeIsHeldToTheFramesOwnDuration()
    {
        var limit = Metrics.Of(MetricKey.EncodeTime).Limit(AtSixty);

        Assert.Equal(1000.0 / 60, limit);
    }

    [Fact]
    public void AnUnboundedEncodeHasNoBitrateLimit()
    {
        var plot = ChartPlots.Of([Read(0, 60), Read(1, 60)], Metrics.Of(MetricKey.Bitrate),
            AtSixty with { CeilingMbps = null }, ChartRange.OneMinute, []);

        Assert.Null(plot.Limit);
        Assert.Equal(0, plot.Summary.PastLimitSeconds);
    }

    [Fact]
    public void TheTopClearsBothThePeakAndTheLimit()
    {
        var plot = ChartPlots.Of([Read(0, 30), Read(1, 31)], Metrics.Of(MetricKey.FrameRate), AtSixty,
            ChartRange.OneMinute, []);

        Assert.True(plot.Top > 60, $"top {plot.Top} leaves the target off the chart");
    }

    [Fact]
    public void AMarkInsideTheRangeIsPlacedAndNamed()
    {
        var readings = new[] { Read(0, 60), Read(100, 60) };

        var marks = ChartPlots.Of(readings, Metrics.Of(MetricKey.FrameRate), AtSixty, ChartRange.OneMinute,
            [Joined("Lea", 10), Joined("Kim", 70)]).Marks;

        var mark = Assert.Single(marks);
        Assert.Equal(30, mark.At);
        Assert.Equal("Kim joined", mark.Label);
    }

    [Fact]
    public void TheReadoutGivesEveryFigureOfTheNearestReading()
    {
        var readings = new[] { Read(0, 60), Read(1, 48, transitMs: 21, mbps: 14.2), Read(2, 60) };

        var readout = ChartPlots.ReadoutAt(readings, ChartRange.OneMinute, seconds: 59.2);

        Assert.NotNull(readout);
        Assert.Equal("48 fps", readout.FrameRate);
        Assert.Equal("21 ms", readout.EncodeTime);
        Assert.Equal("14.2 Mb/s", readout.Bitrate);
    }
}
