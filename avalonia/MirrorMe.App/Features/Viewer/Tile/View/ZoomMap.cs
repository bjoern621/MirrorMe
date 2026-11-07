using Avalonia;
using Avalonia.Controls;
using Avalonia.Media;
using Avalonia.Threading;

using MirrorMe.App.Contracts;
using MirrorMe.App.Features.Viewer.Tile.Model;

namespace MirrorMe.App.Features.Viewer.Tile.View;

/// <summary>
/// Map of a zoomed picture: the whole picture as a frame, the part the card shows as a box inside it.
///
/// Every zoom step and every pan raises it, and it goes again once the zoom has held still for <see cref="HoldDuration"/>.
/// A picture zoomed into and then watched carries nothing over it.
/// The styles set the fade, the size and the brushes.
/// </summary>
internal sealed class ZoomMap : Control
{
    private static readonly TimeSpan HoldDuration = TimeSpan.FromSeconds(1);

    public static readonly StyledProperty<TileZoom> ZoomProperty =
        AvaloniaProperty.Register<ZoomMap, TileZoom>(nameof(Zoom), TileZoom.None);

    /// <summary>Picture's height over its width, which the map takes its shape from.</summary>
    public static readonly StyledProperty<double> AspectProperty =
        AvaloniaProperty.Register<ZoomMap, double>(nameof(Aspect), 9.0 / 16);

    public static readonly StyledProperty<IBrush?> BackgroundProperty = Border.BackgroundProperty.AddOwner<ZoomMap>();
    public static readonly StyledProperty<IBrush?> BorderBrushProperty = Border.BorderBrushProperty.AddOwner<ZoomMap>();
    public static readonly StyledProperty<CornerRadius> CornerRadiusProperty = Border.CornerRadiusProperty.AddOwner<ZoomMap>();

    public static readonly StyledProperty<double> StrokeThicknessProperty =
        AvaloniaProperty.Register<ZoomMap, double>(nameof(StrokeThickness), 1);

    public static readonly StyledProperty<IBrush?> WindowFillProperty =
        AvaloniaProperty.Register<ZoomMap, IBrush?>(nameof(WindowFill));

    public static readonly StyledProperty<IBrush?> WindowStrokeProperty =
        AvaloniaProperty.Register<ZoomMap, IBrush?>(nameof(WindowStroke));

    private readonly DispatcherTimer _hold = new() { Interval = HoldDuration };

    static ZoomMap()
    {
        AffectsRender<ZoomMap>(ZoomProperty, AspectProperty, BackgroundProperty, BorderBrushProperty, CornerRadiusProperty,
            StrokeThicknessProperty, WindowFillProperty, WindowStrokeProperty);
        AffectsMeasure<ZoomMap>(AspectProperty);
    }

    public ZoomMap()
    {
        _hold.Tick += (_, _) =>
        {
            _hold.Stop();
            PseudoClasses.Set(":shown", false);
        };
    }

    public TileZoom Zoom { get => GetValue(ZoomProperty); set => SetValue(ZoomProperty, value); }
    public double Aspect { get => GetValue(AspectProperty); set => SetValue(AspectProperty, value); }
    public IBrush? Background { get => GetValue(BackgroundProperty); set => SetValue(BackgroundProperty, value); }
    public IBrush? BorderBrush { get => GetValue(BorderBrushProperty); set => SetValue(BorderBrushProperty, value); }
    public CornerRadius CornerRadius { get => GetValue(CornerRadiusProperty); set => SetValue(CornerRadiusProperty, value); }
    public double StrokeThickness { get => GetValue(StrokeThicknessProperty); set => SetValue(StrokeThicknessProperty, value); }
    public IBrush? WindowFill { get => GetValue(WindowFillProperty); set => SetValue(WindowFillProperty, value); }
    public IBrush? WindowStroke { get => GetValue(WindowStrokeProperty); set => SetValue(WindowStrokeProperty, value); }

    /// <summary>
    /// A zoom set to the value it holds raises nothing, so a card redrawing its zoom on every pass leaves the map alone.
    /// Back at the whole picture, the map goes at once.
    /// </summary>
    protected override void OnPropertyChanged(AvaloniaPropertyChangedEventArgs change)
    {
        base.OnPropertyChanged(change);

        if (change.Property != ZoomProperty)
        {
            return;
        }

        _hold.Stop();
        PseudoClasses.Set(":shown", Zoom.IsZoomed);
        if (Zoom.IsZoomed)
        {
            _hold.Start();
        }
    }

    protected override Size MeasureOverride(Size availableSize)
    {
        Assert.That(double.IsFinite(Aspect) && Aspect > 0, "a picture's shape is a positive height over width", Aspect);

        var width = double.IsNaN(Width) ? 0 : Width;
        return new Size(width, width * Aspect);
    }

    public override void Render(DrawingContext context)
    {
        var size = Bounds.Size;
        var frame = new Rect(size).Deflate(StrokeThickness / 2);
        context.DrawRectangle(Background, new Pen(BorderBrush, StrokeThickness), frame, CornerRadius.TopLeft, CornerRadius.TopLeft);

        var visible = Zoom.Visible;
        var window = new Rect(visible.X * size.Width, visible.Y * size.Height, visible.Width * size.Width, visible.Height * size.Height)
            .Deflate(StrokeThickness / 2);
        context.DrawRectangle(WindowFill, new Pen(WindowStroke, StrokeThickness), window, CornerRadius.TopLeft, CornerRadius.TopLeft);
    }
}
