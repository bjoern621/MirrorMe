using Avalonia;
using ScreenShare.App.Controls;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Where one wheel event leaves a scroll viewer, every viewer in the app scrolling through it
/// (<c>Design/Scrolling.axaml</c>).
/// A wheel reports a notch as a delta of one, positive towards the top and the left edge.
/// </summary>
public sealed class WheelScrollTests
{
    private static readonly Size Viewport = new(300, 200);
    private static readonly Size Tall = new(300, 1000);
    private static readonly Size Wide = new(1000, 200);

    [Fact]
    public void ANotchDownMovesOneStep()
    {
        var offset = WheelScroll.Offset(new Vector(0, 0), new Vector(0, -1), Tall, Viewport, shift: false);

        Assert.Equal(new Vector(0, WheelScroll.Step), offset);
    }

    /// <summary>A touchpad reports fractions of a notch, and each moves its share of a step.</summary>
    [Fact]
    public void AFractionOfANotchMovesItsShareOfAStep()
    {
        var offset = WheelScroll.Offset(new Vector(0, 100), new Vector(0, 0.5), Tall, Viewport, shift: false);

        Assert.Equal(new Vector(0, 100 - (WheelScroll.Step / 2)), offset);
    }

    [Fact]
    public void TheOffsetStopsAtEitherEnd()
    {
        var bottom = Tall.Height - Viewport.Height;

        Assert.Equal(new Vector(0, 0), WheelScroll.Offset(new Vector(0, 5), new Vector(0, 3), Tall, Viewport, false));
        Assert.Equal(
            new Vector(0, bottom), WheelScroll.Offset(new Vector(0, bottom - 5), new Vector(0, -3), Tall, Viewport, false));
    }

    /// <summary>An axis whose content fits has nothing to scroll, so a delta along it moves nothing.</summary>
    [Fact]
    public void AnAxisThatFitsStaysPut()
    {
        var offset = WheelScroll.Offset(new Vector(0, 0), new Vector(-1, -1), Tall, Viewport, shift: false);

        Assert.Equal(0, offset.X);
    }

    /// <summary>Shift turns an up-and-down wheel sideways, for a mouse that has no sideways wheel.</summary>
    [Fact]
    public void ShiftTurnsAnUpAndDownNotchSideways()
    {
        var offset = WheelScroll.Offset(new Vector(0, 0), new Vector(0, -1), Wide, Viewport, shift: true);

        Assert.Equal(new Vector(WheelScroll.Step, 0), offset);
    }
}
