using System.Collections.Specialized;
using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Insights.Model;
using ScreenShare.App.Features.Insights.ViewerTable.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Row per reader the relay named, cell per figure it stated about that reader.
/// Unstated figure reads absent, never zero, so a viewer nobody timed never reads as a viewer with a perfect link.
/// </summary>
public sealed class ViewerRosterTests
{
    /// <summary>SRT reader: only leg the relay times and states a loss rate on.</summary>
    private static RelayReader Srt(string address, double rttMs, double lossPercent, ulong dropped = 0) => new()
    {
        Type = "srtConn",
        Id = $"id-{address}",
        Transport = "srt",
        RemoteAddr = address,
        Joined = "2026-08-09T22:01:12.0000000+00:00",
        BytesSent = 373004,
        RttMs = rttMs,
        LossPercent = lossPercent,
        PacketsSent = 385,
        PacketsLost = 0,
        PacketsDropped = dropped,
        FramesDiscarded = 0,
    };

    /// <summary>RTMP reader: bytes and the relay's own discards, nothing about the line.</summary>
    private static RelayReader Rtmp(string address, ulong discarded = 0) => new()
    {
        Type = "rtmpConn",
        Id = $"id-{address}",
        Transport = "rtmp",
        RemoteAddr = address,
        Joined = "2026-08-09T22:01:15.0000000+00:00",
        BytesSent = 358672,
        FramesDiscarded = discarded,
    };

    private static RelayStatus Serving(params RelayReader[] readers)
    {
        var path = new RelayPath { Name = "desk", Ready = true, Readers = readers.Length };
        path.ReaderRoster.AddRange(readers);

        var relay = new RelayStatus { Reachable = true };
        relay.Paths.Add(path);
        return relay;
    }

    private static PublishState Live() => new()
    {
        Live = new PublishState.Types.Live { Publish = new PublishSettings(), StreamName = "desk" },
    };

    private static ViewerTableViewModel Table(RelayStatus? relay)
    {
        var path = InsightsSnapshot.PathOf(relay, "desk");
        var table = new ViewerTableViewModel
        {
            Reported = path is null ? [] : path.ReaderRoster.Select(ViewerRow.Of).ToList(),
            Readers = path?.Readers,
            IsLive = true,
        };

        return table;
    }

    [Fact]
    public void ARowIsBuiltFromTheReaderTheRelayNamed()
    {
        var table = Table(Serving(Srt("10.0.0.4:52157", rttMs: 18.4, lossPercent: 0.25)));

        Assert.True(table.HasRows);
        var row = Assert.Single(table.Rows);

        Assert.Equal("10.0.0.4:52157", row.Name);
        Assert.Equal("18", row.Rtt);
        Assert.Equal("0.25", row.Loss);
        Assert.Equal("srt", row.Via);
        Assert.Equal("0", row.Dropped);
    }

    [Fact]
    public void AFigureTheLegDoesNotReportReadsAsAbsentRatherThanAsZero()
    {
        var table = Table(Serving(Rtmp("10.0.0.9:55372")));

        var row = Assert.Single(table.Rows);

        // Nothing times an RTMP reader and nothing states its loss.
        Assert.Equal(Figure.NoValue, row.Rtt);
        Assert.Equal(Figure.NoValue, row.Loss);

        // Measured zero is a measurement, and prints as one.
        Assert.Equal("0", row.Dropped);
        Assert.Equal("rtmp", row.Via);
        Assert.False(row.IsStruggling);
    }

    [Fact]
    public void AReaderTheRelayDescribedNowhereIsStillNamed()
    {
        // Named on the path, absent from every per-protocol list: a relay with that listener off.
        var unmeasured = new RelayReader { Type = "moqSession", Id = "5c1f", Transport = "moq" };

        var table = Table(Serving(unmeasured));

        var row = Assert.Single(table.Rows);
        Assert.Equal("5c1f", row.Name);
        Assert.Equal(Figure.NoValue, row.Rtt);
        Assert.Equal(Figure.NoValue, row.Loss);
        Assert.Equal(Figure.NoValue, row.Dropped);
        Assert.Equal("moq", row.Via);
        Assert.False(row.IsStruggling);
    }

