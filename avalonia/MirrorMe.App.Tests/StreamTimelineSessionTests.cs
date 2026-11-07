using MirrorMe.Api.V1;
using MirrorMe.App.Backend;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// The backend judges the publish run, and the session holds its last answer whole:
/// read on load, replaced by every announcement.
/// </summary>
public sealed class StreamTimelineSessionTests
{
    private static readonly Action<Action> Inline = action => action();

    private static StreamTimeline At(StreamVerdict.Types.Level level)
        => new() { Verdict = new StreamVerdict { Level = level } };

    [Fact]
    public async Task TheLoadedTimelineIsReplacedByAnAnnouncedOne()
    {
        var backend = new SeededBackend("linux") { Timeline = At(StreamVerdict.Types.Level.Clean) };
        backend.Announcements.Add(new Event { StreamTimeline = At(StreamVerdict.Types.Level.Disappoints) });
        var session = new Session(backend, Inline);

        _ = session.Start();
        var deadline = DateTime.UtcNow.AddSeconds(5);
        while (session.Timeline?.Verdict.Level != StreamVerdict.Types.Level.Disappoints && DateTime.UtcNow < deadline)
        {
            await Task.Delay(1);
        }

        Assert.Equal(StreamVerdict.Types.Level.Disappoints, session.Timeline?.Verdict.Level);
    }
}
