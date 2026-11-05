using System.Collections;
using System.Collections.Specialized;
using Avalonia;
using Avalonia.Controls;
using Avalonia.Controls.Templates;
using Avalonia.Metadata;
using Avalonia.Threading;
using ScreenShare.App.Contracts;

namespace ScreenShare.App.Controls;

/// <summary>
/// Draws one control per position, for a list whose rows change under the reader, a pick among them.
/// A changed row lands on the control at its position as a new <c>DataContext</c>, so a pick eases and keeps the focus.
/// <see cref="ItemsControl"/> builds a new container for each new row, and <c>Reconcile.Onto</c> replaces every row
/// of a list that differs, which would land a pick in one frame.
/// </summary>
public sealed class RowList : Decorator
{
    public static readonly StyledProperty<IEnumerable?> ItemsSourceProperty =
        AvaloniaProperty.Register<RowList, IEnumerable?>(nameof(ItemsSource));

    public static readonly StyledProperty<IDataTemplate?> ItemTemplateProperty =
        AvaloniaProperty.Register<RowList, IDataTemplate?>(nameof(ItemTemplate));

    public static readonly StyledProperty<ITemplate<Panel?>?> ItemsPanelProperty =
        AvaloniaProperty.Register<RowList, ITemplate<Panel?>?>(nameof(ItemsPanel));

    private INotifyCollectionChanged? _watched;
    private bool _drawPosted;

    public IEnumerable? ItemsSource
    {
        get => GetValue(ItemsSourceProperty);
        set => SetValue(ItemsSourceProperty, value);
    }

    [InheritDataTypeFromItems(nameof(ItemsSource))]
    public IDataTemplate? ItemTemplate
    {
        get => GetValue(ItemTemplateProperty);
        set => SetValue(ItemTemplateProperty, value);
    }

    /// <summary>Lays the rows out. A list naming no panel stacks them.</summary>
    public ITemplate<Panel?>? ItemsPanel
    {
        get => GetValue(ItemsPanelProperty);
        set => SetValue(ItemsPanelProperty, value);
    }

    protected override void OnPropertyChanged(AvaloniaPropertyChangedEventArgs change)
    {
        base.OnPropertyChanged(change);

        if (change.Property == ItemsSourceProperty && VisualRoot is not null)
        {
            Watch();
            Draw();
        }
    }

    protected override void OnAttachedToVisualTree(VisualTreeAttachmentEventArgs e)
    {
        base.OnAttachedToVisualTree(e);
        Watch();
        Draw();
    }

    protected override void OnDetachedFromVisualTree(VisualTreeAttachmentEventArgs e)
    {
        base.OnDetachedFromVisualTree(e);
        Unwatch();
    }

    private void Watch()
    {
        Unwatch();
        _watched = ItemsSource as INotifyCollectionChanged;
        if (_watched is not null)
        {
            _watched.CollectionChanged += OnRowsChanged;
        }
    }

    private void Unwatch()
    {
        if (_watched is not null)
        {
            _watched.CollectionChanged -= OnRowsChanged;
            _watched = null;
        }
    }

    // A reconcile clears and refills in one pass, so the draw waits for the last row to arrive.
    private void OnRowsChanged(object? sender, NotifyCollectionChangedEventArgs change)
    {
        if (_drawPosted)
        {
            return;
        }

        _drawPosted = true;
        Dispatcher.UIThread.Post(() =>
        {
            _drawPosted = false;
            Draw();
        });
    }

    /// <summary>Writes one control per row and hands each its row. Idempotent.</summary>
    private void Draw()
    {
        var rows = ItemsSource?.Cast<object>().ToList() ?? [];
        var panel = Child as Panel ?? BuildPanel();
        var template = Assert.NotNull(ItemTemplate, "a row list states the template it draws its rows with");

        while (panel.Children.Count > rows.Count)
        {
            panel.Children.RemoveAt(panel.Children.Count - 1);
        }

        while (panel.Children.Count < rows.Count)
        {
            var row = rows[panel.Children.Count];
            panel.Children.Add(Assert.NotNull(template.Build(row), "a row list's template builds a control for every row"));
        }

        for (var i = 0; i < rows.Count; i++)
        {
            panel.Children[i].DataContext = rows[i];
        }

        Assert.That(panel.Children.Count == rows.Count, "one control per row", panel.Children.Count, rows.Count);
    }

    private Panel BuildPanel()
    {
        var panel = ItemsPanel?.Build() ?? new StackPanel();
        Child = panel;
        return panel;
    }
}
