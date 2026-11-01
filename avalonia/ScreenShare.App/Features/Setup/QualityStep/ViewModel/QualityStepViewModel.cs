using System.Collections.ObjectModel;
using ScreenShare.App.Contracts;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Fields.ViewModel;
using ScreenShare.App.Features.Setup.Model;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Setup.QualityStep.ViewModel;

/// <summary>
/// Encode group of the resolved form, laid out as questions over an expert fold instead of by the generic renderer.
///
/// No state of its own.
/// Which entries each control offers, what each costs and raises, and which are greyed
/// arrive through <see cref="FieldGroupViewModel"/> already decided (<c>docs/ipc-api.md</c>, "The rule"),
/// and every write leaves through the field the reader moved.
/// What this class owns is placement: which controls are asked as questions (<see cref="QualityLayout"/>),
/// and the fold holding the rest.
///
/// Outputs only, written by <see cref="Apply"/>, which the flow calls after every pass over the group.
/// </summary>
public sealed class QualityStepViewModel : Observable
{
    private readonly FieldGroupViewModel _group;

    private string _title = "";
    private string _help = "";
    private string _summary = "";
    private bool _isResolved;

    public QualityStepViewModel(FieldGroupViewModel group)
    {
        Assert.NotNull(group, "the encode step draws a group of the resolved form");

        _group = group;
        Apply();
    }

    public ObservableCollection<QuestionRow> Questions { get; } = [];

    public ExpertFoldViewModel Expert { get; } = new();

    public string Title { get => _title; private set => Set(ref _title, value); }

    public string Help { get => _help; private set => Set(ref _help, value); }

    /// <summary>What this step settled on, in the backend's own sentence, as the strip repeats it.</summary>
    public string Summary { get => _summary; private set => Set(ref _summary, value); }

    /// <summary>
    /// Whether the form carries this group at all.
    /// False before the first resolve and for a backend that does not describe the step,
    /// which then draws nothing: an unreachable backend is reported once, above the column.
    /// </summary>
    public bool IsResolved { get => _isResolved; private set => Set(ref _isResolved, value); }

    /// <summary>
    /// The one render function.
    /// Safe to run twice: the group hands back the same field view models by key,
    /// so an unchanged pass builds equal rows and fires no binding.
    /// </summary>
    public void Apply()
    {
        IsResolved = _group.IsResolved;
        Title = _group.Title;
        Help = _group.Help;
        Summary = _group.Summary;

        Reconcile.Onto(Questions, _group.Fields
            .Where(QualityLayout.AsQuestion)
            .Select(field => new QuestionRow { Heading = Cards.Question(field.Key), Field = field })
            .ToList());
        Expert.Apply(_group.Fields.Where(QualityLayout.InFold).ToList());

        Assert.That(
            Questions.Count + Expert.Rows.Count == _group.Fields.Count,
            "the questions and the fold draw every control of the group once", Questions.Count, Expert.Rows.Count);
        Assert.That(IsResolved || Questions.Count == 0, "a step the form did not describe asks nothing", Questions.Count);
    }
}
