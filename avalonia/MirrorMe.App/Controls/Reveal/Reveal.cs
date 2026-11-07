using Avalonia;
using Avalonia.Controls;
using Avalonia.VisualTree;
using MirrorMe.App.Contracts;

namespace MirrorMe.App.Controls;

/// <summary>
/// Holds the body of a fold and grows it to its full height as the fold opens, fading it in on the way.
/// Closing plays the same motion back, and the body hides once nothing of it shows.
/// A press halfway through turns the motion round from where it stands.
///
/// <see cref="IsOpen"/> names the state wanted.
/// <see cref="Openness"/> follows it under the transition that <c>Controls/Reveal/Reveal.axaml</c> sets,
/// so the body is clipped to the height reached so far and the rows under the fold move with it.
/// While <see cref="FollowsOpening"/> holds, the nearest scroll viewer moves with the opening,
/// so a fold pressed at the foot of a column grows into view.
/// </summary>
public sealed class Reveal : Decorator
{
    /// <summary>Room kept above and below the body, wide enough for a fold's heading and the padding of its card.</summary>
    private const double FollowMarginPixels = 40;

    public static readonly StyledProperty<bool> IsOpenProperty =
        AvaloniaProperty.Register<Reveal, bool>(nameof(IsOpen));

    /// <summary>Share of the body's height drawn: 0 closed, 1 open.</summary>
    public static readonly StyledProperty<double> OpennessProperty =
        AvaloniaProperty.Register<Reveal, double>(nameof(Openness));

    /// <summary>False on a reveal that can open out of the reader's view, where following would pull the scroll away from the press.</summary>
    public static readonly StyledProperty<bool> FollowsOpeningProperty =
        AvaloniaProperty.Register<Reveal, bool>(nameof(FollowsOpening), defaultValue: true);

    /// <summary>True on a reveal in a spaced stack, which would keep a gap for a closed body.</summary>
    public static readonly StyledProperty<bool> LeavesNoGapProperty =
        AvaloniaProperty.Register<Reveal, bool>(nameof(LeavesNoGap));

    /// <summary>Body height held while closing, since a line bound to its text closes as the text clears.</summary>
    private double _heldHeight;

    static Reveal()
    {
        AffectsMeasure<Reveal>(OpennessProperty);
        ClipToBoundsProperty.OverrideDefaultValue<Reveal>(true);
    }

    public Reveal()
    {
        Draw();
    }

    public bool IsOpen
    {
        get => GetValue(IsOpenProperty);
        set => SetValue(IsOpenProperty, value);
    }

    public double Openness
    {
        get => GetValue(OpennessProperty);
        set => SetValue(OpennessProperty, value);
    }

    public bool FollowsOpening
    {
        get => GetValue(FollowsOpeningProperty);
        set => SetValue(FollowsOpeningProperty, value);
    }

    public bool LeavesNoGap
    {
        get => GetValue(LeavesNoGapProperty);
        set => SetValue(LeavesNoGapProperty, value);
    }

    protected override void OnPropertyChanged(AvaloniaPropertyChangedEventArgs change)
    {
        base.OnPropertyChanged(change);

        if (change.Property == IsOpenProperty)
        {
            Openness = IsOpen ? 1 : 0;
        }

        if (change.Property == IsOpenProperty || change.Property == OpennessProperty
            || change.Property == ChildProperty || change.Property == LeavesNoGapProperty)
        {
            Draw();
        }

        if (change.Property == OpennessProperty && IsOpen && FollowsOpening && this.IsAttachedToVisualTree())
        {
            LayoutUpdated -= FollowOnce;
            LayoutUpdated += FollowOnce;
        }
    }

    /// <summary>Runs after layout, since the scroll offset clamps to an extent that layout grows.</summary>
    private void FollowOnce(object? sender, EventArgs e)
    {
        LayoutUpdated -= FollowOnce;
        this.BringIntoView(new Rect(Bounds.Size).Inflate(new Thickness(0, FollowMarginPixels)));
    }

    /// <summary>Writes the body's presence and fade from the two properties. Idempotent.</summary>
    private void Draw()
    {
        var shown = IsOpen || Openness > 0;
        IsVisible = shown || !LeavesNoGap;

        if (Child is not { } body)
        {
            return;
        }

        body.IsVisible = shown;
        body.Opacity = Openness;
    }

    /// <summary>The body measures at its full height, and the fold asks for the share of it that shows.</summary>
    protected override Size MeasureOverride(Size availableSize)
    {
        Assert.That(Openness is >= 0 and <= 1, "openness runs from 0 closed to 1 open, the easing never overshooting", Openness);

        if (Child is not { } body)
        {
            return default;
        }

        body.Measure(availableSize.WithHeight(double.PositiveInfinity));
        _heldHeight = IsOpen ? body.DesiredSize.Height : Math.Max(_heldHeight, body.DesiredSize.Height);
        var full = body.DesiredSize.WithHeight(_heldHeight);
        var shown = new Size(full.Width, full.Height * Openness);

        Assert.That(shown.Height <= full.Height, "a fold never asks for more than its body's height", shown, full);
        return shown;
    }

    /// <summary>The body keeps its full height and its top edge, so it unrolls downwards under the clip.</summary>
    protected override Size ArrangeOverride(Size finalSize)
    {
        if (Child is not { } body)
        {
            return finalSize;
        }

        body.Arrange(new Rect(0, 0, finalSize.Width, Math.Max(finalSize.Height, body.DesiredSize.Height)));
        return finalSize;
    }
}
