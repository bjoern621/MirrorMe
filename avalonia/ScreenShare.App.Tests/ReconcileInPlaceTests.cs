using System.Collections.ObjectModel;
using System.Collections.Specialized;
using ScreenShare.App.Mvvm;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A bound list whose items own controls that hold resources, a tile's frame subscription among them.
/// Taking an item out and putting it back detaches its control, so an item staying in the list is never taken out.
/// </summary>
public sealed class ReconcileInPlaceTests
{
    private static List<string> Removed(ObservableCollection<string> bound, IReadOnlyList<string> rows)
    {
        var removed = new List<string>();
        bound.CollectionChanged += (_, change) =>
        {
            if (change.Action is NotifyCollectionChangedAction.Remove or NotifyCollectionChangedAction.Move
                or NotifyCollectionChangedAction.Replace or NotifyCollectionChangedAction.Reset)
            {
                removed.AddRange(change.OldItems?.Cast<string>() ?? ["reset"]);
            }
        };

        Reconcile.InPlace(bound, rows);
        return removed;
    }

    [Fact]
    public void AnItemThatStaysIsNeverTakenOut()
    {
        var bound = new ObservableCollection<string>(["a", "b", "c"]);

        var removed = Removed(bound, ["b", "x", "c", "d"]);

        Assert.Equal(["b", "x", "c", "d"], bound);
        Assert.Equal(["a"], removed);
    }

    [Fact]
    public void AnUnchangedListIsLeftAlone()
    {
        var bound = new ObservableCollection<string>(["a", "b"]);

        Assert.Empty(Removed(bound, ["a", "b"]));
    }
}
