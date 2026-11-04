using System.Collections.ObjectModel;
using System.Globalization;
using ScreenShare.App.Contracts;
using ScreenShare.App.Controls;
using ScreenShare.App.Features.Setup.Model;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Setup.Checks.ViewModel;

/// <summary>
/// Foot of the step column: what the form says that no control on screen already says.
///
/// A line about a control the step on screen draws stands under that control, and here it would be said twice.
/// What remains is a line about a control on another step or another screen, carrying the press that reaches it,
/// a line about the settings as a whole, and why no pipeline builds from the draft.
/// Nothing left, no card.
/// A note reports nothing wrong, so notes fold under one closed disclosure and what needs the reader reads first.
/// </summary>
public sealed class ChecksCardViewModel : Observable
{
    private string _refusal = "";
    private bool _hasRefusal;
    private bool _isShown;
    private bool _notesShown;
    private bool _hasNotes;
    private string _notesCount = "";

    public ChecksCardViewModel()
    {
        ToggleNotes = new DelegateCommand(() => NotesShown = !NotesShown);
    }

    public ObservableCollection<PreflightCheckRow> Lines { get; } = [];

    public ObservableCollection<PreflightCheckRow> Notes { get; } = [];

    /// <summary>Reader-owned: written by the press alone, kept across render passes.</summary>
    public bool NotesShown { get => _notesShown; set => Set(ref _notesShown, value); }

    /// <summary>Flips the notes fold. Two presses are two states, as every disclosure's are.</summary>
    public DelegateCommand ToggleNotes { get; }

    public bool HasNotes { get => _hasNotes; private set => Set(ref _hasNotes, value); }

    /// <summary>Notes behind the fold, beside the disclosure.</summary>
    public string NotesCount { get => _notesCount; private set => Set(ref _notesCount, value); }

    /// <summary>
    /// Why no pipeline can be built from the draft, in the backend's own words, empty while one can.
    /// The string a reader takes into a bug report, so the view draws it selectable.
    /// </summary>
    public string Refusal { get => _refusal; private set => Set(ref _refusal, value); }

    public bool HasRefusal { get => _hasRefusal; private set => Set(ref _hasRefusal, value); }

    public bool IsShown { get => _isShown; private set => Set(ref _isShown, value); }

    /// <summary>
    /// The one render function.
    /// Idempotent: the lines are records reconciled onto, so a pass over one form notifies nothing.
    /// </summary>
    /// <param name="checks">Form's diagnostics as lines, already ranked.</param>
    /// <param name="drawnHere">Whether the step on screen draws the control with this settings key.</param>
    /// <param name="refusal">Why no pipeline builds from the draft, empty while one does.</param>
    public void Apply(IReadOnlyList<PreflightCheckRow> checks, Func<string, bool> drawnHere, string refusal)
    {
        Assert.NotNull(checks, "the card draws the list the form's diagnostics became");
        Assert.NotNull(drawnHere, "the card leaves out what the step on screen draws, so it is told what that is");
        Assert.NotNull(refusal, "the card draws the refusal the summary carried, empty where there is none");

        var shown = checks.Where(check => check.FieldKey.Length == 0 || !drawnHere(check.FieldKey)).ToList();
        Reconcile.Onto(Lines, shown.Where(check => check.State != CheckState.Note).ToList());
        Reconcile.Onto(Notes, shown.Where(check => check.State == CheckState.Note).ToList());
        HasNotes = Notes.Count > 0;
        NotesCount = Notes.Count.ToString(CultureInfo.InvariantCulture);

        Refusal = refusal;
        HasRefusal = refusal.Length > 0;
        IsShown = Lines.Count > 0 || HasNotes || HasRefusal;

        Assert.That(Lines.Count + Notes.Count <= checks.Count, "the card lists a subset of the form's lines", Lines.Count, Notes.Count, checks.Count);
        Assert.That(IsShown || (Lines.Count == 0 && !HasNotes && !HasRefusal), "a hidden card hides nothing it holds", IsShown);
    }
}
