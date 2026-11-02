using Avalonia.Controls;
using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Fields.ViewModel;
using ScreenShare.App.Features.Shell.Model;

namespace ScreenShare.App.Features.Fields.View;

/// <summary>
/// The info glyph beside a control's name.
/// A press opens its flyout on a wide window and a <see cref="ComparisonSheet"/> below
/// <see cref="NavWidth.LabelsMinWidth"/>, where a flyout would cover the cards it compares.
/// </summary>
public sealed class ComparisonButton : Button
{
    protected override Type StyleKeyOverride => typeof(Button);

    protected override void OnClick()
    {
        var top = Assert.NotNull(TopLevel.GetTopLevel(this), "a pressed control is drawn in a window");
        if (top.Bounds.Width >= NavWidth.LabelsMinWidth)
        {
            base.OnClick();
            return;
        }

        var field = Assert.NotNull(DataContext as FieldViewModel, "the glyph sits on a field");
        var comparison = Assert.NotNull(field.Comparison, "a glyph shows on a field with pictures");
        ComparisonSheet.Show(this, comparison);
    }
}
