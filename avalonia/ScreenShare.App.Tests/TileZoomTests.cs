using Avalonia;
using ScreenShare.App.Features.Viewer.Tile.Model;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// What a reader zooming into a tile expects: the spot under the pointer stays under it,
/// the picture never slides off and leaves a bare edge, and a resized card keeps the same spot in view.
/// Pure, so none of it needs a window (<c>Features/Viewer/Tile/Model/TileZoom.cs</c>).
/// </summary>
public sealed class TileZoomTests
{
    private static readonly Size Card = new(800, 450);

    private static Point OnCard(TileZoom zoom, Size card, Point picture) => zoom.Transform(card).Transform(picture);

    [Fact]
    public void AnUnzoomedCardDrawsThePictureAsItIs()
    {
        Assert.Equal(Matrix.Identity, TileZoom.None.Transform(Card));
        Assert.False(TileZoom.None.IsZoomed);
    }

    [Fact]
    public void TheSpotUnderThePointerStaysUnderIt()
    {
        var pointer = new Point(600, 100);
        var zoomed = TileZoom.None.At(pointer, Card, 2);

        Assert.Equal(2, zoomed.Scale, 6);
        var drawn = OnCard(zoomed, Card, pointer);
        Assert.Equal(pointer.X, drawn.X, 6);
        Assert.Equal(pointer.Y, drawn.Y, 6);
    }

    [Fact]
    public void TwoStepsLandWhereOneStepOfTheirProductDoes()
    {
        var pointer = new Point(200, 300);
        var twice = TileZoom.None.At(pointer, Card, 1.5).At(pointer, Card, 2);
        var once = TileZoom.None.At(pointer, Card, 3);

        Assert.Equal(once.Scale, twice.Scale, 6);
        Assert.Equal(once.CenterX, twice.CenterX, 6);
        Assert.Equal(once.CenterY, twice.CenterY, 6);
    }

    /// <summary>A zoom near a corner would put the picture's edge inside the card, so the picture is held covering it.</summary>
    [Fact]
    public void ThePictureAlwaysCoversTheCard()
    {
        var zoomed = TileZoom.None.At(new Point(0, 0), Card, 2).Panned(new Vector(500, 500), Card);

        var topLeft = OnCard(zoomed, Card, new Point(0, 0));
        var bottomRight = OnCard(zoomed, Card, new Point(Card.Width, Card.Height));
        Assert.True(topLeft.X <= 0 && topLeft.Y <= 0, $"picture starts at {topLeft} inside the card");
        Assert.True(bottomRight.X >= Card.Width && bottomRight.Y >= Card.Height, $"picture ends at {bottomRight} inside the card");
    }

    [Fact]
    public void ZoomingOutStopsAtThePictureAsItIs()
    {
        var back = TileZoom.None.At(new Point(100, 100), Card, 4).At(new Point(700, 400), Card, 0.01);

        Assert.Equal(TileZoom.None, back);
    }

    [Fact]
    public void ZoomingInStopsAtTheCeiling()
    {
        var deep = TileZoom.None.At(new Point(400, 225), Card, 1000);

        Assert.Equal(TileZoom.MaxScale, deep.Scale, 6);
    }

    /// <summary>Dragging the picture moves it with the pointer, as a map does.</summary>
    [Fact]
    public void PanningMovesThePictureWithThePointer()
    {
        var zoomed = TileZoom.None.At(new Point(400, 225), Card, 4);
        var before = OnCard(zoomed, Card, new Point(400, 225));
        var after = OnCard(zoomed.Panned(new Vector(-30, 20), Card), Card, new Point(400, 225));

        Assert.Equal(before.X - 30, after.X, 6);
        Assert.Equal(before.Y + 20, after.Y, 6);
    }

    [Fact]
    public void AnUnzoomedCardShowsAllOfThePicture() => Assert.Equal(new Rect(0, 0, 1, 1), TileZoom.None.Visible);

    /// <summary>The part the zoom map outlines: the card's share of the picture, around the spot in the card's middle.</summary>
    [Fact]
    public void AZoomedCardShowsItsShareOfThePictureAroundItsMiddle()
    {
        var zoomed = TileZoom.None.At(new Point(600, 100), Card, 4);
        var visible = zoomed.Visible;

        Assert.Equal(0.25, visible.Width, 6);
        Assert.Equal(0.25, visible.Height, 6);
        Assert.Equal(zoomed.CenterX, visible.Center.X, 6);
        Assert.Equal(zoomed.CenterY, visible.Center.Y, 6);
    }

    [Fact]
    public void TheVisiblePartStaysInsideThePicture()
    {
        var corner = TileZoom.None.At(new Point(0, 0), Card, 2).Panned(new Vector(5000, 5000), Card);

        Assert.True(new Rect(0, 0, 1, 1).Contains(corner.Visible), $"visible part {corner.Visible} reaches past the picture");
    }

    /// <summary>A focus or a window drag resizes the card, and the spot being looked at stays in its middle.</summary>
    [Fact]
    public void AResizedCardKeepsTheSameSpotInTheMiddle()
    {
        var zoomed = TileZoom.None.At(new Point(600, 100), Card, 3);
        var larger = new Size(1600, 900);

        var middle = OnCard(zoomed, Card, new Point(zoomed.CenterX * Card.Width, zoomed.CenterY * Card.Height));
        var middleLarger = OnCard(zoomed, larger, new Point(zoomed.CenterX * larger.Width, zoomed.CenterY * larger.Height));
        Assert.Equal(Card.Width / 2, middle.X, 6);
        Assert.Equal(larger.Width / 2, middleLarger.X, 6);
        Assert.Equal(larger.Height / 2, middleLarger.Y, 6);
    }
}
