using System.Collections.Specialized;
using ScreenShare.Api.V1;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Insights.Model;
using ScreenShare.App.Features.Insights.ViewerTable.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Row per viewer that reported, cell per figure its own app measured.
/// Unmeasured figure reads absent, never zero, so a viewer whose clock nothing could compare never reads as
/// a viewer with no latency at all.
/// </summary>
public sealed class ViewerTableTests
{
    private static ViewerReport Report(string name, string transport, double? latencyMs = null, double? fps = null,
        double? droppedFps = null)
    {
        var report = new ViewerReport { MemberId = $"m-{name}", DisplayName = name, Transport = transport };
        if (latencyMs is { } latency)
        {
            report.PathMs = latency;
        }
        if (fps is { } shown)
        {
            report.ShownFps = shown;
        }
        if (droppedFps is { } dropped)
        {
            report.DroppedFps = dropped;
        }
        return report;
    }

    private static RelayStatus Serving(int readers, params ViewerReport[] reports)
    {
        var path = new RelayPath { Name = "desk", Ready = true, Readers = readers };
        path.Viewers.AddRange(reports);

        var relay = new RelayStatus { Reachable = true };
        relay.Paths.Add(path);
        return relay;
    }

    private static RelayStatus Serving(params ViewerReport[] reports) => Serving(reports.Length, reports);

    private static ViewerTableViewModel Table(RelayStatus? relay)
    {
        var path = InsightsSnapshot.PathOf(relay, "desk");
        return new ViewerTableViewModel
        {
            Reported = path is null ? [] : path.Viewers.Select(ViewerRow.Of).ToList(),
            Readers = path?.Readers,
            IsLive = true,
        };
    }

    [Fact]
    public void ARowCarriesWhatTheViewersOwnAppMeasured()
    {
        var table = Table(Serving(Report("Alice", "rtsp", latencyMs: 41.6, fps: 59.8, droppedFps: 0)));

        var row = Assert.Single(table.Rows);
        Assert.Equal("Alice", row.Name);
        Assert.Equal("42", row.Latency);
        Assert.Equal("60", row.Fps);
        Assert.Equal("0", row.Dropped);
        Assert.Equal("rtsp", row.Via);
        Assert.False(row.IsStruggling);
    }

    /// <summary>Every leg reports the same figures, so an RTMP viewer is timed like an SRT one.</summary>
    [Fact]
    public void EveryLegCarriesALatency()
    {
        var table = Table(Serving(
            Report("Alice", "rtmp", latencyMs: 2700),
            Report("Bob", "hls", latencyMs: 540),
            Report("Carol", "webrtc", latencyMs: 33)));

        Assert.Equal(["2700", "540", "33"], table.Rows.Select(row => row.Latency));
    }

    [Fact]
    public void AFigureTheViewerDidNotMeasureReadsAsAbsentRatherThanAsZero()
    {
        var table = Table(Serving(Report("Alice", "srt")));

        var row = Assert.Single(table.Rows);
        Assert.Equal(Figure.NoValue, row.Latency);
        Assert.Equal(Figure.NoValue, row.Fps);
        Assert.Equal(Figure.NoValue, row.Dropped);
        Assert.False(row.IsStruggling);
    }

    /// <summary>A member the group's list no longer names is still somebody watching.</summary>
    [Fact]
    public void AViewerTheGroupNamesNothingStillRenders()
    {
        var table = Table(Serving(new ViewerReport { MemberId = "m-gone", Transport = "rtsp" }));

        var row = Assert.Single(table.Rows);
        Assert.Equal(Figure.NoValue, row.Name);
        Assert.Equal("rtsp", row.Via);
    }

    [Fact]
    public void OnlyFramesTheViewerShedMakeARowStruggle()
    {
        var table = Table(Serving(
            Report("Alice", "rtsp", latencyMs: 40, fps: 60, droppedFps: 0),
            Report("Bob", "rtsp", latencyMs: 40, fps: 52, droppedFps: 8),
            Report("Carol", "rtmp", latencyMs: 2700, fps: 60, droppedFps: 0)));

        Assert.Equal(1, table.StrugglingCount);
        Assert.False(table.Rows[0].IsStruggling);
        Assert.True(table.Rows[1].IsStruggling);
        Assert.False(table.Rows[2].IsStruggling);
    }

    [Fact]
    public void OneRowEndsTheTableAndCarriesNoSeparator()
    {
        var table = Table(Serving(Report("Alice", "rtsp"), Report("Bob", "srt"), Report("Carol", "hls")));

        Assert.Equal(3, table.Rows.Count);
        Assert.False(table.Rows[0].IsLast);
        Assert.False(table.Rows[1].IsLast);
        Assert.True(table.Rows[2].IsLast);
    }

    /// <summary>
    /// Relay is polled while the pointer rests on the card,
    /// so a clear-and-fill per poll would reset the scroll position of a table nothing happened to.
    /// </summary>
    [Fact]
    public void AnUnchangedTableRenderedAgainTouchesNothing()
    {
        var relay = Serving(Report("Alice", "rtsp", latencyMs: 40), Report("Bob", "srt"));
        var table = Table(relay);

        var changes = 0;
        ((INotifyCollectionChanged)table.Rows).CollectionChanged += (_, _) => changes++;

        table.Reported = InsightsSnapshot.PathOf(relay, "desk")!.Viewers.Select(ViewerRow.Of).ToList();
        table.Apply();
        table.Apply();

        Assert.Equal(0, changes);
        Assert.Equal(2, table.Rows.Count);
    }

    [Fact]
    public void ATableWithNoRowsSaysWhichAbsenceItIs()
    {
        var unasked = Table(null);
        Assert.False(unasked.HasRows);
        Assert.Equal(Cards.ViewersUnasked, unasked.Notice);

        var watched = Table(Serving());
        Assert.False(watched.HasRows);
        Assert.Equal(Cards.ViewersNone, watched.Notice);

        var idle = new ViewerTableViewModel();
        Assert.False(idle.HasRows);
        Assert.Equal(Cards.ViewersIdle, idle.Notice);
    }

    /// <summary>
    /// A viewer in a browser or an external player sends no figures, so the relay counts it and no row stands
    /// behind it.
    /// </summary>
    [Fact]
    public void ACountWithNoReportBehindItSaysTheViewersSentNothing()
    {
        var table = Table(Serving(readers: 2));

        Assert.False(table.HasRows);
        Assert.Equal(Cards.ViewersUnreported, table.Notice);
    }
}
