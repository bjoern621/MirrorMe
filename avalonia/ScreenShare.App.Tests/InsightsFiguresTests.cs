using ScreenShare.Api.V1;
using ScreenShare.App.Features.Insights.Model;
using ScreenShare.App.Features.Shell.Model;
using ScreenShare.App.Features.Shell.NavStrip.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Every figure on the live screen is a measurement some other side took.
///
/// A figure with nothing behind it looks measured: a timer reading the same eight digits whatever is publishing,
/// a window label naming a span the plot does not cover, a rule marking a ceiling wherever the design put it.
/// Each test states a reading and asserts the screen's figure came out of it, which a fixed number cannot do.
/// </summary>
public sealed class InsightsFiguresTests
{
    /// <summary>Rate over the last interval, at a point in the run.</summary>
    private static PublishStats Sample(double mbps, double timeSec) => new()
    {
        InstMbps = mbps,
        TimeSec = timeSec,
    };

    /// <summary>Sample from a run that has been timed and has measured no rate.</summary>
    private static PublishStats Timed(double timeSec) => new()
    {
        TimeSec = timeSec,
    };

    /// <summary>
    /// Running stream, held to a rate where one is passed.
    /// The ceiling is the backend's reading of what bounds this encoder, not the settings field beside it:
    /// a quality target on an encoder that holds none is unbounded whatever the field carries.
    /// </summary>
    private static PublishState Live(int? ceilingMbps = null)
    {
        var live = new PublishState.Types.Live { Publish = new PublishSettings(), StreamName = "desk" };
        if (ceilingMbps is { } ceiling)
        {
            live.RateCeilingMbps = ceiling;
        }

        return new PublishState { Live = live };
    }

    [Fact]
    public void TheSharingTimerIsTheEncodersOwnClock()
    {
        var reading = InsightsSnapshot.Of(Live(), Sample(4, 3_671), null);

        Assert.Equal("01:01:11", reading.Elapsed);
    }

    [Fact]
    public void TheSharingTimerIsUnmeasuredBeforeTheFirstSample()
    {
        var reading = InsightsSnapshot.Of(Live(), null, null);

        Assert.True(reading.IsLive);
        Assert.Equal(Figure.NoValue, reading.Elapsed);
    }

    /// <summary>
    /// The screen reports the stream that has ended as well as the running one,
    /// so the segment that opens it is not a control the running state takes away.
    /// </summary>
    [Fact]
    public void TheStripReachesInsightsWithNothingPublishing()
    {
        var asked = new List<Destination>();
        var strip = new NavStripViewModel(asked.Add);

        strip.Show(Destination.Setup);
        strip.SelectedTab = strip.Tabs.Single(tab => tab.Value == Destination.Insights);

        Assert.Equal(Destination.Insights, Assert.Single(asked));
    }

    /// <summary>
    /// A live publish and a snapshot name one stream, so a machine in a group reads its own figures:
    /// the prefix the group publishes under is the backend's and reaches neither string.
    /// </summary>
    [Fact]
    public void AGroupedPublishReadsItsOwnRelayRow()
    {
        var relay = new RelayStatus { Reachable = true };
        relay.Paths.Add(new RelayPath { Name = "bjoern/monitor-0", Readers = 2 });

        var live = new PublishState
        {
            Live = new PublishState.Types.Live { Publish = new PublishSettings(), StreamName = "bjoern/monitor-0" },
        };

        Assert.Equal(2, InsightsSnapshot.Of(live, null, relay).Viewers);
    }

    /// <summary>Sample from a run that timed a frame through the encoder.</summary>
    private static PublishStats Encoded(double transitMs, double timeSec) => new()
    {
        TransitMs = transitMs,
        TimeSec = timeSec,
    };

    /// <summary>One reader on this stream's path.</summary>
    private static RelayStatus Roster()
    {
        var reader = new RelayReader { Transport = "srt" };
        var path = new RelayPath { Name = "desk", Readers = 1 };
        path.ReaderRoster.Add(reader);

        var relay = new RelayStatus { Reachable = true };
        relay.Paths.Add(path);
        return relay;
    }

    /// <summary>
    /// One second is short enough for a healthy stream to measure nothing: an encoder that emitted no frame
    /// over the interval timed none, and the row would otherwise alternate between a figure and an ellipsis.
    /// </summary>
    [Fact]
    public void AFigureNoPassMeasuredReadsTheLastOneThatDid()
    {
        var held = new HeldFigures();

        held.Fill(InsightsSnapshot.Of(Live(), Encoded(20.5, 12), null));
        var reading = held.Fill(InsightsSnapshot.Of(Live(), Timed(13), null));

        Assert.Equal(20.5, reading.EncodeMs);
    }

    /// <summary>
    /// A stream that stopped measures nothing at all, and the next one is a different stream.
    /// A figure carried across either would name this stream by the last one's reading.
    /// </summary>
    [Fact]
    public void AStoppedStreamHoldsNothingForTheNextOne()
    {
        var held = new HeldFigures();
        held.Fill(InsightsSnapshot.Of(Live(), Encoded(20.5, 12), null));

        var idle = held.Fill(InsightsSnapshot.Of(new PublishState(), null, null));

        Assert.Null(idle.EncodeMs);

        var next = held.Fill(InsightsSnapshot.Of(Live(), Timed(1), null));

        Assert.Null(next.EncodeMs);
    }

    /// <summary>
    /// A poll that landed on nothing states no path, which says nothing about the viewers on it:
    /// same stream, and the readers on it were counted a second ago.
    /// </summary>
    [Fact]
    public void APollThatNamesNoPathHoldsWhatTheLastOneNamed()
    {
        var held = new HeldFigures();
        held.Fill(InsightsSnapshot.Of(Live(), Timed(12), Roster()));

        var reading = held.Fill(InsightsSnapshot.Of(Live(), Timed(13), null));

        Assert.Equal(1, reading.Viewers);
    }

    /// <summary>
    /// A stream between attempts has no pipeline measuring anything, and the pill says so.
    /// Figures left standing beside that would read as a stream carrying frames.
    /// </summary>
    [Fact]
    public void AStreamWaitingOutARetryHoldsNoFigure()
    {
        var held = new HeldFigures();
        held.Fill(InsightsSnapshot.Of(Live(), Encoded(20.5, 12), null));

        var live = Live();
        live.Live.Retry = new PublishState.Types.Retry { Attempt = 1, Budget = 3 };
        var reading = held.Fill(InsightsSnapshot.Of(live, Timed(13), null));

        Assert.Null(reading.EncodeMs);
    }
}
