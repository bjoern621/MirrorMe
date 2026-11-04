using Avalonia.Controls;
using ScreenShare.App.Controls;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A dialog's ground stays drawn while it fades out, and takes no press once it is closing.
/// Off the visual tree no transition runs, so the openness lands on the state asked for.
/// </summary>
public sealed class OverlayTests
{
    private static (Overlay Overlay, Border Body) Holding()
    {
        var body = new Border();
        return (new Overlay { Child = body }, body);
    }

    [Fact]
    public void AClosedOverlayDrawsNothing()
    {
        var (_, body) = Holding();

        Assert.False(body.IsVisible);
    }

    [Fact]
    public void AnOpenOverlayDrawsItsBodyWhole()
    {
        var (overlay, body) = Holding();
        overlay.IsOpen = true;

        Assert.True(body.IsVisible);
        Assert.Equal(1, overlay.Opacity);
        Assert.True(overlay.IsHitTestVisible);
    }

    [Fact]
    public void AClosingOverlayStaysDrawnAndLetsPressesThrough()
    {
        var (overlay, body) = Holding();
        overlay.IsOpen = true;
        overlay.IsOpen = false;
        overlay.Openness = 0.5;

        Assert.True(body.IsVisible);
        Assert.Equal(0.5, overlay.Opacity);
        Assert.False(overlay.IsHitTestVisible);
    }

    [Fact]
    public void AClosedOverlayHidesItsBodyOnceTheFadeEnds()
    {
        var (overlay, body) = Holding();
        overlay.IsOpen = true;
        overlay.IsOpen = false;

        Assert.False(body.IsVisible);
        Assert.Equal(0, overlay.Opacity);
    }
}
