using System.Collections.ObjectModel;
using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Fields.ViewModel;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Setup.QualityStep.ViewModel;

/// <summary>One expert control as the fold draws it: closed with its value, or open on the control itself.</summary>
public sealed record ExpertRow
{
    public required FieldViewModel Field { get; init; }

    public required bool IsOpen { get; init; }

    /// <summary>Opens this row and closes the one open before it, or closes this one where it is open.</summary>
    public required DelegateCommand Toggle { get; init; }

    /// <summary>One per control that opens, true at this row's own place. Empty on a greyed row.</summary>
    public required ValueList<bool> Pips { get; init; }

    /// <summary>"Next: Color range", empty on the last control that opens.</summary>
    public required string NextLabel { get; init; }

    public bool HasNext => NextLabel.Length > 0;

    /// <summary>Opens the control <see cref="NextLabel"/> names.</summary>
    public required DelegateCommand Next { get; init; }
}

/// <summary>
/// Fold under the encode questions, every control they leave as a closed row and at most one row open.
///
/// The open row is the reader's and the one state held here, kept across passes like <see cref="FoldViewModel.Shown"/>.
/// A greyed control never opens, its closed row carrying the reason.
/// The open key clears once the form stops sending that control, so a field that returns comes back closed.
/// </summary>
public sealed class ExpertFoldViewModel : Observable
{
    private readonly Dictionary<string, DelegateCommand> _toggles = [];
    private readonly Dictionary<string, DelegateCommand> _nexts = [];
    private IReadOnlyList<FieldViewModel> _fields = [];
    private string _open = "";

    public FoldViewModel Fold { get; } = new(Copy.Fields.ExpertTitle);

    public ObservableCollection<ExpertRow> Rows { get; } = [];

    /// <summary>The one render function. Idempotent: an unchanged field list rebuilds equal rows.</summary>
    public void Apply(IReadOnlyList<FieldViewModel> fields)
    {
        Assert.NotNull(fields, "the fold draws the fields the step leaves it");

        _fields = fields;
        var keys = fields.Select(field => field.Key).ToHashSet();
        foreach (var gone in _toggles.Keys.Where(key => !keys.Contains(key)).ToList())
        {
            _toggles.Remove(gone);
            _nexts.Remove(gone);
        }
        if (!keys.Contains(_open))
        {
            _open = "";
        }

        var openable = fields.Where(field => field.IsEnabled).ToList();
        Reconcile.Onto(Rows, fields.Select(field => RowOf(field, openable)).ToList());
        Fold.Apply(fields);

        Assert.That(Rows.Count(row => row.IsOpen) <= 1, "the fold holds one control open at most", Rows.Count);
    }

    private ExpertRow RowOf(FieldViewModel field, List<FieldViewModel> openable)
    {
        var place = openable.IndexOf(field);
        var next = place >= 0 && place + 1 < openable.Count ? openable[place + 1] : null;
        return new ExpertRow
        {
            Field = field,
            IsOpen = field.Key == _open && field.IsEnabled,
            Toggle = CommandOf(_toggles, field.Key, () => Toggle(field.Key)),
            Pips = place < 0 ? ValueList<bool>.Empty : new ValueList<bool>(openable.Select((_, index) => index == place)),
            NextLabel = next is null ? "" : Copy.Fields.NextControl(next.Label),
            Next = CommandOf(_nexts, field.Key, () => OpenAfter(field.Key)),
        };
    }

    /// <summary>One command per key across passes, so an unchanged row compares equal.</summary>
    private static DelegateCommand CommandOf(Dictionary<string, DelegateCommand> commands, string key, Action run)
    {
        if (!commands.TryGetValue(key, out var command))
        {
            command = new DelegateCommand(run);
            commands[key] = command;
        }
        return command;
    }

    /// <summary>
    /// Two presses are two states, the departure from idempotency every disclosure makes.
    /// A press on a greyed row closes whatever was open and opens nothing.
    /// </summary>
    private void Toggle(string key)
    {
        var enabled = _fields.Any(field => field.Key == key && field.IsEnabled);
        _open = _open == key || !enabled ? "" : key;
        Apply(_fields);
    }

    /// <summary>Opens the control after <paramref name="key"/> among those that open, a no-op past the last.</summary>
    private void OpenAfter(string key)
    {
        var openable = _fields.Where(field => field.IsEnabled).Select(field => field.Key).ToList();
        var place = openable.IndexOf(key);
        if (place < 0 || place + 1 >= openable.Count)
        {
            return;
        }
        _open = openable[place + 1];
        Apply(_fields);
    }
}
