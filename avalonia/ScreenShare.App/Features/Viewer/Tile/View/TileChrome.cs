namespace ScreenShare.App.Features.Viewer.Tile.View;

/// <summary>
/// What a card puts over its picture, off the three facts that decide it.
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
    /// and a picture drawn elsewhere the host's.
    /// </summary>
    public static TileChrome Of(bool pointerResting, bool showStats, bool pictureElsewhere)
    {
        // Figures belong to the card drawing the picture they describe,
        // so a slot whose stream popped out is a picture again and rests like one.
        var stats = showStats && !pictureElsewhere;

        return new TileChrome(stats, pointerResting, pointerResting && !stats);
    }
}
