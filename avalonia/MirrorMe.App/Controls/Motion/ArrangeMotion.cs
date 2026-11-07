using System.Globalization;
using System.Runtime.CompilerServices;
using Avalonia;
using Avalonia.Animation;
using Avalonia.Controls;
using Avalonia.Media.Transformation;
using Avalonia.Styling;
using MirrorMe.App.Contracts;

namespace MirrorMe.App.Controls;

/// <summary>
/// Eases a control into a change that lands in one pass: a place a panel moved it to, or a first showing.
/// A run replaces the run before it, since two animations of one property on one control play back to back.
/// </summary>
internal static class ArrangeMotion
{
    private static readonly ConditionalWeakTable<Control, CancellationTokenSource> Slides = new();
    private static readonly ConditionalWeakTable<Control, CancellationTokenSource> Arrivals = new();

    /// <summary>
    /// Slides a control from <paramref name="from"/> to <paramref name="to"/>, where it stands already.
    /// Leaves a control that stayed, or that changed size, which has no one place to slide from.
    /// </summary>
    public static void Slide(Control control, Rect from, Rect to)
    {
        Assert.NotNull(control, "a slide needs the control it moves");

        var (x, y) = (from.X - to.X, from.Y - to.Y);
        if (from.Size != to.Size || (x == 0 && y == 0))
        {
            return;
        }

        var shifted = string.Create(CultureInfo.InvariantCulture, $"translate({x}px, {y}px)");
        Run(control, Slides,
            new Setter(Visual.RenderTransformProperty, TransformOperations.Parse(shifted)),
            new Setter(Visual.RenderTransformProperty, TransformOperations.Parse("none")));
    }

    /// <summary>
    /// Eases a control from the rectangle it held into <paramref name="to"/>, where it stands already.
    /// Scales on the way, for a tile whose whole rectangle changes as the arrangement does.
    /// Expects the render transform origin at the control's centre, which is the default.
    /// </summary>
    public static void Reshape(Control control, Rect from, Rect to)
    {
        Assert.NotNull(control, "a reshape needs the control it moves");
        Assert.That(to is { Width: > 0, Height: > 0 }, "a reshape ends on a rectangle that shows", to);

        if (from == to)
        {
            return;
        }

        var (x, y) = (from.Center.X - to.Center.X, from.Center.Y - to.Center.Y);
        var (wide, tall) = (from.Width / to.Width, from.Height / to.Height);
        var held = string.Create(CultureInfo.InvariantCulture, $"translate({x}px, {y}px) scale({wide}, {tall})");
        Run(control, Slides,
            new Setter(Visual.RenderTransformProperty, TransformOperations.Parse(held)),
            new Setter(Visual.RenderTransformProperty, TransformOperations.Parse("none")));
    }

    /// <summary>Fades a control in.</summary>
    public static void Arrive(Control control)
    {
        Assert.NotNull(control, "an arrival needs the control it fades");

        Run(control, Arrivals, new Setter(Visual.OpacityProperty, 0d), new Setter(Visual.OpacityProperty, 1d));
    }

    private static void Run(
        Control control, ConditionalWeakTable<Control, CancellationTokenSource> runs, Setter start, Setter end)
    {
        if (runs.TryGetValue(control, out var running))
        {
            running.Cancel();
        }

        var run = new CancellationTokenSource();
        runs.AddOrUpdate(control, run);

        var motion = new Animation
        {
            Duration = PopupMotion.Duration("MotionMove"),
            Easing = PopupMotion.Ease(),
            FillMode = FillMode.Backward,
            Children =
            {
                new KeyFrame { Cue = new Cue(0), Setters = { start } },
                new KeyFrame { Cue = new Cue(1), Setters = { end } },
            },
        };
        _ = motion.RunAsync(control, run.Token);
    }
}
