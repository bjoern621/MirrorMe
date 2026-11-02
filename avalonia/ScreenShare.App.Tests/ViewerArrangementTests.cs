using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Features.Viewer.Model;
using ScreenShare.App.Features.Viewer.Tile.ViewModel;
using ScreenShare.App.Features.Viewer.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Backend describes decodes and a decode is not a tile, so no state on the wire catches a focus and a mode
/// that disagree, a window outliving its stream, or a fullscreen naming a tile that is gone
/// (<c>docs/ipc-api.md</c>).
/// </summary>
public sealed class ViewerArrangementTests
{
    /// <summary>A viewer playing the given streams, each started with no press as every ready stream is.</summary>
    private static ViewerViewModel Grid(params string[] streams) => GridOn(streams).Viewer;

    private static (ViewerViewModel Viewer, ViewerBackend Backend, Session Session) GridOn(
        params string[] streams)
    {
        var backend = new ViewerBackend(streams);
        var (viewer, session) = Flows.Watching(backend);
        return (viewer, backend, session);
    }

    private static TileViewModel Tile(ViewerViewModel viewer, string stream)
        => viewer.Tiles.Single(tile => tile.Name == stream);

    /// <summary>
    /// The join card's rows pick a way in, Discord first, and its press asks the shell for the control that sets it.
    /// Nothing is written.
    /// </summary>
    [Fact]
    public void TheJoinCardLeadsToWhereEachWayInIsSet()
    {
        var viewer = Grid();
        var asked = new List<JoinRoute>();
        viewer.JoinRequested += asked.Add;

        Assert.Equal(JoinRoute.Discord, viewer.PickedJoin);
        viewer.Join.Execute(null);
        viewer.PickKey.Execute(null);
        viewer.Join.Execute(null);

        Assert.Equal([JoinRoute.Discord, JoinRoute.Key], asked);
        Assert.Equal(JoinRoute.Key, viewer.PickedJoin);
    }

    /// <summary>The press says what it does: an unlinked machine links Discord before any channel can be read.</summary>
    [Fact]
    public void TheJoinPressNamesWhatItLeadsTo()
    {
        var viewer = Grid();

        Assert.Equal(JoinPresses.Of(JoinRoute.Discord, linked: false).Label, viewer.JoinLabel);

        viewer.PickKey.Execute(null);

        Assert.Equal(JoinPresses.Of(JoinRoute.Key, linked: false).Label, viewer.JoinLabel);
        Assert.NotEqual(
            JoinPresses.Of(JoinRoute.Discord, linked: false).Label,
            JoinPresses.Of(JoinRoute.Discord, linked: true).Label);
    }

    /// <summary>One stream on the stage and the rest in the strip under it, the first stream staged.</summary>
    [Fact]
    public void SeveralStreamsOpenOnTheStage()
    {
        var viewer = Grid("one", "two");

        Assert.Equal(LayoutMode.Focus, viewer.Mode);
        Assert.Equal("one", viewer.Focused);
        Assert.True(Tile(viewer, "one").IsFocused);
        Assert.False(Tile(viewer, "two").IsFocused);
        Assert.False(viewer.IsGrid);
        Assert.True(viewer.ShowsGridToggle);
    }

    /// <summary>A stage with an empty strip under it would take the strip's height off the one picture.</summary>
    [Fact]
    public void OneStreamFillsTheWindow()
    {
        var viewer = Grid("one");

        Assert.Equal(LayoutMode.Grid, viewer.Mode);
        Assert.Equal("", viewer.Focused);
        Assert.False(viewer.ShowsGridToggle);
    }

    /// <summary>No state holds two staged tiles, so none needs the stage given up first.</summary>
    [Fact]
    public void AStripTileSwapsOntoTheStage()
    {
        var viewer = Grid("one", "two");

        Tile(viewer, "two").ToggleFocus.Execute(null);

        Assert.Equal("two", viewer.Focused);
        Assert.Single(viewer.Tiles, tile => tile.IsFocused);
    }

