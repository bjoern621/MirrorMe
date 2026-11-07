using System.Collections.ObjectModel;
using MirrorMe.App.Contracts;
using MirrorMe.App.Features.Fields.ViewModel;
using MirrorMe.App.Mvvm;

namespace MirrorMe.App.Features.Setup.QualityStep.ViewModel;

/// <summary>
/// One expert control as the fold draws it: closed with its value, or open on the control itself.
/// One per key across passes, written in place, so its container stays and the fold under it moves on a press.
/// </summary>
public sealed class ExpertRow : Observable
{
    private FieldViewModel _field;
    private bool _isOpen;

    public ExpertRow(FieldViewModel field, DelegateCommand toggle)
    {
        Assert.NotNull(field, "a row draws one control");
        Assert.NotNull(toggle, "a row opens on a press");

        _field = field;
        Toggle = toggle;
    }

    public FieldViewModel Field { get => _field; private set => Set(ref _field, value); }

    public bool IsOpen { get => _isOpen; private set => Set(ref _isOpen, value); }

    /// <summary>Opens this row and closes the one open before it, or closes this one where it is open.</summary>
    public DelegateCommand Toggle { get; }

    /// <summary>The row's render function. Idempotent.</summary>
    public void Apply(FieldViewModel field, bool isOpen)
    {
        Assert.NotNull(field, "a row draws one control");
        Assert.That(field.Key == _field.Key, "a row draws the one control it was made for", field.Key, _field.Key);
        Assert.That(!isOpen || field.IsEnabled, "a greyed control never opens", field.Key);

        Field = field;
        IsOpen = isOpen;
    }
}

/// <summary>
/// Fold under the encode questions, every control they leave as a closed row and at most one row open.
///
/// The open row is the reader's and the one state held here, kept across passes like <see cref="FoldViewModel.Shown"/>.
/// A greyed control never opens, its closed row carrying the reason.
/// The open key and the control's row clear once the form stops sending that control, so a field that returns comes back closed.
/// </summary>
public sealed class ExpertFoldViewModel : Observable
{
    private readonly Dictionary<string, ExpertRow> _rows = [];
    private IReadOnlyList<FieldViewModel> _fields = [];
    private string _open = "";

    public FoldViewModel Fold { get; } = new(Copy.Fields.ExpertTitle);

    public ObservableCollection<ExpertRow> Rows { get; } = [];

    /// <summary>The one render function. Idempotent: an unchanged field list writes nothing.</summary>
    public void Apply(IReadOnlyList<FieldViewModel> fields)
    {
        Assert.NotNull(fields, "the fold draws the fields the step leaves it");

        _fields = fields;
        var keys = fields.Select(field => field.Key).ToHashSet();
        foreach (var gone in _rows.Keys.Where(key => !keys.Contains(key)).ToList())
        {
            _rows.Remove(gone);
        }
        if (!keys.Contains(_open))
        {
            _open = "";
        }

        Reconcile.InPlace(Rows, fields.Select(Draw).ToList());
        Fold.Apply(fields);

        Assert.That(Rows.Count(row => row.IsOpen) <= 1, "the fold holds one control open at most", Rows.Count);
    }

    private ExpertRow Draw(FieldViewModel field)
    {
        if (!_rows.TryGetValue(field.Key, out var row))
        {
            var key = field.Key;
            row = new ExpertRow(field, new DelegateCommand(() => Toggle(key)));
            _rows[key] = row;
        }

        row.Apply(field, field.Key == _open && field.IsEnabled);
        return row;
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
