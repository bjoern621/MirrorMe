using ScreenShare.App.Features.Viewer.Tile.View;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A still pointer clears everything drawn over the picture but the stats overlay,
/// and a panel somebody reads keeps the cursor pointing at it.
/// Asserted: what a card clears, off the pointer, the tile's overlay and the host's answer.
/// </summary>
public sealed class TileChromeTests
{
    [Fact]
    public void APointerHoldingStillOverAPictureClearsIt()
    {
        var chrome = TileChrome.Of(pointerResting: true, showStats: false, pictureElsewhere: false);

        Assert.False(chrome.Stats);
        Assert.True(chrome.Resting);
        Assert.True(chrome.HidesCursor);
    }

    /// <summary>The cursor is what a reader points the figures out with.</summary>
    [Fact]
    public void TheStatsOverlayHoldsTheCursorAndClearsThePlates()
    {
        var chrome = TileChrome.Of(pointerResting: true, showStats: true, pictureElsewhere: false);

        Assert.True(chrome.Stats);
        Assert.True(chrome.Resting);
        Assert.False(chrome.HidesCursor);
    }

    [Fact]
    public void AStirredPointerDrawsTheChrome()
    {
        var chrome = TileChrome.Of(pointerResting: false, showStats: false, pictureElsewhere: false);

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
        var chrome = TileChrome.Of(pointerResting: true, showStats: true, pictureElsewhere: true);

        Assert.False(chrome.Stats);
        Assert.True(chrome.Resting);
        Assert.True(chrome.HidesCursor);
    }
}
