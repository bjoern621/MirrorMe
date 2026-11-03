using System.Text.RegularExpressions;
using Avalonia.Media;
using ScreenShare.App.Controls;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A fill fading in from no fill keeps its own hue on the way.
/// Transparent is white at zero alpha, so a mix of the channels as written passes through a light grey.
/// </summary>
public sealed class BrushFadeTests
{
    private static readonly Color Hover = Color.Parse("#303030");

    private static Color Mid(IBrush? from, IBrush? to, double progress) =>
        Assert.IsAssignableFrom<ISolidColorBrush>(BrushFade.Between(progress, from, to)).Color;

    [Theory]
    [InlineData(0.25)]
    [InlineData(0.5)]
    [InlineData(0.75)]
    public void AFillFadingInFromTransparentKeepsItsHue(double progress)
    {
        var mid = Mid(Brushes.Transparent, new SolidColorBrush(Hover), progress);

        Assert.Equal((Hover.R, Hover.G, Hover.B), (mid.R, mid.G, mid.B));
        Assert.InRange(mid.A, (byte)(255 * progress) - 1, (byte)(255 * progress) + 1);
    }

    [Fact]
    public void AFillFadingOutToTransparentKeepsItsHue()
    {
        var mid = Mid(new SolidColorBrush(Hover), Brushes.Transparent, 0.5);

        Assert.Equal((Hover.R, Hover.G, Hover.B), (mid.R, mid.G, mid.B));
    }

    [Fact]
    public void TheFadeLandsOnBothEnds()
    {
        var from = Color.Parse("#1C1C1C");

        Assert.Equal(from, Mid(new SolidColorBrush(from), new SolidColorBrush(Hover), 0));
        Assert.Equal(Hover, Mid(new SolidColorBrush(from), new SolidColorBrush(Hover), 1));
    }

    [Fact]
    public void TwoOpaqueFillsMixChannelByChannel()
    {
        var mid = Mid(new SolidColorBrush(Colors.Black), new SolidColorBrush(Colors.White), 0.5);

        Assert.Equal(Color.FromArgb(255, 128, 128, 128), mid);
    }

    /// <summary>Avalonia's own brush transition mixes the channels as written, the flash a hover showed.</summary>
    [Fact]
    public void NoViewFadesABrushThroughAvaloniasTransition()
    {
        var stock = new Regex(@"<BrushTransition\b");
        var found = Markup.Views()
            .Where(view => stock.IsMatch(File.ReadAllText(view)))
            .Select(Path.GetFileName)
            .ToList();

        Assert.True(found.Count == 0, $"a brush fades through controls:BrushFade: {string.Join(", ", found)}");
    }
}