    [Fact]
    public void TheGridTogglesBetweenEqualCellsAndTheStage()
    {
        var viewer = Grid("one", "two");
        Tile(viewer, "two").ToggleFocus.Execute(null);

        viewer.ToggleGrid.Execute(null);
        Assert.Equal(LayoutMode.Grid, viewer.Mode);
        Assert.Equal("", viewer.Focused);
        Assert.True(viewer.IsGrid);

        // The stage comes back on the stream it held.
        viewer.ToggleGrid.Execute(null);
        Assert.Equal(LayoutMode.Focus, viewer.Mode);
        Assert.Equal("two", viewer.Focused);
    }

    [Fact]
    public void FocusingTheStagedTileReturnsToTheGrid()
    {
        var viewer = Grid("one", "two");

        Tile(viewer, "one").ToggleFocus.Execute(null);

        Assert.Equal(LayoutMode.Grid, viewer.Mode);
        Assert.Equal("", viewer.Focused);
        Assert.True(viewer.IsGrid);
    }

    /// <summary>A tile in the grid asks for the stage and gets it.</summary>
    [Fact]
    public void FocusingATileInTheGridStagesIt()
    {
        var viewer = Grid("one", "two");
        viewer.ToggleGrid.Execute(null);

        Tile(viewer, "two").ToggleFocus.Execute(null);

        Assert.Equal(LayoutMode.Focus, viewer.Mode);
        Assert.Equal("two", viewer.Focused);
    }

    /// <summary>Slot stops the grid reflowing on a pop-out, and is what the stream comes back into.</summary>
    [Fact]
    public void APoppedOutStreamKeepsItsSlot()
    {
        var viewer = Grid("one", "two");

        Tile(viewer, "one").TogglePopOut.Execute(null);

        Assert.Equal(2, viewer.Tiles.Count);
        Assert.True(Tile(viewer, "one").IsPoppedOut);
        Assert.Contains("one", viewer.PoppedOut);
    }

    /// <summary>Focus and pop-out are different questions about one tile.</summary>
    [Fact]
    public void PoppingOutLeavesFocusAlone()
    {
        var viewer = Grid("one", "two");
        var tile = Tile(viewer, "two");

        tile.ToggleFocus.Execute(null);
        tile.TogglePopOut.Execute(null);

        Assert.Equal("two", viewer.Focused);
        Assert.Equal(LayoutMode.Focus, viewer.Mode);
        Assert.True(tile.IsPoppedOut);
    }

    /// <summary>Per-window fullscreen is what lets two streams fill two monitors.</summary>
    [Fact]
    public void FullscreenFollowsTheWindowTheStreamIsIn()
    {
        var viewer = Grid("one", "two");

        Tile(viewer, "one").ToggleFullscreen.Execute(null);
        Assert.Equal("one", viewer.Fullscreen);
        Assert.Empty(viewer.PoppedFullscreen);

        Tile(viewer, "two").TogglePopOut.Execute(null);
        Tile(viewer, "two").ToggleFullscreen.Execute(null);

        Assert.Equal("one", viewer.Fullscreen);
        Assert.Contains("two", viewer.PoppedFullscreen);
    }

    /// <summary>
    /// Every close runs this, the reconcile pass's own closes for a stream already given back included.
    /// One direction only: a toggle here asks for the window again,
    /// so leaving a pop-out from inside it opens a second one.
    /// </summary>
    [Fact]
    public void LeavingAPopOutIsOneDirectionAndIdempotent()
    {
        var viewer = Grid("one", "two");
        var tile = Tile(viewer, "one");

        tile.TogglePopOut.Execute(null);
        tile.ToggleFullscreen.Execute(null);

        tile.LeavePopOut.Execute(null);
        Assert.Empty(viewer.PoppedOut);
        Assert.False(tile.IsPoppedOut);

        // Window it filled went with it, so the stream comes back windowed.
        Assert.Empty(viewer.PoppedFullscreen);
        Assert.False(tile.IsFullscreen);

        // Close the reconcile pass performs for a stream already back in the grid.
        tile.LeavePopOut.Execute(null);
        Assert.Empty(viewer.PoppedOut);
        Assert.False(tile.IsPoppedOut);
    }

