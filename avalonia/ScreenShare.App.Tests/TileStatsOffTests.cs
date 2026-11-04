using ScreenShare.App.Features.Viewer.Tile.Model;
using ScreenShare.App.Features.Viewer.Tile.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// The stats overlay goes off for good once a card has no room for it.
/// Asserted: hiding names a state, so a second call or a call on a closed overlay changes nothing.
/// </summary>
public sealed class TileStatsOffTests
{
    private static TileViewModel Tile()
        => new(TileSource.Relay("desk", "rtsp"), new SeededBackend("linux"), static action => action(), static _ => { });

    [Fact]
    public void HidingTurnsTheOverlayOff()
    {
        var tile = Tile();
        tile.ToggleStats.Execute(null);

        tile.HideStats();

        Assert.False(tile.ShowStats);
    }

    [Fact]
    public void HidingAClosedOverlayChangesNothing()
    {
        var tile = Tile();
        var notified = 0;
        tile.PropertyChanged += (_, _) => notified++;

        tile.HideStats();
        tile.HideStats();

        Assert.False(tile.ShowStats);
        Assert.Equal(0, notified);
    }
}
