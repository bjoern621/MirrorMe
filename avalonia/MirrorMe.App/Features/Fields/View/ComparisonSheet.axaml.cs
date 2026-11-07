using Avalonia;
using Avalonia.Controls;
using Avalonia.Controls.Primitives;
using Avalonia.Input;
using Avalonia.Interactivity;
using MirrorMe.App.Contracts;
using MirrorMe.App.Controls;
using MirrorMe.App.Features.Fields.ViewModel;

namespace MirrorMe.App.Features.Fields.View;

/// <summary>
/// A field's comparison over the whole window, for a window too narrow to hang it beside the glyph.
///
/// Open while it sits in the window's overlay layer, the state a flyout keeps in its popup.
/// Done and a press on the dimmed ground both take it out again.
/// </summary>
public sealed partial class ComparisonSheet : UserControl
{
    private IDisposable? _fill;

    public ComparisonSheet()
    {
        InitializeComponent();
    }

    /// <summary>Lays the sheet over the window holding <paramref name="anchor"/>.</summary>
    public static void Show(Visual anchor, ComparisonViewModel comparison)
    {
        Assert.NotNull(comparison, "a sheet draws the comparison it was opened on");

        var layer = Assert.NotNull(OverlayLayer.GetOverlayLayer(anchor), "a pressed control is drawn in a window");
        var sheet = new ComparisonSheet { DataContext = comparison };
        // The layer draws a child no larger than it asks for, so the sheet asks for the layer's size to reach the foot.
        sheet._fill = layer.GetObservable(BoundsProperty).Subscribe(new Avalonia.Reactive.AnonymousObserver<Rect>(bounds =>
        {
            sheet.Width = bounds.Width;
            sheet.Height = bounds.Height;
        }));
        layer.Children.Add(sheet);
    }

    /// <summary>Fades the sheet out, then takes it off the layer. A second call changes nothing.</summary>
    private async Task CloseAsync()
    {
        if (_fill is null)
        {
            return;
        }

        _fill.Dispose();
        _fill = null;
        IsHitTestVisible = false;
        Opacity = 0;
        await Task.Delay(PopupMotion.Duration("MotionOpen"));

        if (Parent is OverlayLayer layer)
        {
            layer.Children.Remove(this);
        }
    }

    private void DonePressed(object? sender, RoutedEventArgs e) => _ = CloseAsync();

    private void ScrimPressed(object? sender, PointerPressedEventArgs e)
    {
        _ = CloseAsync();
        e.Handled = true;
    }
}
