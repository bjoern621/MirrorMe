using MirrorMe.Api.V1;
using MirrorMe.App.Backend;
using MirrorMe.App.Features.Insights.Chart.Model;
using MirrorMe.App.Features.Insights.Chart.ViewModel;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// The chart draws one figure over a span, against that figure's limit.
/// Every point and summary comes out of the readings handed in.
/// </summary>
public sealed class InsightsChartTests
{
    private static readonly DateTimeOffset T0 = new(2026, 11, 1, 18, 40, 0, TimeSpan.Zero);

    private static readonly Limits AtSixty = new(TargetFps: 60, CeilingMbps: 20);

    private static PublishReading Read(int second, double fps, double transitMs = 10, double mbps = 12) => new(
        new PublishStats { Fps = fps, TransitMs = transitMs, InstMbps = mbps },
        T0.AddSeconds(second));

    [Fact]
    public void AReadingOlderThanTheRangeLeavesThePlot()
    {
        var readings = new[] { Read(0, 60), Read(30, 60), Read(70, 60), Read(90, 60) };

        var plot = ChartPlots.Of(readings, Metrics.Of(MetricKey.FrameRate), AtSixty, ChartRange.OneMinute);

        Assert.Equal(60, plot.Span);
        Assert.Equal([0.0, 40, 60], plot.Points.Select(point => point.X));
    }

    [Fact]
    public void TheSessionRangeSpansEveryReadingKept()
    {
        var readings = new[] { Read(0, 60), Read(200, 60), Read(400, 60) };

        var plot = ChartPlots.Of(readings, Metrics.Of(MetricKey.FrameRate), AtSixty, ChartRange.Session);

        Assert.Equal(400, plot.Span);
        Assert.Equal(3, plot.Points.Count);
    }

    [Fact]
    public void TheSummaryStatesTheAverageAndTheLow()
    {
        var readings = new[] { Read(0, 60), Read(1, 48), Read(2, 50), Read(3, 60) };

        var summary = ChartPlots.Of(readings, Metrics.Of(MetricKey.FrameRate), AtSixty, ChartRange.OneMinute).Summary;

        Assert.Equal(54.5, summary.Average);
        Assert.Equal(48, summary.Worst);
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
            AtSixty with { CeilingMbps = null }, ChartRange.OneMinute);

        Assert.Null(plot.Limit);
        Assert.DoesNotContain(true, plot.Past);
    }

    [Fact]
    public void TheAxisHoldsBothThePeakAndTheLimit()
    {
        var plot = ChartPlots.Of([Read(0, 30), Read(1, 31)], Metrics.Of(MetricKey.FrameRate), AtSixty,
            ChartRange.OneMinute);

        Assert.True(plot.Top >= 60, $"top {plot.Top} leaves the target off the chart");
        Assert.True(plot.Bottom <= 30, $"bottom {plot.Bottom} leaves the low off the chart");
    }

    /// <summary>A dip of a few frames fills the height that a zero-based axis would spend on nothing.</summary>
    [Fact]
    public void AShallowDipIsDrawnOnAnAxisCloseAroundIt()
    {
        var plot = ChartPlots.Of([Read(0, 60), Read(1, 48), Read(2, 60)], Metrics.Of(MetricKey.FrameRate), AtSixty,
            ChartRange.OneMinute);

        Assert.Equal(20, plot.Bottom);
        Assert.Equal(60, plot.Top);
    }

    [Fact]
    public void AReadingPastTheLimitIsMarkedPast()
    {
        var plot = ChartPlots.Of([Read(0, 60), Read(1, 48), Read(2, 60)], Metrics.Of(MetricKey.FrameRate), AtSixty,
            ChartRange.OneMinute);

        Assert.Equal([false, true, false], plot.Past);
    }

    [Fact]
    public void TheReadoutGivesEveryFigureOfTheNearestReading()
    {
        var readings = new[] { Read(0, 60), Read(1, 48, transitMs: 21, mbps: 14.2), Read(2, 60) };

        var readout = ChartPlots.ReadoutAt(readings, ChartRange.OneMinute, seconds: 59.2);

        Assert.Equal("48 fps", readout.FrameRate);
        Assert.Equal("21 ms", readout.EncodeTime);
        Assert.Equal("14.2 Mb/s", readout.Bitrate);
    }

    [Fact]
    public void TheReadoutPrintsAnEllipsisForEveryFigureWhileThePointerRestsNowhere()
    {
        var readout = ChartPlots.ReadoutAt([Read(0, 60)], ChartRange.OneMinute, double.NaN);

        Assert.Equal(new Readout("…", "…", "…", "…"), readout);
    }

    [Fact]
    public void TheReadoutPrintsAnEllipsisForEveryFigureWithNoReadings()
    {
        var readout = ChartPlots.ReadoutAt([], ChartRange.OneMinute, seconds: 30);

        Assert.Equal(new Readout("…", "…", "…", "…"), readout);
    }

    [Fact]
    public void TheChartOpensOnAReadoutOfEllipses()
    {
        Assert.Equal(new Readout("…", "…", "…", "…"), new ChartViewModel().Readout);
    }

    [Fact]
    public void TheWidestReadoutHoldsThreeDigitsOfEveryFigure()
    {
        var widest = ChartPlots.WidestReadout;

        Assert.Equal(8, widest.Time.Length);
        Assert.Equal("999 fps", widest.FrameRate);
        Assert.Equal("999 ms", widest.EncodeTime);
        Assert.Equal("999.0 Mb/s", widest.Bitrate);
    }
}
