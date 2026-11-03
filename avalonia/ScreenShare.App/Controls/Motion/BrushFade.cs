using Avalonia.Animation;
using Avalonia.Media;
using Avalonia.Media.Immutable;
using ScreenShare.App.Contracts;

namespace ScreenShare.App.Controls;

/// <summary>
/// Eases a fill from one colour to another, each colour weighted by its own alpha.
/// Avalonia's <see cref="BrushTransition"/> mixes the channels as written.
/// Transparent is white at zero alpha, so a hover from no fill to a dark one passes through light grey and flashes.
/// A brush other than a solid colour switches at the midpoint.
/// </summary>
public sealed class BrushFade : InterpolatingTransitionBase<IBrush?>
{
    protected override IBrush? Interpolate(double progress, IBrush? from, IBrush? to) => Between(progress, from, to);

    /// <summary>The fill at <paramref name="progress"/> from 0 to 1. A missing brush reads as no fill.</summary>
    public static IBrush? Between(double progress, IBrush? from, IBrush? to)
    {
        Assert.That(progress is >= 0 and <= 1, "a fade runs from 0 to 1, the easing never overshooting", progress);

        if (from is not (null or ISolidColorBrush) || to is not (null or ISolidColorBrush))
        {
            return progress < 0.5 ? from : to;
        }

        var (a, b) = (Premultiplied(from), Premultiplied(to));
        var alpha = Mix(a.A, b.A, progress);
        if (alpha <= 0)
        {
            return new ImmutableSolidColorBrush(Colors.Transparent);
        }

        return new ImmutableSolidColorBrush(Color.FromArgb(
            Channel(alpha),
            Channel(Mix(a.R, b.R, progress) / alpha),
            Channel(Mix(a.G, b.G, progress) / alpha),
            Channel(Mix(a.B, b.B, progress) / alpha)));
    }

    /// <summary>Channels from 0 to 1, each colour channel multiplied by the alpha the brush draws at.</summary>
    private static (double A, double R, double G, double B) Premultiplied(IBrush? brush)
    {
        if (brush is not ISolidColorBrush solid)
        {
            return (0, 0, 0, 0);
        }

        var alpha = solid.Color.A / 255.0 * solid.Opacity;
        return (alpha, solid.Color.R / 255.0 * alpha, solid.Color.G / 255.0 * alpha, solid.Color.B / 255.0 * alpha);
    }

    private static double Mix(double from, double to, double progress) => from + (to - from) * progress;

    private static byte Channel(double share) => (byte)Math.Round(Math.Clamp(share, 0, 1) * 255);
}
