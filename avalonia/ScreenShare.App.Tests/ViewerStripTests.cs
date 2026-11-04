using ScreenShare.Api.V1;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Viewer.Model;
using ScreenShare.App.Features.Viewer.Tile.ViewModel;
using ScreenShare.App.Features.Viewer.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Every ready stream plays with no press, and a stream the reader stopped waits in the strip until a press.
/// The strip under the stage carries the other streams, a line per member sharing nothing, and the Grid switch.
/// </summary>
public sealed class ViewerStripTests
{
    private static (ViewerViewModel Viewer, ViewerBackend Backend, Backend.Session Session) Watching(
        params string[] streams)
        => Watching(new ViewerBackend(streams));

    private static (ViewerViewModel Viewer, ViewerBackend Backend, Backend.Session Session) Watching(
        ViewerBackend backend)
    {
        var (viewer, session) = Flows.Watching(backend);
        return (viewer, backend, session);
    }

    private static TileViewModel Tile(ViewerViewModel viewer, string stream)
        => viewer.Tiles.Single(tile => tile.Name == stream);

    private static StreamRowViewModel Waiting(ViewerViewModel viewer, string stream)
        => viewer.Strip.OfType<StreamRowViewModel>().Single(row => row.Name == stream);

    private static void Reread(ViewerViewModel viewer, Backend.Session session)
    {
        session.Start();
        viewer.Apply();
    }

    [Fact]
    public void EveryReadyStreamPlaysWithoutAPress()
    {
        var (viewer, backend, _) = Watching("one", "two");

        Assert.Equal(["one", "two"], viewer.Tiles.Select(tile => tile.Name));
        Assert.Equal(2, backend.Decoding.Count);
        Assert.Equal(2, backend.Starts);
    }

    /// <summary>A stop holds across every later reading of the relay, so the decode stays closed.</summary>
    [Fact]
    public void AStoppedStreamWaitsInTheStripForAPress()
    {
        var (viewer, backend, session) = Watching("one", "two");

        Tile(viewer, "one").Close.Execute(null);
        Reread(viewer, session);

        Assert.Equal(["two"], viewer.Tiles.Select(tile => tile.Name));
        Assert.Equal(["two"], backend.Decoding.Select(decode => decode.StreamName));
        Assert.Equal(2, backend.Starts);
        Assert.False(Waiting(viewer, "one").IsTiled);
    }

    [Fact]
    public void PlayingAStoppedStreamOpensItsDecodeAgain()
    {
        var (viewer, backend, _) = Watching("one", "two");
        Tile(viewer, "one").Close.Execute(null);

        Waiting(viewer, "one").Show.Execute(null);

        Assert.Contains(viewer.Tiles, tile => tile.Name == "one");
        Assert.Equal(2, backend.Decoding.Count);
        Assert.DoesNotContain(viewer.Strip, item => item is StreamRowViewModel);
    }

    /// <summary>
    /// A decode that ended takes its tile with it, and the stream waits in the strip for a press.
    /// Opening it again unasked would repeat whatever ended it.
    /// </summary>
    [Fact]
    public void AnEndedDecodeClosesItsTile()
    {
        var (viewer, backend, session) = Watching("one", "two");

        backend.End("one");
        Reread(viewer, session);

        Assert.Equal(["two"], viewer.Tiles.Select(tile => tile.Name));
        Assert.Equal(2, backend.Starts);
        Assert.False(Waiting(viewer, "one").IsTiled);
    }

    /// <summary>A stream that ended and started again is a new share, and plays like any other.</summary>
    [Fact]
    public void AStreamBackOnTheRelayPlaysAgain()
    {
        var (viewer, backend, session) = Watching("one", "two");
        Tile(viewer, "one").Close.Execute(null);

        backend.Carry("two");
        Reread(viewer, session);
        backend.Carry("one", "two");
        Reread(viewer, session);

        Assert.Contains(viewer.Tiles, tile => tile.Name == "one");
    }

    /// <summary>A decode of this machine's own share shows the reader what they are already looking at.</summary>
    [Fact]
    public void ThisMachinesOwnStreamWaitsForAPress()
    {
        var backend = new ViewerBackend("me", "one");
        backend.Seed.Publish = new PublishState { Live = new PublishState.Types.Live { StreamName = "me" } };

        var (viewer, _, _) = Watching(backend);

        Assert.Equal(["one"], viewer.Tiles.Select(tile => tile.Name));
        Assert.False(Waiting(viewer, "me").IsTiled);
    }

    [Fact]
    public void AMemberSharingNothingStandsInTheStrip()
    {
        var backend = new ViewerBackend("mara");
        backend.Seed.Members = new MembersState
        {
            Joined = true,
            Members =
            {
                new Member { MemberId = "m", DisplayName = "Mara", Publishing = true },
                new Member { MemberId = "k", DisplayName = "Kim" },
                new Member { MemberId = "s", DisplayName = "Self", Self = true },
            },
        };

        var (viewer, _, _) = Watching(backend);

        var hint = Assert.Single(viewer.Strip.OfType<MemberHint>());
        Assert.Equal(Cards.NotSharing("Kim"), hint.Line);
        Assert.Equal(LayoutMode.Focus, viewer.Mode);
        Assert.Equal("mara", viewer.Focused);
    }

    /// <summary>Equal cells are a choice while two streams play, and the stage is left from the strip's end.</summary>
    [Fact]
    public void TheGridSwitchEndsTheStripWhileTwoStreamsPlay()
    {
        var (viewer, _, _) = Watching("one", "two");

        Assert.IsType<GridSwitch>(viewer.Strip[^1]);

        viewer.ToggleGrid.Execute(null);

        Assert.Equal(LayoutMode.Grid, viewer.Mode);
        Assert.DoesNotContain(viewer.Strip, item => item is GridSwitch);
    }

    [Fact]
    public void OneStreamAloneTakesNoSwitch()
    {
        var (viewer, _, _) = Watching("one");

        Assert.DoesNotContain(viewer.Strip, item => item is GridSwitch);
    }

    /// <summary>Why the group refused this machine, in the group service's own words.</summary>
    [Fact]
    public void AGroupRefusalIsStated()
    {
        var backend = new ViewerBackend();
        backend.Seed.Members = new MembersState { Refusal = new Text { Code = TextCode.GroupNameTaken } };

        var (viewer, _, _) = Watching(backend);

        Assert.True(viewer.HasGroupRefusal);
        Assert.Contains("holds that name", viewer.GroupRefusal);
    }

    /// <summary>
    /// A window too narrow for a strip beside the stage stacks the strip under it, two to a row.
    /// The lines for members sharing nothing leave the strip for a list under it.
    /// </summary>
    [Fact]
    public void ANarrowWindowStacksTheStripUnderTheStage()
    {
        var backend = new ViewerBackend("one", "two");
        backend.Seed.Members = new MembersState { Joined = true, Members = { new Member { MemberId = "k", DisplayName = "Kim" } } };
        var (viewer, _, _) = Watching(backend);

        viewer.SetWindowWidth(360);

        Assert.Equal(LayoutMode.Stack, viewer.Mode);
        Assert.DoesNotContain(viewer.Strip, item => item is MemberHint);
        Assert.Equal([Cards.NotSharing("Kim")], viewer.HintsBelow.Select(hint => hint.Line));
    }
}
