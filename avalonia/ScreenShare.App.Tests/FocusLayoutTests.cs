using ScreenShare.App.Features.Viewer.Model;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Failures a screenshot shows once a group is big enough:
/// strip tiles past the right edge with no way to reach them, a stage that slides away as the strip is scrolled,
/// a scrollbar over the tiles it scrolls.
/// Solver is pure, so none of it needs a window (<c>Features/Viewer/Model/FocusLayout.cs</c>).
/// </summary>
public sealed class FocusLayoutTests
{
    private const double Gap = 8;
    private const double Wide = 16.0 / 9;

    private static FocusLayout.Arrangement Solve(int count, double width, double height, double offset = 0)
        => FocusLayout.Solve([.. Enumerable.Repeat(Wide, count)], 0, width, height, Gap, offset);

    private static TileLayout.Placement Stage(FocusLayout.Arrangement arrangement)
        => arrangement.Tiles.Single(tile => tile.Index == 0);

    private static List<TileLayout.Placement> Strip(FocusLayout.Arrangement arrangement)
        => arrangement.Tiles.Where(tile => tile.Index != 0).OrderBy(tile => tile.X).ToList();

    /// <summary>Slack split in two reads as a margin, and a strip that fits has nothing to scroll.</summary>
    [Fact]
    public void AStripThatFitsIsCentredAndScrollsNothing()
    {
        var arrangement = Solve(3, 1200, 800);
        var strip = Strip(arrangement);

        Assert.Equal(1200, arrangement.Extent, 3);
        Assert.Equal(strip[0].X, 1200 - (strip[^1].X + strip[^1].Width), 3);
    }

    /// <summary>
    /// The one thing the reader cannot do without: a strip past the width reports how wide it is,
    /// so the scroll viewer around the panel has something to scroll.
    /// </summary>
    [Fact]
    public void AStripWiderThanTheBoxReportsItsOwnWidth()
    {
        var arrangement = Solve(12, 600, 800);
        var strip = Strip(arrangement);

        Assert.True(arrangement.Extent > 600, $"the strip outgrew the box, not {arrangement.Extent}");
        Assert.Equal(0, strip[0].X, 3);
        Assert.Equal(arrangement.Extent, strip[^1].X + strip[^1].Width, 3);
    }

    /// <summary>
    /// The panel is drawn shifted by the offset, so a stage placed at the offset is a stage that stays put
    /// while the strip under it moves.
    /// </summary>
    [Fact]
    public void TheStageRidesTheOffsetAndTheStripDoesNot()
    {
        var arrangement = Solve(12, 600, 800, 100);

        Assert.Equal(100, Stage(arrangement).X, 3);
        Assert.Equal(0, Strip(arrangement)[0].X, 3);
    }

    /// <summary>An offset past the end leaves the last tile against the right edge.</summary>
    [Fact]
    public void AnOffsetPastTheEndStopsAtIt()
    {
        var arrangement = Solve(12, 600, 800, 10_000);

        Assert.Equal(arrangement.Extent - 600, Stage(arrangement).X, 3);
    }

    /// <summary>The stage keeps its stream shape, the leftover space going to its margins.</summary>
    [Fact]
    public void TheStageKeepsItsStreamsShape()
    {
        var arrangement = FocusLayout.Solve([21.0 / 9, Wide, Wide], 0, 1200, 800, Gap, 0);
        var stage = Stage(arrangement);

        Assert.Equal(21.0 / 9, stage.Width / stage.Height, 3);
        Assert.True(stage.Width <= 1200.001, $"the stage fits the width, not {stage.Width}");
    }

    /// <summary>
    /// The band under the strip is reserved whether or not the strip scrolls.
    /// Taking it only where the strip overflows would shrink the tiles into fitting, which frees the band,
    /// which overflows them again.
    /// </summary>
    [Fact]
    public void TheStripLeavesRoomForItsScrollbar()
    {
        foreach (var count in (int[])[3, 12])
        {
            var bottom = Strip(Solve(count, 600, 800)).Max(tile => tile.Y + tile.Height);

            Assert.True(800 - bottom >= FocusLayout.ScrollbarBand, $"{count} tiles left {800 - bottom} under the strip");
        }
    }

    /// <summary>With nothing beside it the stage takes the box, no strip meaning no band to keep clear.</summary>
    [Fact]
    public void AloneOnTheStageATileTakesTheWholeBox()
    {
        var stage = Stage(Solve(1, 1200, 800));

        Assert.Equal(1200, stage.Width, 3);
        Assert.Equal(1200 / Wide, stage.Height, 3);
    }

    /// <summary>
    /// A narrow window keeps the stage at the full width and its stream's shape,
    /// and wraps the strip under it two to a row.
    /// </summary>
    [Fact]
    public void AStackedStripRunsTwoToARowUnderTheStage()
    {
        var arrangement = FocusLayout.Stack([.. Enumerable.Repeat(Wide, 4)], 0, 336, Gap);
        var stage = Stage(arrangement);
        var strip = Strip(arrangement).OrderBy(tile => tile.Y).ThenBy(tile => tile.X).ToList();
        var cell = (336 - Gap) / 2;

        Assert.Equal(336, stage.Width, 3);
        Assert.Equal(336 / Wide, stage.Height, 3);
        Assert.Equal([0, cell + Gap, 0], strip.Select(tile => tile.X).ToList());
        Assert.All(strip, tile => Assert.Equal(cell, tile.Width, 3));
        Assert.Equal(stage.Height + Gap, strip[0].Y, 3);
        Assert.Equal(strip[0].Y + strip[0].Height + Gap, strip[2].Y, 3);
        Assert.Equal(336, arrangement.Extent, 3);
    }

    /// <summary>A box with no room places nothing, which is the first measure pass inside a scroll viewer.</summary>
    [Fact]
    public void ABoxWithNoRoomPlacesNothing()
    {
        var arrangement = Solve(4, 0, 800);

        Assert.Empty(arrangement.Tiles);
        Assert.Equal(0, arrangement.Extent, 3);
    }
}
