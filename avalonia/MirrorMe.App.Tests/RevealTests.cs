using Avalonia;
using Avalonia.Controls;
using MirrorMe.App.Controls;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// A fold's body takes the height that the fold's openness gives it.
/// Off the visual tree no transition runs, so the openness lands on the state asked for.
/// </summary>
public sealed class RevealTests
{
    private static Reveal Holding(double bodyHeight) => new() { Child = new Border { Width = 50, Height = bodyHeight } };

    private static double MeasuredHeight(Reveal reveal)
    {
        reveal.Measure(new Size(200, double.PositiveInfinity));
        return reveal.DesiredSize.Height;
    }

    [Fact]
    public void AClosedFoldTakesNoHeight()
    {
        var reveal = Holding(80);

        Assert.Equal(0, MeasuredHeight(reveal));
    }

    [Fact]
    public void AnOpenFoldTakesItsBodysHeight()
    {
        var reveal = Holding(80);
        reveal.IsOpen = true;

        Assert.Equal(80, MeasuredHeight(reveal));
    }

    [Fact]
    public void AFoldHalfwayOpenTakesHalfItsBodysHeight()
    {
        var reveal = Holding(80);
        reveal.IsOpen = true;
        reveal.Openness = 0.5;

        Assert.Equal(40, MeasuredHeight(reveal));
    }

    [Fact]
    public void AFoldClosingAgainTakesNoHeight()
    {
        var reveal = Holding(80);
        reveal.IsOpen = true;
        MeasuredHeight(reveal);
        reveal.IsOpen = false;

        Assert.Equal(0, MeasuredHeight(reveal));
    }
}
