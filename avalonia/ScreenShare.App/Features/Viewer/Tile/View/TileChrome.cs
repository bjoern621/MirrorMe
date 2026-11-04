using Avalonia;

using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Shell.Model;

namespace ScreenShare.App.Features.Viewer.Tile.View;

/// <summary>
/// What a card puts over its picture, off the four facts that decide it.
///
/// A still pointer clears the plates and the hover chrome whatever else is up.
/// The stats overlay holds the cursor alone: a pointer holding still there is aimed at a row.
/// </summary>
/// <param name="Stats">Whether this card draws the stats overlay.</param>
/// <param name="Resting">Whether the plates and the hover chrome come off.</param>
/// <param name="HidesCursor">Whether the cursor comes off.</param>
public readonly record struct TileChrome(bool Stats, bool Resting, bool HidesCursor)
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
        // A narrow upright card would sit almost wholly under the panel.
        // The tile keeps its overlay on, which returns once the card widens.
        var stats = showStats && !pictureElsewhere && !Cramped(card);

        return new TileChrome(stats, pointerResting, pointerResting && !stats);
    }

    private static bool Cramped(Size card)
        => card.Height > card.Width && card.Width < NavWidth.LabelsMinWidth;
}
