using System.ComponentModel;
using System.Runtime.CompilerServices;
using Avalonia;
using Avalonia.Animation;
using Avalonia.Animation.Easings;
using Avalonia.Controls;
using Avalonia.Controls.Primitives;
using Avalonia.Input;
using Avalonia.Media.Transformation;
using Avalonia.Styling;
using ScreenShare.App.Contracts;

namespace ScreenShare.App.Controls;

/// <summary>
/// Moves every floating surface in and out: a menu, a dropdown's list, a submenu, a tooltip.
/// Avalonia shows and hides a popup in one frame, so the motion hooks both ends.
/// Every popup opening plays its content in.
/// A flyout asked to close cancels that close once, plays its presenter out, and closes again.
/// A submenu and a tooltip close in one frame: a bare popup's cancellable close is internal to Avalonia.
/// Durations come from <c>Design/Metrics.axaml</c>.
/// </summary>
public static class PopupMotion
{
    private const string Shrunk = "scale(0.96)";

    private static bool _installed;

    // A flyout whose exit has played, so the close it repeats goes through.
    private static readonly ConditionalWeakTable<PopupFlyoutBase, object> Leaving = new();

    // The exit playing on a surface, cancelled when the surface opens again before it ends.
    private static readonly ConditionalWeakTable<Control, CancellationTokenSource> Exits = new();

    /// <summary>Hooks every popup in the process. A second call changes nothing.</summary>
    public static void Install()
    {
        if (_installed)
        {
            return;
        }

        _installed = true;
        Popup.IsOpenProperty.Changed.AddClassHandler<Popup>(OnPopupIsOpen);
        FlyoutBase.IsOpenProperty.Changed.AddClassHandler<PopupFlyoutBase>(OnFlyoutIsOpen);
    }

    private static void OnPopupIsOpen(Popup popup, AvaloniaPropertyChangedEventArgs change)
    {
        if (!change.GetNewValue<bool>() || popup.Child is not { } surface)
        {
            return;
        }

        Enter(surface);
    }

    // Subscribed on every open: a flyout is built by whoever declares it, and an open is the first moment
    // this class sees one. The unsubscribe keeps the handler single.
    private static void OnFlyoutIsOpen(PopupFlyoutBase flyout, AvaloniaPropertyChangedEventArgs change)
    {
        if (!change.GetNewValue<bool>())
        {
            return;
        }

        Leaving.Remove(flyout);
        flyout.Closing -= OnFlyoutClosing;
        flyout.Closing += OnFlyoutClosing;
    }

    private static void OnFlyoutClosing(object? sender, CancelEventArgs close)
    {
        var flyout = Assert.NotNull(sender as PopupFlyoutBase, "only a popup flyout subscribes to its own close");
        if (Leaving.TryGetValue(flyout, out _))
        {
            Leaving.Remove(flyout);
            return;
        }

        if (flyout.Popup.Child is not { } surface)
        {
            return;
        }

        close.Cancel = true;

        // A second dismiss during the fade leaves it running: a new transition list would drop the fade in one frame.
        if (!Exits.TryGetValue(surface, out _))
        {
            _ = LeaveAsync(flyout, surface);
        }
    }

    /// <summary>
    /// Fades and grows the surface into place.
    /// Clears what an exit left behind first, with no transitions attached, so the clearing itself does not animate.
    /// </summary>
    private static void Enter(Control surface)
    {
        if (Exits.TryGetValue(surface, out var exit))
        {
            exit.Cancel();
            Exits.Remove(surface);
        }

        surface.Transitions = null;
        surface.ClearValue(Visual.OpacityProperty);
        surface.ClearValue(Visual.RenderTransformProperty);
        surface.ClearValue(InputElement.IsHitTestVisibleProperty);

        var enter = new Animation
        {
            Duration = Duration("MotionOpen"),
            Easing = Ease(),
            FillMode = FillMode.Backward,
            Children =
            {
                Frame(0, opacity: 0, Shrunk),
                Frame(1, opacity: 1, "none"),
            },
        };
        _ = enter.RunAsync(surface);
    }

    /// <summary>
    /// Fades the surface out and hides the flyout once the fade ends.
    /// Driven by transitions toward local values rather than by a keyframe run:
    /// a finished run hands its property back to the value under it, which would draw the menu whole for a frame
    /// before the hide lands.
    /// </summary>
    private static async Task LeaveAsync(PopupFlyoutBase flyout, Control surface)
    {
        var duration = Duration("MotionClose");
        var easing = Ease();
        var run = new CancellationTokenSource();
        Exits.AddOrUpdate(surface, run);

        surface.IsHitTestVisible = false;
        surface.Transitions =
        [
            new DoubleTransition { Property = Visual.OpacityProperty, Duration = duration, Easing = easing },
            new TransformOperationsTransition { Property = Visual.RenderTransformProperty, Duration = duration, Easing = easing },
        ];
        surface.Opacity = 0;
        surface.RenderTransform = TransformOperations.Parse(Shrunk);

        try
        {
            await Task.Delay(duration, run.Token);
        }
        catch (TaskCanceledException)
        {
            // Opened again before the fade ended, which Enter has already played back in.
            return;
        }

        Exits.Remove(surface);
        Leaving.AddOrUpdate(flyout, flyout);
        flyout.Hide();
    }

    private static KeyFrame Frame(double cue, double opacity, string transform) => new()
    {
        Cue = new Cue(cue),
        Setters =
        {
            new Setter(Visual.OpacityProperty, opacity),
            new Setter(Visual.RenderTransformProperty, TransformOperations.Parse(transform)),
        },
    };

    internal static TimeSpan Duration(string key)
    {
        var found = Application.Current?.FindResource(key);
        Assert.That(found is TimeSpan, "Design/Metrics.axaml declares each motion duration as a TimeSpan", key);
        return (TimeSpan)found;
    }

    internal static Easing Ease()
        => Assert.NotNull(
            Application.Current?.FindResource("MotionEase") as Easing,
            "Design/Metrics.axaml declares MotionEase as an easing");
}
