using Avalonia;

using MirrorMe.App.Features.Viewer.Tile.View;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// A still pointer clears everything drawn over the picture but the stats overlay,
/// and a panel somebody reads keeps the cursor pointing at it.
/// Asserted: what a card clears, off the pointer, the tile's overlay, the host's answer and the card's size.
/// </summary>
public sealed class TileChromeTests
{
    private static readonly Size Wide = new(1280, 720);

    [Fact]
    public void APointerHoldingStillOverAPictureClearsIt()
    {
        var chrome = TileChrome.Of(pointerResting: true, showStats: false, pictureElsewhere: false, card: Wide);

        Assert.False(chrome.Stats);
        Assert.True(chrome.Resting);
        Assert.True(chrome.HidesCursor);
    }

    /// <summary>The cursor is what a reader points the figures out with.</summary>
    [Fact]
    public void TheStatsOverlayHoldsTheCursorAndClearsThePlates()
    {
        var chrome = TileChrome.Of(pointerResting: true, showStats: true, pictureElsewhere: false, card: Wide);

        Assert.True(chrome.Stats);
        Assert.True(chrome.Resting);
        Assert.False(chrome.HidesCursor);
    }

    [Fact]
    public void AStirredPointerDrawsTheChrome()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: false, pictureElsewhere: false, card: Wide);

        Assert.False(chrome.Resting);
        Assert.False(chrome.HidesCursor);
    }

    /// <summary>
    /// A stream popped out draws its figures in the window it went to,
    /// which leaves the slot behind a picture like any other.
    /// </summary>
    [Fact]
    public void ASlotWhosePictureMovedRestsWithTheOverlayOn()
    {
        var chrome = TileChrome.Of(pointerResting: true, showStats: true, pictureElsewhere: true, card: Wide);

        Assert.False(chrome.Stats);
        Assert.True(chrome.Resting);
        Assert.True(chrome.HidesCursor);
    }

    /// <summary>A portrait window a quarter of a screen wide leaves almost no picture beside the panel.</summary>
    [Fact]
    public void ANarrowUprightCardHidesTheOverlay()
    {
        var chrome = TileChrome.Of(pointerResting: true, showStats: true, pictureElsewhere: false, card: new Size(400, 700));

        Assert.False(chrome.Stats);
        Assert.True(chrome.HidesCursor);
    }

    /// <summary>A tile staged again after a turn in the strip comes back without the panel it had.</summary>
    [Fact]
    public void ANarrowCardTurnsTheOverlayOff()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: true, pictureElsewhere: false, card: new Size(270, 152));

        Assert.True(chrome.ClosesStats);
    }

    [Fact]
    public void ARoomyCardLeavesTheOverlayOn()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: true, pictureElsewhere: false, card: Wide);

        Assert.False(chrome.ClosesStats);
    }

    /// <summary>The window a stream popped out to draws its figures, so a narrow slot left behind turns nothing off.</summary>
    [Fact]
    public void ANarrowSlotWhosePictureMovedLeavesTheOverlayOn()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: true, pictureElsewhere: true, card: new Size(270, 152));

        Assert.False(chrome.ClosesStats);
    }

    [Fact]
    public void ANarrowCardWithTheOverlayOffClosesNothing()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: false, pictureElsewhere: false, card: new Size(270, 152));

        Assert.False(chrome.ClosesStats);
    }

    /// <summary>The menu row greys and drops its tick, so it never claims a panel nobody can see.</summary>
    [Fact]
    public void ANarrowUprightCardGreysTheMenuRow()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: true, pictureElsewhere: false, card: new Size(400, 700));

        Assert.False(chrome.StatsFits);
        Assert.False(chrome.StatsTicked);
    }

    [Fact]
    public void ARoomyCardTicksTheMenuRowWhileTheOverlayIsOn()
    {
        var on = TileChrome.Of(pointerResting: false, showStats: true, pictureElsewhere: false, card: Wide);
        var off = TileChrome.Of(pointerResting: false, showStats: false, pictureElsewhere: false, card: Wide);

        Assert.True(on.StatsFits);
        Assert.True(on.StatsTicked);
        Assert.True(off.StatsFits);
        Assert.False(off.StatsTicked);
    }

    /// <summary>The window a stream popped out to draws its figures, so the slot's own shape decides nothing.</summary>
    [Fact]
    public void ANarrowSlotWhosePictureMovedKeepsTheMenuRow()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: true, pictureElsewhere: true, card: new Size(400, 700));

        Assert.True(chrome.StatsFits);
        Assert.True(chrome.StatsTicked);
    }

    /// <summary>A strip tile in a portrait window, two to a row, sits as wide as the panel.</summary>
    [Fact]
    public void ANarrowWideCardHidesTheOverlay()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: true, pictureElsewhere: false, card: new Size(270, 152));

        Assert.False(chrome.Stats);
        Assert.False(chrome.StatsFits);
    }

    [Fact]
    public void AWideUprightCardKeepsTheOverlay()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: true, pictureElsewhere: false, card: new Size(720, 1280));

        Assert.True(chrome.Stats);
    }

    /// <summary>A card before its first layout has no shape to judge, and the reader's choice stands.</summary>
    [Fact]
    public void AnUnmeasuredCardKeepsTheOverlay()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: true, pictureElsewhere: false, card: default);

        Assert.True(chrome.Stats);
    }
}