    /// <summary>
    /// Relay naming a reader nothing is an Umgebungsfehler, so the row renders unnameable rather than asserting.
    /// Somebody is connected either way.
    /// </summary>
    [Fact]
    public void AReaderTheRelayNamedNothingStillRenders()
    {
        var table = Table(Serving(new RelayReader()));

        var row = Assert.Single(table.Rows);
        Assert.Equal(Figure.NoValue, row.Name);
        Assert.Equal(Figure.NoValue, row.Via);
        Assert.False(row.IsStruggling);
        Assert.Equal(1, table.Readers);
    }

    [Fact]
    public void OnlyAMeasuredFigureOverItsLimitMakesARowStruggle()
    {
        var table = Table(Serving(
            Srt("10.0.0.1:1", rttMs: 20, lossPercent: 0.1),      // healthy
            Srt("10.0.0.2:2", rttMs: 20, lossPercent: 7.5),      // loss past its limit
            Srt("10.0.0.3:3", rttMs: 480, lossPercent: 0),       // round trip past its limit
            Srt("10.0.0.4:4", rttMs: 20, lossPercent: 0, 12),    // dropped at all
            Rtmp("10.0.0.5:5")));                                // untimed, so never struggling

        Assert.Equal(5, table.Rows.Count);
        Assert.Equal(3, table.StrugglingCount);
        Assert.False(table.Rows[0].IsStruggling);
        Assert.True(table.Rows[1].IsStruggling);
        Assert.True(table.Rows[2].IsStruggling);
        Assert.True(table.Rows[3].IsStruggling);
        Assert.False(table.Rows[4].IsStruggling);
    }

    [Fact]
    public void OneRowEndsTheTableAndCarriesNoSeparator()
    {
        var table = Table(Serving(Srt("10.0.0.1:1", 20, 0), Srt("10.0.0.2:2", 20, 0), Rtmp("10.0.0.3:3")));

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
    public void AnUnchangedRosterRenderedAgainTouchesNothing()
    {
        var relay = Serving(Srt("10.0.0.1:1", 20, 0.5), Rtmp("10.0.0.2:2"));
        var table = Table(relay);

        var changes = 0;
        ((INotifyCollectionChanged)table.Rows).CollectionChanged += (_, _) => changes++;

        // Same roster again, through the path the render pass reads it by.
        table.Reported = InsightsSnapshot.PathOf(relay, "desk")!.ReaderRoster.Select(ViewerRow.Of).ToList();
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
        Assert.Contains("has not been asked", unasked.Notice);

        var watched = Table(Serving());
        Assert.False(watched.HasRows);
        Assert.Contains("Nobody is connected", watched.Notice);

        // Card is drawn with nothing publishing too, a third absence and not an empty roster.
        var idle = new ViewerTableViewModel();
        Assert.False(idle.HasRows);
        Assert.Contains("Nothing is publishing", idle.Notice);
    }

    /// <summary>
    /// A member reads the group service's index, which counts the readers of a path and leaves their names
    /// at the service (<c>docs/ipc-api.md</c>).
    /// The count then stands with no rows behind it, an unavailable roster wearing the look of an empty one,
    /// and this card is where that difference is stated.
    /// </summary>
    [Fact]
    public void ACountWithNoRosterBehindItSaysTheViewersAreUnnamed()
    {
        var path = new RelayPath { Name = "desk", Ready = true, Readers = 2 };
        var relay = new RelayStatus { Reachable = true };
        relay.Paths.Add(path);

        var table = Table(relay);

        Assert.False(table.HasRows);
        Assert.Equal(Cards.ViewersUnnamed, table.Notice);
    }
}
