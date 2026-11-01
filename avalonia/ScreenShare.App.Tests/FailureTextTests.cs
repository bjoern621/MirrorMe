using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Features.Insights.Verdict.ViewModel;
using ScreenShare.App.Features.Insights.Model;
using ScreenShare.App.Features.Viewer.Tile.Model;
using ScreenShare.App.Features.Viewer.Tile.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A picture that never arrives and a stream that keeps relaunching are the two failures a reader meets
/// without pressing anything.
/// A screen holding "connecting" through either one describes a stream that is still coming.
/// Each carries the statement the backend made and the child's own words beside it.
/// </summary>
public sealed class FailureTextTests
{
    private static readonly Action<Action> Inline = action => action();

    private static TileViewModel Tile(IBackend backend)
        => new(TileSource.Relay("desk", "rtsp"), backend, Inline, static _ => { });

    private static TilePipeline Decode(bool live, Text? failure = null)
        => new(live, HasAudio: false, Volume: 1, Muted: false, Failure: failure);

    [Fact]
    public void ATileWithNothingLeftToArriveSaysWhyRatherThanConnecting()
    {
        var tile = Tile(new SeededBackend("linux"));

        tile.Apply(Decode(live: false, new Text { Code = TextCode.StreamLeftTheRelay }), sample: null);

        Assert.True(tile.HasNotice);
        Assert.Contains("stopped arriving", tile.Notice);
    }

    /// <summary>
    /// A relay that closed a connection states no reason of its own,
    /// so the app resolves it from the membership it holds and the tile says which of the two it was.
    /// </summary>
    [Fact]
    public void ATileClosedForLapsedMembershipSaysMembershipRatherThanTheBareClose()
    {
        var tile = Tile(new SeededBackend("linux"));

        tile.Apply(Decode(live: false, new Text { Code = TextCode.GroupMembershipLapsed }), sample: null);

        Assert.Contains("not a member of the group", tile.Notice);
    }

    /// <summary>A decode that is opening has nothing to report, a different state from one that failed.</summary>
    [Fact]
    public void ATileStillOpeningStillSaysItIsConnecting()
    {
        var tile = Tile(new SeededBackend("linux"));

        tile.Apply(Decode(live: false), sample: null);

        Assert.Equal("Connecting.", tile.Notice);
    }

    [Fact]
    public void ATileDrawingAPictureSaysNothing()
    {
        var tile = Tile(new SeededBackend("linux"));

        tile.Apply(Decode(live: true, new Text { Code = TextCode.StreamLeftTheRelay }), sample: null);

        Assert.False(tile.HasNotice);
        Assert.Equal("", tile.Notice);
    }

    /// <summary>
    /// The attempt counter says which relaunch is pending and nothing about what ended the last pipeline,
    /// the half a reader can act on.
    /// </summary>
    [Fact]
    public void ARetryingPublishSaysWhyUnderTheAttemptCounter()
    {
        var live = new PublishState.Types.Live { Publish = new PublishSettings(), StreamName = "desk" };
        live.Retry = new PublishState.Types.Retry
        {
            Attempt = 2,
            Budget = 5,
            Cause = new Text { Code = TextCode.GroupMembershipLapsed },
            Message = "srt: connection was rejected by peer",
        };

        var verdict = Verdict(Restarting(live.Retry.Cause), live);

        Assert.True(verdict.IsCaution);
        Assert.Equal("Restarting, attempt 2 of 5", verdict.Headline);
        Assert.Contains("not a member of the group", verdict.Detail);
        Assert.True(verdict.HasMessage);
        Assert.Equal("srt: connection was rejected by peer", verdict.Message);
    }

    /// <summary>A stream carrying frames has no relaunch, so the verdict quotes no pipeline.</summary>
    [Fact]
    public void AStreamThatIsNotRelaunchingCarriesNoCause()
    {
        var live = new PublishState.Types.Live { Publish = new PublishSettings(), StreamName = "desk" };

        var verdict = Verdict(new StreamTimeline { Verdict = new StreamVerdict { Level = StreamVerdict.Types.Level.Clean } }, live);

        Assert.False(verdict.IsCaution);
        Assert.False(verdict.HasMessage);
        Assert.False(verdict.HasLog);
    }

    /// <summary>
    /// A relaunch the backend named no cause for still says which attempt it is on,
    /// so the counter never waits on a statement.
    /// </summary>
    [Fact]
    public void ARelaunchWithNoStatementStillCountsItsAttempt()
    {
        var live = new PublishState.Types.Live { Publish = new PublishSettings(), StreamName = "desk" };
        live.Retry = new PublishState.Types.Retry { Attempt = 1, Budget = 3 };

        var verdict = Verdict(Restarting(null), live);

        Assert.Equal("Restarting, attempt 1 of 3", verdict.Headline);
        Assert.Equal("", verdict.Detail);
    }

    /// <summary>A run no retry recovers names its log, the one way to it from this screen.</summary>
    [Fact]
    public void AStoppedRunOffersItsLog()
    {
        var timeline = new StreamTimeline
        {
            Verdict = new StreamVerdict
            {
                Level = StreamVerdict.Types.Level.Stopped,
                Message = "vaapih264enc: could not allocate a surface",
                LogPath = "/tmp/publish.log",
            },
        };

        var verdict = Verdict(timeline, null);

        Assert.True(verdict.IsStopped);
        Assert.Equal("vaapih264enc: could not allocate a surface", verdict.Message);
        Assert.True(verdict.HasLog);
    }

    private static StreamTimeline Restarting(Text? cause) => new()
    {
        Verdict = new StreamVerdict { Level = StreamVerdict.Types.Level.Disappoints, Cause = cause },
    };

    private static VerdictViewModel Verdict(StreamTimeline timeline, PublishState.Types.Live? live)
    {
        var verdict = new VerdictViewModel(_ => Task.CompletedTask, Inline);
        verdict.Apply(timeline, InsightsSnapshot.Of(new PublishState { Live = live }, null, null));
        return verdict;
    }
}