    /// <summary>
    /// Main window's fullscreen names a stream drawn in the main window.
    /// Left standing, it fills a screen with a plate saying the picture is in another window.
    /// </summary>
    [Fact]
    public void PoppingOutAFullscreenStreamGivesTheMainWindowBack()
    {
        var viewer = Grid("one", "two");
        var tile = Tile(viewer, "one");

        tile.ToggleFullscreen.Execute(null);
        tile.TogglePopOut.Execute(null);

        Assert.Equal("", viewer.Fullscreen);
        Assert.False(viewer.HasFullscreen);
        Assert.Null(viewer.FullscreenTile);

        // Fullscreen does not travel with the stream, so the window it went to arrives windowed.
        Assert.Empty(viewer.PoppedFullscreen);
        Assert.False(tile.IsFullscreen);
    }

    [Fact]
    public void TwoPoppedOutStreamsCanBothBeFullscreen()
    {
        var viewer = Grid("one", "two");

        foreach (var name in new[] { "one", "two" })
        {
            Tile(viewer, name).TogglePopOut.Execute(null);
            Tile(viewer, name).ToggleFullscreen.Execute(null);
        }

        Assert.Equal(2, viewer.PoppedFullscreen.Count);
    }

    /// <summary>
    /// Render pass derives this from the tile being gone rather than each caller undoing it,
    /// so no window outlives its tile.
    /// </summary>
    [Fact]
    public void AStreamLeavingTheGridTakesItsArrangementWithIt()
    {
        var viewer = Grid("one", "two");
        var tile = Tile(viewer, "one");

        tile.ToggleFocus.Execute(null);
        tile.TogglePopOut.Execute(null);
        tile.ToggleFullscreen.Execute(null);

        tile.Close.Execute(null);

        // The stage passes to the stream still playing, the stopped one waiting in the strip under it.
        Assert.Equal("two", viewer.Focused);
        Assert.DoesNotContain("one", viewer.PoppedOut);
        Assert.Empty(viewer.PoppedFullscreen);
    }

    /// <summary>Overlay is state of its own, not pinned hover chrome that goes off as the pointer leaves.</summary>
    [Fact]
    public void TheStatsOverlayIsItsOwnState()
    {
        var tile = Tile(Grid("one"), "one");

        Assert.False(tile.ShowStats);

        tile.ToggleStats.Execute(null);
        Assert.True(tile.ShowStats);

        tile.ToggleStats.Execute(null);
        Assert.False(tile.ShowStats);
    }

    /// <summary>
    /// Composing it while it is down costs rows per tile per sample for a panel nobody opened.
    /// Composing it on the next sample opens it empty for as long as a second, reading as a decode reporting nothing.
    /// </summary>
    [Fact]
    public void TheStatsPanelIsComposedOnlyWhileItIsUp()
    {
        var tile = Tile(Grid("one"), "one");

        Assert.Empty(tile.Stats);

        tile.ToggleStats.Execute(null);
        Assert.NotEmpty(tile.Stats);

        tile.ToggleStats.Execute(null);
        Assert.Empty(tile.Stats);
    }

    /// <summary>
    /// A row's words are fixed and its state is the tick beside them,
    /// so the bindings carry the flag and never a rewritten label.
    /// </summary>
    [Fact]
    public void TheMenuReportsWhatIsInForce()
    {
        var viewer = Grid("one", "two");
        var tile = Tile(viewer, "two");

        Assert.False(tile.IsFocused);
        Assert.False(tile.IsPoppedOut);
        Assert.False(tile.IsFullscreen);
        Assert.False(tile.ShowStats);

        tile.ToggleFocus.Execute(null);
        Assert.True(tile.IsFocused);

        tile.ToggleStats.Execute(null);
        Assert.True(tile.ShowStats);

        // Fullscreen belongs to the window a stream is drawn in,
        // so the flag follows the stream across both rather than answering for the main window.
        tile.ToggleFullscreen.Execute(null);
        Assert.True(tile.IsFullscreen);

        tile.TogglePopOut.Execute(null);
        Assert.True(tile.IsPoppedOut);
        Assert.False(tile.IsFullscreen);

        tile.ToggleFullscreen.Execute(null);
        Assert.True(tile.IsFullscreen);
    }

