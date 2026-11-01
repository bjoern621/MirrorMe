using System.Globalization;
using Avalonia;
using Avalonia.Controls;
using Avalonia.Data;
using Avalonia.Input;
using Avalonia.Media;
using ScreenShare.App.Features.Insights.Chart.Model;

namespace ScreenShare.App.Features.Insights.Chart.View;

/// <summary>
/// One series over time: three gridlines labelled on the left, the limit as a dashed rule,
/// the timeline's marks along the foot, and a rule where the pointer rests.
/// Points come in the source space that Extent names: X in seconds from 0, Y in the figure's unit from Extent.Y up.
/// A segment touching a reading past the limit takes PastBrush.
/// A press pins the pointer's moment, a touch screen having no hover.
/// </summary>
public sealed class TimeChart : Control
{
    public static readonly StyledProperty<IReadOnlyList<Point>?> PointsProperty =
        AvaloniaProperty.Register<TimeChart, IReadOnlyList<Point>?>(nameof(Points));

    public static readonly StyledProperty<Rect> ExtentProperty =
        AvaloniaProperty.Register<TimeChart, Rect>(nameof(Extent), new Rect(0, 0, 60, 1));

    public static readonly StyledProperty<IReadOnlyList<bool>?> PastProperty =
        AvaloniaProperty.Register<TimeChart, IReadOnlyList<bool>?>(nameof(Past));

    public static readonly StyledProperty<double> LimitFractionProperty =
        AvaloniaProperty.Register<TimeChart, double>(nameof(LimitFraction), double.NaN);

    public static readonly StyledProperty<string> LimitLabelProperty =
        AvaloniaProperty.Register<TimeChart, string>(nameof(LimitLabel), "");

    public static readonly StyledProperty<IReadOnlyList<ChartMark>?> MarksProperty =
        AvaloniaProperty.Register<TimeChart, IReadOnlyList<ChartMark>?>(nameof(Marks));

    public static readonly StyledProperty<double> HoverSecondsProperty =
        AvaloniaProperty.Register<TimeChart, double>(nameof(HoverSeconds), double.NaN,
            defaultBindingMode: BindingMode.TwoWay);

    public static readonly StyledProperty<IBrush?> StrokeProperty =
        AvaloniaProperty.Register<TimeChart, IBrush?>(nameof(Stroke));

    public static readonly StyledProperty<IBrush?> FillProperty =
        AvaloniaProperty.Register<TimeChart, IBrush?>(nameof(Fill));

    public static readonly StyledProperty<IBrush?> GridBrushProperty =
        AvaloniaProperty.Register<TimeChart, IBrush?>(nameof(GridBrush));

    public static readonly StyledProperty<IBrush?> LimitBrushProperty =
        AvaloniaProperty.Register<TimeChart, IBrush?>(nameof(LimitBrush));

    public static readonly StyledProperty<IBrush?> LabelBrushProperty =
        AvaloniaProperty.Register<TimeChart, IBrush?>(nameof(LabelBrush));

    public static readonly StyledProperty<IBrush?> PastBrushProperty =
        AvaloniaProperty.Register<TimeChart, IBrush?>(nameof(PastBrush));

    /// <summary>Width of the axis labels on the left, px.</summary>
    private const double Gutter = 30;

    /// <summary>Height of the marks band at the foot, px.</summary>
    private const double MarkBand = 22;

    /// <summary>Room above the top gridline for the limit's label, px.</summary>
    private const double TopBand = 16;

    private const double LabelSize = 11;

    private bool _pinned;

    static TimeChart() => AffectsRender<TimeChart>(
        PointsProperty, PastProperty, ExtentProperty, LimitFractionProperty, LimitLabelProperty, MarksProperty,
        HoverSecondsProperty, StrokeProperty, FillProperty, GridBrushProperty, LimitBrushProperty, LabelBrushProperty,
        PastBrushProperty);

    public IReadOnlyList<Point>? Points { get => GetValue(PointsProperty); set => SetValue(PointsProperty, value); }

