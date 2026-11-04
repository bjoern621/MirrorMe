using Avalonia;

using ScreenShare.App.Features.Viewer.Tile.View;
using Xunit;

namespace ScreenShare.App.Tests;

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

    [Fact]
    public void ANarrowWideCardKeepsTheOverlay()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: true, pictureElsewhere: false, card: new Size(480, 270));

        Assert.True(chrome.Stats);
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
