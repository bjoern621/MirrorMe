using Avalonia;
using ScreenShare.App.Contracts;

namespace ScreenShare.App.Features.Viewer.Tile.Model;

/// <summary>
/// How far a card is zoomed into its picture, and which spot of the picture sits in the card's middle.
///
/// The spot is a fraction of the picture, so a card resized by a focus or a window drag keeps the same spot in view.
/// Every result keeps the picture covering the whole card: a zoom near a corner or a drag past the edge stops at it.
/// </summary>
/// <param name="Scale">Magnification, 1 for the picture as it is.</param>
/// <param name="CenterX">Spot in the card's middle, as a fraction of the picture's width.</param>
/// <param name="CenterY">Spot in the card's middle, as a fraction of the picture's height.</param>
public readonly record struct TileZoom(double Scale, double CenterX, double CenterY)
{
    /// <summary>Deepest zoom: past it a decoded pixel covers more of the card than anything in it can be read by.</summary>
    public const double MaxScale = 8;

    public static readonly TileZoom None = new(1, 0.5, 0.5);

    public bool IsZoomed => Scale > 1;

    /// <summary>Part of the picture the card shows, in fractions of the picture's width and height.</summary>
    public Rect Visible => new(CenterX - 1 / (2 * Scale), CenterY - 1 / (2 * Scale), 1 / Scale, 1 / Scale);

    /// <summary>Zooms by <paramref name="factor"/> and keeps the spot under <paramref name="anchor"/> under it.</summary>
    public TileZoom At(Point anchor, Size card, double factor)
    {
        Assert.That(double.IsFinite(factor) && factor > 0, "a zoom step multiplies the scale by a positive factor", factor);
        AssertDrawn(card);

        var scale = Math.Clamp(Scale * factor, 1, MaxScale);
        var offset = Offset(card);
        var pictureX = (anchor.X - offset.X) / Scale;
        var pictureY = (anchor.Y - offset.Y) / Scale;

        return Covering(
            scale,
            (card.Width / 2 - (anchor.X - pictureX * scale)) / (card.Width * scale),
            (card.Height / 2 - (anchor.Y - pictureY * scale)) / (card.Height * scale));
    }

    /// <summary>Moves the picture by <paramref name="delta"/> card pixels, the way a dragged map follows the pointer.</summary>
    public TileZoom Panned(Vector delta, Size card)
    {
        AssertDrawn(card);

        return Covering(
            Scale,
            CenterX - delta.X / (card.Width * Scale),
            CenterY - delta.Y / (card.Height * Scale));
    }

    /// <summary>Maps a point of the unzoomed card onto the card as drawn.</summary>
    public Matrix Transform(Size card)
    {
        var offset = Offset(card);
        return new Matrix(Scale, 0, 0, Scale, offset.X, offset.Y);
    }

    private Point Offset(Size card) => new(
        card.Width / 2 - CenterX * card.Width * Scale,
        card.Height / 2 - CenterY * card.Height * Scale);

    /// <summary>Holds the middle spot far enough from each edge that the picture still reaches past it.</summary>
    private static TileZoom Covering(double scale, double centerX, double centerY)
    {
        var margin = 1 / (2 * scale);
        var zoom = new TileZoom(scale, Math.Clamp(centerX, margin, 1 - margin), Math.Clamp(centerY, margin, 1 - margin));

        Assert.That(zoom.Scale is >= 1 and <= MaxScale, "a zoom stays between the picture as it is and the ceiling", zoom.Scale);
        return zoom;
    }

    private static void AssertDrawn(Size card) =>
        Assert.That(card.Width > 0 && card.Height > 0, "a zoom is taken on a card that has been laid out", card);
}
