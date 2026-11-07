using System.Collections.ObjectModel;
using MirrorMe.App.Contracts;

namespace MirrorMe.App.Mvvm;

/// <summary>
/// Converges a bound collection onto the rows a render function derived.
///
/// Clear-then-fill rather than an incremental patch: these lists are short, and a rebuild leaks no handler and
/// repeats without effect (<c>docs/development-principles.md</c>, "Idempotency").
///
/// The guard in front of it makes that safe on every pass.
/// Identity is a row's value at its position, rows being records, so an unchanged reading compares equal
/// and the collection is left untouched.
/// A row that moved therefore rebuilds the list exactly as a row that changed does.
///
/// The guard is correctness and not economy: a rebuild replaces every container, so a per-second pass would
/// reset scroll position, drop a selection, and close a tooltip open under a resting pointer.
/// </summary>
public static class Reconcile
{
    public static void Onto<T>(IList<T> bound, IReadOnlyList<T> rows)
    {
        Assert.NotNull(bound, "a reconcile needs the collection it converges");
        Assert.NotNull(rows, "a reconcile needs the rows it converges onto");

        if (bound.SequenceEqual(rows))
        {
            return;
        }

        bound.Clear();
        foreach (var row in rows)
        {
            bound.Add(row);
        }

        Assert.That(bound.Count == rows.Count, "a bound row per rendered row", bound.Count, rows.Count);
    }

    /// <summary>
    /// Converges by the fewest edits, for a list whose items own controls holding resources.
    /// An item that stays keeps its container, a rebuild detaching it and a tile's frame subscription with it.
    /// Items compare by equality, so an item stays where an equal one is rendered.
    /// </summary>
    public static void InPlace<T>(ObservableCollection<T> bound, IReadOnlyList<T> rows)
    {
        Assert.NotNull(bound, "a reconcile needs the collection it converges");
        Assert.NotNull(rows, "a reconcile needs the rows it converges onto");

        for (var i = bound.Count - 1; i >= 0; i--)
        {
            if (!rows.Contains(bound[i]))
            {
                bound.RemoveAt(i);
            }
        }

        for (var i = 0; i < rows.Count; i++)
        {
            if (i < bound.Count && Equals(bound[i], rows[i]))
            {
                continue;
            }

            var at = IndexFrom(bound, rows[i], i + 1);
            if (at >= 0)
            {
                bound.Move(at, i);
            }
            else
            {
                bound.Insert(i, rows[i]);
            }
        }

        Assert.That(bound.SequenceEqual(rows), "a bound row per rendered row, in its order", bound.Count, rows.Count);
    }

    private static int IndexFrom<T>(IList<T> list, T item, int start)
    {
        for (var i = start; i < list.Count; i++)
        {
            if (Equals(list[i], item))
            {
                return i;
            }
        }

        return -1;
    }
}
