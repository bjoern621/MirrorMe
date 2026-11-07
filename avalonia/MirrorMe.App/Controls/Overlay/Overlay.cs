using Avalonia;
using Avalonia.Controls;
using Avalonia.Threading;
using MirrorMe.App.Contracts;

namespace MirrorMe.App.Controls;

/// <summary>
/// Holds a surface over the window, a dialog's dimmed ground or a panel, and fades it in as it opens and out as it closes.
/// The body stays drawn until the fade ends, and a closing overlay lets presses through to the window.
/// An overlay that <see cref="TakesFocus"/> takes the focus as it opens, so a key binding on the view around it
/// hears Escape before the window does.
///
/// <see cref="IsOpen"/> names the state wanted.
/// <see cref="Openness"/> follows it under the transition that <c>Controls/Overlay/Overlay.axaml</c> sets.
/// </summary>
public sealed class Overlay : Decorator
{
    public static readonly StyledProperty<bool> IsOpenProperty =
        AvaloniaProperty.Register<Overlay, bool>(nameof(IsOpen));

    /// <summary>Opacity the overlay draws at: 0 closed, 1 open.</summary>
    public static readonly StyledProperty<double> OpennessProperty =
        AvaloniaProperty.Register<Overlay, double>(nameof(Openness));

    /// <summary>Whether opening moves the focus here: set on a dialog, left off on a panel beside the content.</summary>
    public static readonly StyledProperty<bool> TakesFocusProperty =
        AvaloniaProperty.Register<Overlay, bool>(nameof(TakesFocus));

    static Overlay()
    {
        FocusableProperty.OverrideDefaultValue<Overlay>(true);
    }

    public Overlay()
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

    public bool TakesFocus
    {
        get => GetValue(TakesFocusProperty);
        set => SetValue(TakesFocusProperty, value);
    }

    protected override void OnPropertyChanged(AvaloniaPropertyChangedEventArgs change)
    {
        base.OnPropertyChanged(change);

        if (change.Property == IsOpenProperty)
        {
            Openness = IsOpen ? 1 : 0;
            TakeFocus();
        }

        if (change.Property == IsOpenProperty || change.Property == OpennessProperty || change.Property == ChildProperty)
        {
            Draw();
        }
    }

    protected override void OnAttachedToVisualTree(VisualTreeAttachmentEventArgs e)
    {
        base.OnAttachedToVisualTree(e);
        TakeFocus();
        LayOutAhead();
    }

    /// <summary>
    /// Lays a closed body out once at zero opacity, then hides it again.
    /// A body's first layout outlasts the fade, so without this the first open after start lands in one frame.
    /// </summary>
    private void LayOutAhead()
    {
        if (IsOpen || Child is not { } body)
        {
            return;
        }

        body.IsVisible = true;
        Dispatcher.UIThread.Post(Draw, DispatcherPriority.Background);
    }

    /// <summary>Posted past layout, a control not yet arranged refusing the focus.</summary>
    private void TakeFocus()
    {
        if (IsOpen && TakesFocus)
        {
            Dispatcher.UIThread.Post(() => Focus(), DispatcherPriority.Loaded);
        }
    }

    /// <summary>Writes the body's presence, the fade and the hit test from the two properties. Idempotent.</summary>
    private void Draw()
    {
        Assert.That(Openness is >= 0 and <= 1, "openness runs from 0 closed to 1 open, the easing never overshooting", Openness);

        Opacity = Openness;
        IsHitTestVisible = IsOpen;

        if (Child is { } body)
        {
            body.IsVisible = IsOpen || Openness > 0;
        }
    }
}
