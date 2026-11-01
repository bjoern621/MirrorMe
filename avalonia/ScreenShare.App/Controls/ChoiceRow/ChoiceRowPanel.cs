using Avalonia;
using Avalonia.Controls;

namespace ScreenShare.App.Controls;

/// <summary>
/// Lays a question's choices out in equal columns, as many across as <see cref="ChoiceRowLayout.Columns"/> allows,
/// each line as tall as its tallest choice.
/// Nothing moves to another control at a narrow width: the same choices wrap into more lines.
/// </summary>
public sealed class ChoiceRowPanel : Panel
{
    protected override Size MeasureOverride(Size availableSize)
    {
        var width = double.IsInfinity(availableSize.Width) ? 0 : availableSize.Width;
        var columns = ChoiceRowLayout.Columns(width, Children.Count);
        var cell = CellWidth(width, columns);

        var height = 0.0;
        for (var start = 0; start < Children.Count; start += columns)
        {
            var line = 0.0;
            for (var i = start; i < Math.Min(start + columns, Children.Count); i++)
            {
                Children[i].Measure(new Size(cell, double.PositiveInfinity));
                line = Math.Max(line, Children[i].DesiredSize.Height);
            }
            height += line + (start > 0 ? ChoiceRowLayout.Gap : 0);
        }
        return new Size(width, height);
    }

    protected override Size ArrangeOverride(Size finalSize)
    {
        var columns = ChoiceRowLayout.Columns(finalSize.Width, Children.Count);
        var cell = CellWidth(finalSize.Width, columns);

        var top = 0.0;
        for (var start = 0; start < Children.Count; start += columns)
        {
            var end = Math.Min(start + columns, Children.Count);
            var line = 0.0;
            for (var i = start; i < end; i++)
            {
                line = Math.Max(line, Children[i].DesiredSize.Height);
            }
            for (var i = start; i < end; i++)
            {
                var left = (i - start) * (cell + ChoiceRowLayout.Gap);
                Children[i].Arrange(new Rect(left, top, cell, line));
            }
            top += line + ChoiceRowLayout.Gap;
        }
        return finalSize;
    }

    private static double CellWidth(double width, int columns)
        => Math.Max(0, (width - ChoiceRowLayout.Gap * (columns - 1)) / columns);
}
