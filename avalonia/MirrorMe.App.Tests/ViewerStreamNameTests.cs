using MirrorMe.Api.V1;
using MirrorMe.App.Backend;
using MirrorMe.App.Features.Viewer.ViewModel;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// A group is a path prefix, so every entry of a member's list carries the same one and it separates none of them.
/// The backend takes it off before the snapshot crosses and puts it back on to reach the relay,
/// so a row prints and opens one string.
/// </summary>
public sealed class ViewerStreamNameTests
{
    private static (ViewerViewModel Viewer, SeededBackend Backend) Viewing(RelayPath path)
    {
        var backend = new SeededBackend("linux")
        {
            Relay = new RelayStatus { Reachable = true, Paths = { path } },
        };

        var session = new Session(backend, static action => action());
        session.Start();

        var viewer = Flows.Viewer(backend, session);
        viewer.Apply();
        return (viewer, backend);
    }

    [Fact]
    public void AnEntryPrintsTheNameTheSnapshotCarries()
    {
        var (viewer, _) = Viewing(new RelayPath { Name = "bjoern/monitor-0", Ready = true });

        var row = Assert.Single(viewer.Streams);
        Assert.Equal("bjoern/monitor-0", row.Label);
        Assert.Equal("bjoern/monitor-0", row.Name);
    }

    /// <summary>What a row prints is what a decode is asked for, the prefix being the backend's alone.</summary>
    [Fact]
    public void PlayingOneAsksForTheNameItPrints()
    {
        var (_, backend) = Viewing(new RelayPath { Name = "bjoern/monitor-0", Ready = true });

        Assert.Equal("bjoern/monitor-0", Assert.Single(backend.Decoded).StreamName);
    }

    /// <summary>A relay that authenticates nobody carries bare names, with no claim to lead them.</summary>
    [Fact]
    public void APathUnderNoPrefixPrintsAsItArrived()
    {
        var (viewer, _) = Viewing(new RelayPath { Name = "desk", Ready = true });

        var row = Assert.Single(viewer.Streams);
        Assert.Equal("desk", row.Label);
        Assert.Equal("desk", row.Name);
    }
}
