using Avalonia;

using MirrorMe.App.Contracts;
using MirrorMe.App.Features.Shell.Model;

namespace MirrorMe.App.Features.Viewer.Tile.View;

/// <summary>
/// What a card puts over its picture and into its menu, off the four facts that decide it.
///
/// A still pointer clears the plates and the hover chrome whatever else is up.
/// The stats overlay holds the cursor alone: a pointer holding still there is aimed at a row.
/// </summary>
/// <param name="Stats">Whether this card draws the stats overlay.</param>
/// <param name="StatsFits">Whether the card has room for the overlay, which enables its menu row and key.</param>
/// <param name="StatsTicked">Whether the overlay's menu row carries its tick.</param>
/// <param name="ClosesStats">Whether the card turns the tile's overlay off, having no room for it.</param>
/// <param name="Resting">Whether the plates and the hover chrome come off.</param>
/// <param name="HidesCursor">Whether the cursor comes off.</param>
public readonly record struct TileChrome(bool Stats, bool StatsFits, bool StatsTicked, bool ClosesStats, bool Resting, bool HidesCursor)
{
    /// <summary>
    /// Reads a card's state.
    /// The pointer is the card's own fact (<see cref="RestingWatch"/>), the overlay the tile's,
    /// a picture drawn elsewhere the host's and the size the card's layout.
    /// </summary>
    public static TileChrome Of(bool pointerResting, bool showStats, bool pictureElsewhere, Size card)
    {
        Assert.That(card.Width >= 0 && card.Height >= 0, "a card's size is never negative", card);

        // Figures belong to the card drawing the picture they describe,
        // so a slot whose stream popped out is a picture again and rests like one.
        // Its own shape decides nothing about a panel drawn in another window.
        // A narrow card would sit almost wholly under the panel, whatever its shape.
        // The card turns the overlay off, so a tile staged again after a turn in the strip comes back without it.
        var fits = pictureElsewhere || !Cramped(card);
        var stats = showStats && !pictureElsewhere && fits;

        return new TileChrome(stats, fits, showStats && fits, showStats && !fits, pointerResting, pointerResting && !stats);
    }

    private static bool Cramped(Size card)
        => card.Width > 0 && card.Width < NavWidth.LabelsMinWidth;
}
