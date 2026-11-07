using System.Collections;

namespace MirrorMe.App.Mvvm;

/// <summary>
/// Read-only list that compares by its items, for a list held inside a row record.
/// A plain list compares by reference, so every pass would build a row unequal to the last
/// and <see cref="Reconcile"/> would rebuild a collection that did not change.
/// </summary>
public sealed class ValueList<T> : IReadOnlyList<T>, IEquatable<ValueList<T>>
{
    public static readonly ValueList<T> Empty = new([]);

    private readonly T[] _items;

    public ValueList(IEnumerable<T> items) => _items = items.ToArray();

    public int Count => _items.Length;

    public T this[int index] => _items[index];

    public IEnumerator<T> GetEnumerator() => ((IEnumerable<T>)_items).GetEnumerator();

    IEnumerator IEnumerable.GetEnumerator() => _items.GetEnumerator();

    public bool Equals(ValueList<T>? other) => other is not null && _items.SequenceEqual(other._items);

    public override bool Equals(object? obj) => Equals(obj as ValueList<T>);

    public override int GetHashCode()
    {
        var hash = new HashCode();
        foreach (var item in _items)
        {
            hash.Add(item);
        }
        return hash.ToHashCode();
    }
}
