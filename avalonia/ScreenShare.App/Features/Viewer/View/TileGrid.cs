using Avalonia;
using Avalonia.Controls;
using Avalonia.Input;
using Avalonia.VisualTree;

using ScreenShare.App.Contracts;
using ScreenShare.App.Controls;
using ScreenShare.App.Features.Viewer.Model;

namespace ScreenShare.App.Features.Viewer.View;

/// <summary>
/// Panel that puts tiles where the arrangement says, computing nothing itself.
/// Every rectangle comes from <see cref="TileLayout"/> or <see cref="FocusLayout"/>, both pure and tested
/// without a window, so what is here is the two Avalonia passes and the reading of each child's shape
/// (<c>avalonia/README.md</c>).
///
/// A child declares its aspect ratio through <see cref="AspectProperty"/>
/// and whether it is the focused one through <see cref="IsFocusedTileProperty"/>, both bound in the item template.
/// Attached values rather than anybody's data context, so this knows no view model and a second kind of tile
/// needs no change here.
/// </summary>
public sealed class TileGrid : Panel
{
    /// <summary>
    /// Space between two tiles, and between two rows, in device-independent pixels.
    /// Fixed rather than styled: an argument to the arrangement, and spacing off a theme would be a layout whose
    /// test and whose screen disagreed.
    /// </summary>
    private const double Gap = 8;

    /// <summary>Width over height of one tile's stream, as the tile knows it.</summary>
    public static readonly AttachedProperty<double> AspectProperty =
        AvaloniaProperty.RegisterAttached<TileGrid, Control, double>("Aspect", TileLayout.UnknownAspect);

    /// <summary>At most one child carries it.</summary>
    public static readonly AttachedProperty<bool> IsFocusedTileProperty =
        AvaloniaProperty.RegisterAttached<TileGrid, Control, bool>("IsFocusedTile");

    public static readonly StyledProperty<LayoutMode> ModeProperty =
        AvaloniaProperty.Register<TileGrid, LayoutMode>(nameof(Mode));

    /// <summary>
    /// Size the arrangement is fitted into where the panel is measured with an unbounded one,
    /// which is what a scroll viewer hands it.
    /// Bound to the viewport rather than guessed: "fits the box" is the whole of what the arrangement decides,
    /// and a panel measured against infinity has no box.
    /// Empty before anything measured the viewport, the arrangement then taking whatever finite size the pass
    /// was given.
    /// </summary>
    public static readonly StyledProperty<Size> ViewportProperty =
        AvaloniaProperty.Register<TileGrid, Size>(nameof(Viewport));

    /// <summary>
    /// How far the scroll viewer around this panel has scrolled it sideways.
    /// The focused tile is placed at it so it stands still while the strip moves (<see cref="FocusLayout"/>).
    /// </summary>
    public static readonly StyledProperty<double> OffsetProperty =
        AvaloniaProperty.Register<TileGrid, double>(nameof(Offset));

    /// <summary>Box the last measure pass solved, which the arrange pass places the children in.</summary>
    private Size _measured;

    static TileGrid()
    {
        AffectsMeasure<TileGrid>(ModeProperty, ViewportProperty);
        AffectsArrange<TileGrid>(OffsetProperty);
        AffectsParentMeasure<TileGrid>(AspectProperty, IsFocusedTileProperty);
    }

    public LayoutMode Mode
    {
        get => GetValue(ModeProperty);
        set => SetValue(ModeProperty, value);
    }

    public Size Viewport
    {
        get => GetValue(ViewportProperty);
        set => SetValue(ViewportProperty, value);
    }

    public double Offset
    {
        get => GetValue(OffsetProperty);
        set => SetValue(OffsetProperty, value);
    }

    public static double GetAspect(Control child) => child.GetValue(AspectProperty);

    public static void SetAspect(Control child, double value) => child.SetValue(AspectProperty, value);

    public static bool GetIsFocusedTile(Control child) => child.GetValue(IsFocusedTileProperty);

    public static void SetIsFocusedTile(Control child, bool value) => child.SetValue(IsFocusedTileProperty, value);

    /// <summary>
    /// Measures every child at the size it will be arranged at, and reports the size the arrangement needs.
    /// The arrangement's own rather than the box's,
    /// so a grid that had to scroll comes out taller than its viewport and a strip that had to scroll comes out
    /// wider, and the scroll viewer around it has something to scroll.
    /// An arrangement that fitted reports what it used.
    /// </summary>
    protected override Size MeasureOverride(Size availableSize)
    {
        _measured = Box(availableSize);
        var (places, width) = Solve(_measured);

        foreach (var (child, rect) in places)
        {
            child.Measure(new Size(rect.Width, rect.Height));
        }

        var height = places.Count == 0 ? 0 : places.Max(place => place.Rect.Bottom);
        return new Size(width, height);
    }

    /// <summary>
    /// Places every child in the box the measure pass solved.
    /// The viewport can move between the two passes, and a child arranged at a size it was not measured at
    /// keeps its insides at the old size: a later measure leaves its rectangle unchanged, so no arrange reaches them.
    /// The viewport moving measures again, and that pass places the tiles in the new box.
    /// </summary>
    protected override Size ArrangeOverride(Size finalSize)
    {
        var (places, _) = Solve(_measured);
        foreach (var (child, rect) in places)
        {
            child.Arrange(rect);
        }

        return finalSize;
    }

