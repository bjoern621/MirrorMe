using Avalonia.Controls;
using Avalonia.Controls.Presenters;
using Avalonia.Controls.Primitives;
using Avalonia.Input;
using Avalonia.Media;
using ScreenShare.App.Contracts;

namespace ScreenShare.App.Controls;

/// <summary>
/// Scroll presenter whose wheel moves the step <see cref="WheelScroll"/> sets for what the scroll came from.
/// Avalonia fixes its own at 50 with no property to change it, so the scroll viewer template in
/// <c>Design/Scrolling.axaml</c> puts this one in its place.
///
/// A child scrolling by its own items, snap points and a right-to-left view keep Avalonia's handling,
/// none of the three occurring in this app.
/// </summary>
public sealed class WheelScrollPresenter : ScrollContentPresenter
{
    protected override void OnPointerEntered(PointerEventArgs e)
    {
        base.OnPointerEntered(e);

        if (TopLevel.GetTopLevel(this) is { } top)
        {
            WaylandTouchpad.Attach(top);
        }
    }

    protected override void OnPointerWheelChanged(PointerWheelEventArgs e)
    {
        Assert.NotNull(e, "a wheel event arrives with its arguments");

        if (KeepsAvaloniasWheel())
        {
            base.OnPointerWheelChanged(e);
            return;
        }

        if (Extent.Height <= Viewport.Height && Extent.Width <= Viewport.Width)
        {
            return;
        }

        var offset = WheelScroll.Offset(Offset, e.Delta, Extent, Viewport, e.KeyModifiers == KeyModifiers.Shift, WaylandTouchpad.Source);
        var moved = offset != Offset;
        SetCurrentValue(OffsetProperty, offset);

        // An offset already at its end passes the wheel to the scroll viewer around this one.
        e.Handled = !IsScrollChainingEnabled || moved;
    }

    private bool KeepsAvaloniasWheel()
        => Child is ILogicalScrollable { IsLogicalScrollEnabled: true }
           || HorizontalSnapPointsType != SnapPointsType.None
           || VerticalSnapPointsType != SnapPointsType.None
           || FlowDirection == FlowDirection.RightToLeft;
}