    public IReadOnlyList<bool>? Past { get => GetValue(PastProperty); set => SetValue(PastProperty, value); }

    public Rect Extent { get => GetValue(ExtentProperty); set => SetValue(ExtentProperty, value); }

    public double LimitFraction { get => GetValue(LimitFractionProperty); set => SetValue(LimitFractionProperty, value); }

    public string LimitLabel { get => GetValue(LimitLabelProperty); set => SetValue(LimitLabelProperty, value); }

    public IReadOnlyList<ChartMark>? Marks { get => GetValue(MarksProperty); set => SetValue(MarksProperty, value); }

    public double HoverSeconds { get => GetValue(HoverSecondsProperty); set => SetValue(HoverSecondsProperty, value); }

    public IBrush? Stroke { get => GetValue(StrokeProperty); set => SetValue(StrokeProperty, value); }

    public IBrush? Fill { get => GetValue(FillProperty); set => SetValue(FillProperty, value); }

    public IBrush? GridBrush { get => GetValue(GridBrushProperty); set => SetValue(GridBrushProperty, value); }

    public IBrush? LimitBrush { get => GetValue(LimitBrushProperty); set => SetValue(LimitBrushProperty, value); }

    public IBrush? LabelBrush { get => GetValue(LabelBrushProperty); set => SetValue(LabelBrushProperty, value); }

    public IBrush? PastBrush { get => GetValue(PastBrushProperty); set => SetValue(PastBrushProperty, value); }

    private Rect Plot => new(Gutter, TopBand, Math.Max(0, Bounds.Width - Gutter), Math.Max(0, Bounds.Height - MarkBand - TopBand));

    public override void Render(DrawingContext context)
    {
        // Painted so the whole surface takes the pointer, the empty space between points included.
        context.FillRectangle(Brushes.Transparent, new Rect(Bounds.Size));

        var plot = Plot;
        if (plot.Width <= 0 || plot.Height <= 0)
        {
            return;
        }

        DrawGrid(context, plot);
        DrawSeries(context, plot);
        DrawLimit(context, plot);
        DrawMarks(context, plot);
        DrawHover(context, plot);
    }

    protected override void OnPointerMoved(PointerEventArgs e)
    {
        base.OnPointerMoved(e);
        if (!_pinned)
        {
            HoverSeconds = SecondsAt(e.GetPosition(this).X);
        }
    }

    protected override void OnPointerPressed(PointerPressedEventArgs e)
    {
        base.OnPointerPressed(e);
        _pinned = !_pinned;
        HoverSeconds = SecondsAt(e.GetPosition(this).X);
    }

    protected override void OnPointerExited(PointerEventArgs e)
    {
        base.OnPointerExited(e);
        if (!_pinned)
        {
            HoverSeconds = double.NaN;
        }
    }

    private double SecondsAt(double x)
    {
        var plot = Plot;
        if (plot.Width <= 0 || x < plot.Left)
        {
            return double.NaN;
        }

        return Math.Clamp((x - plot.Left) / plot.Width, 0, 1) * Extent.Width;
    }

    private Point Map(Rect plot, Point source)
        => new(plot.Left + source.X / Extent.Width * plot.Width,
            plot.Bottom - Math.Clamp((source.Y - Extent.Y) / Extent.Height, 0, 1) * plot.Height);

    private void DrawGrid(DrawingContext context, Rect plot)
    {
        var pen = new Pen(GridBrush, 1);
        foreach (var share in new[] { 0, 0.5, 1 })
        {
            var y = plot.Bottom - share * plot.Height;
            context.DrawLine(pen, new Point(plot.Left, y), new Point(plot.Right, y));

            var value = Extent.Y + Extent.Height * share;
            var label = Text(value.ToString(value < 10 && value % 1 != 0 ? "0.#" : "0", CultureInfo.CurrentCulture));
            context.DrawText(label, new Point(Gutter - 6 - label.Width, y - label.Height / 2));
        }
    }

