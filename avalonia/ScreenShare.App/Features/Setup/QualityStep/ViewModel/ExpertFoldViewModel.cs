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
    private IReadOnlyList<FieldViewModel> _fields = [];
    private string _open = "";

    public FoldViewModel Fold { get; } = new();

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
        }
        if (!keys.Contains(_open))
        {
            _open = "";
        }

        Reconcile.Onto(Rows, fields.Select(field => new ExpertRow
        {
            Field = field,
            IsOpen = field.Key == _open && field.IsEnabled,
            Toggle = ToggleOf(field.Key),
        }).ToList());
        Fold.Apply(fields.Count);

        Assert.That(Rows.Count(row => row.IsOpen) <= 1, "the fold holds one control open at most", Rows.Count);
    }

    private DelegateCommand ToggleOf(string key)
    {
        if (!_toggles.TryGetValue(key, out var toggle))
        {
            toggle = new DelegateCommand(() => Toggle(key));
            _toggles[key] = toggle;
        }
        return toggle;
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
}