    /// <summary>
    /// Wheel means the strip while a tile is focused.
    /// The arrangement fits the height there, so the notch would otherwise be spent on nothing,
    /// and a platform that sends no sideways delta would leave the strip reachable by its scrollbar alone.
    /// </summary>
    protected override void OnPointerWheelChanged(PointerWheelEventArgs e)
    {
        base.OnPointerWheelChanged(e);

        if (e.Handled || Mode != LayoutMode.Focus || this.FindAncestorOfType<ScrollViewer>() is not { } scroll)
        {
            return;
        }

        var room = scroll.Extent.Width - scroll.Viewport.Width;
        if (room <= 0)
        {
            return;
        }

        var delta = e.Delta.X != 0 ? e.Delta.X : e.Delta.Y;
        scroll.Offset = scroll.Offset.WithX(Math.Clamp(scroll.Offset.X - (delta * WheelScroll.StepOf(WaylandTouchpad.Source, e.Delta)), 0, room));
        e.Handled = true;
    }

    /// <summary>
    /// Box the arrangement is fitted into.
    ///
    /// <see cref="Viewport"/> wins over the size a pass was handed.
    /// Inside a scroll viewer the two passes get different sizes,
    /// measure an unbounded one and arrange the size measure returned,
    /// so solving against what each was given solves two boxes,
    /// placing the tiles by one having measured them by the other.
    /// The viewport is the space a reader sees, what "fits the box" is about.
    ///
    /// An unmeasured side counts as no room: the arrangement places nothing until a pass carries a real box.
    /// </summary>
    private Size Box(Size available) => new(Side(Viewport.Width, available.Width), Side(Viewport.Height, available.Height));

    private static double Side(double viewport, double available)
        => viewport > 0 ? viewport : (double.IsFinite(available) && available > 0 ? available : 0);

    /// <summary>Where the tiles go, and how wide the arrangement came out.</summary>
    private (List<(Control Child, Rect Rect)> Places, double Width) Solve(Size box) => Mode switch
    {
        LayoutMode.Focus => Focused(box),
        LayoutMode.Stack => Stacked(box),
        LayoutMode.Stage => Alone(box),
        LayoutMode.Grid => (Grid(box), box.Width),
        _ => Assert.Never<(List<(Control Child, Rect Rect)>, double)>("a layout mode the panel arranges", Mode),
    };

    /// <summary>
    /// Every tile at the one height the arrangement chose, each as wide as its own shape makes it there.
    /// </summary>
    private List<(Control Child, Rect Rect)> Grid(Size box)
    {
        var children = Children.OfType<Control>().ToList();
        var arrangement = TileLayout.Solve(children.Select(GetAspect).ToList(), box.Width, box.Height, Gap);

        return Place(children, arrangement.Tiles);
    }

    /// <summary>
    /// One tile above, the rest in a strip below.
    /// With nothing focused it is the grid,
    /// so the mode is safe to be in while a focused stream is being chosen and after it has gone:
    /// the arrangement degrades to the other mode rather than to an empty screen.
    /// </summary>
    private (List<(Control Child, Rect Rect)> Places, double Width) Focused(Size box)
    {
        var children = Children.OfType<Control>().ToList();
        var focused = children.FindIndex(GetIsFocusedTile);
        if (focused < 0)
        {
            return (Grid(box), box.Width);
        }

        var arrangement = FocusLayout.Solve(
            children.Select(GetAspect).ToList(), focused, box.Width, box.Height, Gap, Offset);

        return (Place(children, arrangement.Tiles), Math.Max(box.Width, arrangement.Extent));
    }

    /// <summary>
    /// One tile across the top, the rest two to a row under it, the column as tall as that makes it.
    /// With nothing focused it is the grid, for the reason <see cref="Focused"/> gives.
    /// </summary>
    private (List<(Control Child, Rect Rect)> Places, double Width) Stacked(Size box)
    {
        var children = Children.OfType<Control>().ToList();
        var focused = children.FindIndex(GetIsFocusedTile);
        if (focused < 0)
        {
            return (Grid(box), box.Width);
        }

        var arrangement = FocusLayout.Stack(children.Select(GetAspect).ToList(), focused, box.Width, Gap);
        return (Place(children, arrangement.Tiles), box.Width);
    }

    /// <summary>
    /// One tile filling the box, the rest at no size.
    /// With nothing focused it is the grid, for the reason <see cref="Focused"/> gives.
    /// </summary>
    private (List<(Control Child, Rect Rect)> Places, double Width) Alone(Size box)
    {
        var children = Children.OfType<Control>().ToList();
        var focused = children.FindIndex(GetIsFocusedTile);
        if (focused < 0)
        {
            return (Grid(box), box.Width);
        }

        var arrangement = FocusLayout.Alone(children.Select(GetAspect).ToList(), focused, box.Width, box.Height);
        return (Place(children, arrangement.Tiles), box.Width);
    }

    /// <summary>
    /// The rectangle each child was given, and an empty one for every child where the box had no room
    /// (<see cref="TileLayout.Solve"/>), which is the first measure pass inside a scroll viewer:
    /// the viewport is unmeasured, so there is no box to solve against until the pass after.
    /// A child this list skips is a child the measure pass never measures.
    /// </summary>
    private static List<(Control Child, Rect Rect)> Place(
        List<Control> children, IReadOnlyList<TileLayout.Placement> tiles)
    {
        Assert.That(
            tiles.Count == children.Count || tiles.Count == 0,
            "the arrangement places every tile or none of them",
            tiles.Count,
            children.Count);

        var places = children.Select(child => (Child: child, Rect: default(Rect))).ToList();
        foreach (var tile in tiles)
        {
            places[tile.Index] = (children[tile.Index], new Rect(tile.X, tile.Y, tile.Width, tile.Height));
        }

        return places;
    }
}