    private void DrawSeries(DrawingContext context, Rect plot)
    {
        if (Points is not { Count: > 1 } points)
        {
            return;
        }

        var mapped = points.Select(point => Map(plot, point)).ToList();
        if (Fill is not null)
        {
            var area = new StreamGeometry();
            using (var g = area.Open())
            {
                g.BeginFigure(new Point(mapped[0].X, plot.Bottom), true);
                foreach (var p in mapped)
                {
                    g.LineTo(p);
                }

                g.LineTo(new Point(mapped[^1].X, plot.Bottom));
                g.EndFigure(true);
            }

            context.DrawGeometry(Fill, null, area);
        }

        var line = new StreamGeometry();
        using (var g = line.Open())
        {
            g.BeginFigure(mapped[0], false);
            foreach (var p in mapped.Skip(1))
            {
                g.LineTo(p);
            }

            g.EndFigure(false);
        }

        context.DrawGeometry(null, new Pen(Stroke, 1.5, lineJoin: PenLineJoin.Round), line);

        // Points and Past arrive as two bindings, so one can lead the other for a pass.
        if (Past is not { } past || past.Count != points.Count)
        {
            return;
        }

        var pastPen = new Pen(PastBrush, 1.5, lineCap: PenLineCap.Round);
        for (var i = 1; i < mapped.Count; i++)
        {
            if (past[i - 1] || past[i])
            {
                context.DrawLine(pastPen, mapped[i - 1], mapped[i]);
            }
        }
    }

    private void DrawLimit(DrawingContext context, Rect plot)
    {
        if (double.IsNaN(LimitFraction))
        {
            return;
        }

        var y = plot.Top + Math.Clamp(LimitFraction, 0, 1) * plot.Height;
        context.DrawLine(new Pen(LimitBrush, 1, new DashStyle([4, 3], 0)), new Point(plot.Left, y), new Point(plot.Right, y));

        var label = Text(LimitLabel, LimitBrush);
        context.DrawText(label, new Point(plot.Right - label.Width - 4, y - label.Height - 1));
    }

    private void DrawMarks(DrawingContext context, Rect plot)
    {
        if (Marks is not { Count: > 0 } marks)
        {
            return;
        }

        var xs = marks.Select(mark => plot.Left + mark.At / Extent.Width * plot.Width).ToList();
        foreach (var x in xs)
        {
            context.DrawEllipse(LabelBrush, null, new Point(x, plot.Bottom + 10), 2.5, 2.5);
        }

        // A label sits right of its dot, or left where the edge cuts it off.
        // One that would cover another label or dot keeps its dot and drops its words.
        var right = double.NegativeInfinity;
        for (var i = 0; i < marks.Count; i++)
        {
            var x = xs[i];
            var label = Text(marks[i].Label);
            var left = Math.Max(plot.Left, x + 6 + label.Width <= plot.Right ? x + 6 : x - 6 - label.Width);
            var covers = xs.Where((other, j) => j != i).Any(other => other > left - 4 && other < left + label.Width + 4);
            if (left <= right || covers)
            {
                continue;
            }

            context.DrawText(label, new Point(left, plot.Bottom + 10 - label.Height / 2));
            right = left + label.Width + 8;
        }
    }

    private void DrawHover(DrawingContext context, Rect plot)
    {
        if (double.IsNaN(HoverSeconds))
        {
            return;
        }

        var x = plot.Left + HoverSeconds / Extent.Width * plot.Width;
        context.DrawLine(new Pen(LabelBrush, 1), new Point(x, plot.Top), new Point(x, plot.Bottom));

        if (Points is { Count: > 0 } points)
        {
            var nearest = Enumerable.Range(0, points.Count).MinBy(i => Math.Abs(points[i].X - HoverSeconds));
            var isPast = Past is { } past && past.Count == points.Count && past[nearest];
            context.DrawEllipse(isPast ? PastBrush : Stroke, null, Map(plot, points[nearest]), 3.5, 3.5);
        }
    }

    private FormattedText Text(string text, IBrush? brush = null)
        => new(text, CultureInfo.CurrentCulture, FlowDirection.LeftToRight, Typeface.Default, LabelSize,
            brush ?? LabelBrush);
}