    /// <summary>Makes a menu row safe to press twice and the arrangement safe to hold as state.</summary>
    [Fact]
    public void EveryIntentIsAToggle()
    {
        var viewer = Grid("one");
        var tile = Tile(viewer, "one");

        tile.TogglePopOut.Execute(null);
        tile.TogglePopOut.Execute(null);

        Assert.Empty(viewer.PoppedOut);
        Assert.False(tile.IsPoppedOut);
    }

    /// <summary>Each popped-out window fills a screen of its own and answers for it.</summary>
    [Fact]
    public void LeavingFullscreenFreesTheMainWindowAlone()
    {
        var viewer = Grid("one", "two");

        Tile(viewer, "one").ToggleFullscreen.Execute(null);
        Tile(viewer, "two").TogglePopOut.Execute(null);
        Tile(viewer, "two").ToggleFullscreen.Execute(null);

        viewer.LeaveFullscreen.Execute(null);

        Assert.Equal("", viewer.Fullscreen);
        Assert.False(Tile(viewer, "one").IsFullscreen);
        Assert.Contains("two", viewer.PoppedFullscreen);
        Assert.True(Tile(viewer, "two").IsFullscreen);
    }

    /// <summary>
    /// The rail lists what the relay carries, so a dropped stream has no row left to close from.
    /// The tile's own close is the way out the reader still has.
    /// </summary>
    [Fact]
    public void ATileOutlivingItsRowStillCloses()
    {
        var (viewer, backend, session) = GridOn("one");

        backend.Carry();
        session.Start();
        viewer.Apply();

        Assert.Empty(viewer.Streams);
        var tile = Tile(viewer, "one");

        tile.Close.Execute(null);

        Assert.Empty(viewer.Tiles);
        Assert.Empty(backend.Decoding);
    }

    /// <summary>Close names the state out of the grid, so a second press finds it holds and does nothing.</summary>
    [Fact]
    public void ClosingATileIsOneDirectionAndIdempotent()
    {
        var (viewer, backend, _) = GridOn("one");
        var tile = Tile(viewer, "one");

        tile.Close.Execute(null);
        tile.Close.Execute(null);

        Assert.Empty(viewer.Tiles);
        Assert.Empty(backend.Decoding);
        Assert.False(viewer.Streams.Single(row => row.Name == "one").IsTiled);
    }

    /// <summary>Names a state rather than toggling one, so it is safe on a key that fires on every screen.</summary>
    [Fact]
    public void LeavingFullscreenNeverEntersIt()
    {
        var viewer = Grid("one");
        var tile = Tile(viewer, "one");

        viewer.LeaveFullscreen.Execute(null);
        tile.LeaveFullscreen.Execute(null);

        Assert.Equal("", viewer.Fullscreen);
        Assert.Empty(viewer.PoppedFullscreen);
        Assert.False(tile.IsFullscreen);

        viewer.LeaveFullscreen.Execute(null);

        Assert.Equal("", viewer.Fullscreen);
    }

    /// <summary>A popped-out stream stops filling its screen and stays in its window.</summary>
    [Fact]
    public void LeavingFullscreenFollowsTheWindowTheStreamIsIn()
    {
        var viewer = Grid("one");
        var tile = Tile(viewer, "one");

        tile.TogglePopOut.Execute(null);
        tile.ToggleFullscreen.Execute(null);
        tile.LeaveFullscreen.Execute(null);

        Assert.Empty(viewer.PoppedFullscreen);
        Assert.Contains("one", viewer.PoppedOut);
        Assert.False(tile.IsFullscreen);

        // And the main window's, for a stream in the grid.
        tile.TogglePopOut.Execute(null);
        tile.ToggleFullscreen.Execute(null);
        tile.LeaveFullscreen.Execute(null);

        Assert.Equal("", viewer.Fullscreen);
        Assert.False(tile.IsFullscreen);
    }

