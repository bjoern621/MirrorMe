using Avalonia;
using MirrorMe.App.Controls;
using Xunit;

namespace MirrorMe.App.Tests;

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
    public void ANotchDownMovesOneNotchStep()
    {
        var offset = WheelScroll.Offset(new Vector(0, 0), new Vector(0, -1), Tall, Viewport, false, ScrollSource.Wheel);

        Assert.Equal(new Vector(0, WheelScroll.NotchStep), offset);
    }

    [Fact]
    public void AWheelMovesFartherPerNotchThanASmoothScrollPerUnit() =>
        Assert.True(
            WheelScroll.StepOf(ScrollSource.Wheel, new Vector(0, 1)) > WheelScroll.StepOf(ScrollSource.Finger, new Vector(0, 1)));

    /// <summary>A high-resolution wheel reports fractions of a notch, and each moves its share of a step.</summary>
    [Fact]
    public void AFractionOfANotchMovesItsShareOfAStep()
    {
        var offset = WheelScroll.Offset(new Vector(0, 100), new Vector(0, 0.5), Tall, Viewport, false, ScrollSource.Wheel);

        Assert.Equal(new Vector(0, 100 - (WheelScroll.NotchStep / 2)), offset);
    }

    /// <summary>
    /// A named source decides alone, a touchpad delta landing on a whole number included.
    /// X11 and Windows name no source, and report a wheel in whole notches and a touchpad in fractions.
    /// </summary>
    [Theory]
    [InlineData(ScrollSource.Wheel, 1, WheelScroll.NotchStep)]
    [InlineData(ScrollSource.Wheel, 0.125, WheelScroll.NotchStep)]
    [InlineData(ScrollSource.WheelTilt, 1, WheelScroll.NotchStep)]
    [InlineData(ScrollSource.Finger, 1, WheelScroll.SmoothStep)]
    [InlineData(ScrollSource.Finger, 0.5, WheelScroll.SmoothStep)]
    [InlineData(ScrollSource.Continuous, 2, WheelScroll.SmoothStep)]
    [InlineData(ScrollSource.Unknown, 1, WheelScroll.NotchStep)]
    [InlineData(ScrollSource.Unknown, -3, WheelScroll.NotchStep)]
    [InlineData(ScrollSource.Unknown, 0.5, WheelScroll.SmoothStep)]
    public void TheSourceSetsTheStep(ScrollSource source, double delta, double step) =>
        Assert.Equal(step, WheelScroll.StepOf(source, new Vector(0, delta)));

    [Fact]
    public void AnUnnamedScrollWithAFractionOnEitherAxisMovesASmoothStep() =>
        Assert.Equal(WheelScroll.SmoothStep, WheelScroll.StepOf(ScrollSource.Unknown, new Vector(0.5, 1)));

    [Fact]
    public void TheOffsetStopsAtEitherEnd()
    {
        var bottom = Tall.Height - Viewport.Height;

        Assert.Equal(
            new Vector(0, 0), WheelScroll.Offset(new Vector(0, 5), new Vector(0, 3), Tall, Viewport, false, ScrollSource.Wheel));
        Assert.Equal(
            new Vector(0, bottom),
            WheelScroll.Offset(new Vector(0, bottom - 5), new Vector(0, -3), Tall, Viewport, false, ScrollSource.Wheel));
    }

    /// <summary>An axis whose content fits has nothing to scroll, so a delta along it moves nothing.</summary>
    [Fact]
    public void AnAxisThatFitsStaysPut()
    {
        var offset = WheelScroll.Offset(new Vector(0, 0), new Vector(-1, -1), Tall, Viewport, false, ScrollSource.Wheel);

        Assert.Equal(0, offset.X);
    }

    /// <summary>Shift turns an up-and-down wheel sideways, for a mouse that has no sideways wheel.</summary>
    [Fact]
    public void ShiftTurnsAnUpAndDownNotchSideways()
    {
        var offset = WheelScroll.Offset(new Vector(0, 0), new Vector(0, -1), Wide, Viewport, true, ScrollSource.Wheel);

        Assert.Equal(new Vector(WheelScroll.NotchStep, 0), offset);
    }
}
