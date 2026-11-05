using Avalonia;
using Avalonia.Controls;

namespace ScreenShare.App.Controls;

/// <summary>
/// Stacks a list whose rows come and go: a new row fades in and the rows under it slide along.
/// A row growing by its own fold moves its neighbours on every frame already, so only a change to the rows slides.
/// </summary>
public sealed class SlidingStack : StackPanel
{
    private bool _arranged;
    private bool _rowsChanged;

    public SlidingStack()
    {
        Children.CollectionChanged += (_, _) => _rowsChanged = _arranged;
    }

    protected override Size ArrangeOverride(Size finalSize)
    {
        var before = _rowsChanged ? Children.Select(row => (Row: row, row.Bounds)).ToList() : null;
        var arranged = base.ArrangeOverride(finalSize);

        foreach (var (row, from) in before ?? [])
        {
            if (from.Size == default)
            {
                ArrangeMotion.Arrive(row);
            }
            else
            {
                ArrangeMotion.Slide(row, from, row.Bounds);
            }
        }

        _rowsChanged = false;
        _arranged = true;
        return arranged;
    }
}