    /// <summary>
    /// A decode runs while a window draws its tile.
    /// A window closed to the tray draws nothing, so what its grid held is a relay client and a decoder running
    /// for nobody (<c>docs/viewer-architecture.md</c>, "A decode runs while a window draws it").
    /// </summary>
    [Fact]
    public void HidingTheWindowStopsEveryDecodeItsGridDrew()
    {
        var (viewer, backend, _) = GridOn("one", "two");

        viewer.SetWindowShown(false);

        Assert.Empty(viewer.Tiles);
        Assert.Empty(backend.Decoding);
    }

    /// <summary>A pop-out window stays on screen with the main window hidden, so the decode it draws runs on.</summary>
    [Fact]
    public void APoppedOutStreamKeepsItsDecodeWhileTheWindowIsHidden()
    {
        var (viewer, backend, _) = GridOn("one", "two");
        Tile(viewer, "one").TogglePopOut.Execute(null);

        viewer.SetWindowShown(false);

        Assert.Single(viewer.Tiles, tile => tile.Name == "one");
        Assert.Single(backend.Decoding, decode => decode.StreamName == "one");
        Assert.Contains("one", viewer.PoppedOut);
    }

    /// <summary>
    /// The slot a closed pop-out returns to is in a window nothing shows, and the stream is back in the grid
    /// when the window is.
    /// </summary>
    [Fact]
    public void APopOutClosedWhileTheWindowIsHiddenStopsItsDecodeUntilTheWindowIsBack()
    {
        var (viewer, backend, _) = GridOn("one");
        var tile = Tile(viewer, "one");
        tile.TogglePopOut.Execute(null);
        viewer.SetWindowShown(false);

        tile.LeavePopOut.Execute(null);

        Assert.Empty(viewer.Tiles);
        Assert.Empty(backend.Decoding);

        viewer.SetWindowShown(true);

        Assert.Single(viewer.Tiles, back => back.Name == "one");
        Assert.Single(backend.Decoding);
        Assert.Empty(viewer.PoppedOut);
    }

    /// <summary>
    /// What the grid watched is the reader's arrangement and survives the hide,
    /// so the window comes back watching it, a start per stream answered as any press is.
    /// Stating the same visibility twice asks for nothing more.
    /// </summary>
    [Fact]
    public void ShowingTheWindowAgainWatchesWhatItWatched()
    {
        var (viewer, backend, _) = GridOn("one", "two");
        viewer.SetWindowShown(false);
        Assert.Empty(viewer.Tiles);

        viewer.SetWindowShown(true);
        viewer.SetWindowShown(true);

        Assert.Equal(["one", "two"], viewer.Tiles.Select(tile => tile.Name));
        Assert.Equal(2, backend.Decoding.Count);
        Assert.Equal(4, backend.Starts);
    }

    /// <summary>
    /// A quit waits on this, so the answer is the backend's rather than the hide's own return:
    /// a stop still on its way as the process exits is a decode left running on a backend that outlives the shell.
    /// </summary>
    [Fact]
    public void PartingClosesEveryDecodeAndAnswersOnceTheBackendHas()
    {
        var (viewer, backend, _) = GridOn("one", "two");
        backend.HoldStops();

        var parted = viewer.PartAsync();

        Assert.Empty(viewer.Tiles);
        Assert.False(parted.IsCompleted);

        Answers.Now(backend.AnswerStops);

        Assert.True(parted.IsCompleted);
        Assert.Empty(backend.Decoding);
        Assert.True(viewer.PartAsync().IsCompleted);
    }

    /// <summary>
    /// A start is a round trip, and the window can hide while the answer is out.
    /// A tile added then would sit in a window nothing shows, holding the decode the hide was meant to end.
    /// </summary>
    [Fact]
    public void AStartAnsweringIntoAHiddenWindowClosesItsDecodeAgain()
    {
        var backend = new ViewerBackend("one");
        backend.HoldStarts();
        var (viewer, _) = Flows.Watching(backend);
        Assert.Single(backend.Decoding);

        viewer.SetWindowShown(false);
        Answers.Now(backend.AnswerStarts);

        Assert.Empty(viewer.Tiles);
        Assert.Empty(backend.Decoding);

        // Kept like the tiles the hide took down, so the press is honoured once there is a window for it.
        viewer.SetWindowShown(true);

        Assert.Single(viewer.Tiles);
        Assert.Single(backend.Decoding);
    }
}
