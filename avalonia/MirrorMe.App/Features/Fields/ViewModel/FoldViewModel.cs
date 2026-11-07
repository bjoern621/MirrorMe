using MirrorMe.App.Contracts;
using MirrorMe.App.Controls;
using MirrorMe.App.Features.Fields.Model;
using MirrorMe.App.Mvvm;

namespace MirrorMe.App.Features.Fields.ViewModel;

/// <summary>
/// Disclosure over a step's folded controls.
///
/// Closed on every launch, and the reader's press is the one opener:
/// the fold is view state the reader set, like an opened refused list
/// (<see cref="FieldViewModel.RefusedShown"/>), so a render pass keeps it and nothing persists it.
/// The rows behind it stay drawn; opening is visibility alone.
/// </summary>
public sealed class FoldViewModel : Observable
{
    private bool _shown;
    private bool _hasAny;
    private string _count = "";
    private ValueList<Impact> _impacts = ValueList<Impact>.Empty;

    /// <param name="title">Disclosure's words, beside the chevron.</param>
    public FoldViewModel(string title)
    {
        Assert.That(title.Length > 0, "a disclosure names what it opens onto");

        Title = title;
        ToggleCommand = new DelegateCommand(() => Shown = !Shown);
    }

    public string Title { get; }

    /// <summary>Flips the fold. Two presses are two states, the one deliberate departure from idempotency.</summary>
    public DelegateCommand ToggleCommand { get; }

    /// <summary>Reader-owned: written by the press alone.</summary>
    public bool Shown { get => _shown; set => Set(ref _shown, value); }

    /// <summary>Whether the step holds anything to fold. The disclosure row draws only while it does.</summary>
    public bool HasAny { get => _hasAny; private set => Set(ref _hasAny, value); }

    /// <summary>Controls behind the fold, beside the disclosure: the figure says whether opening it is worth the trip.</summary>
    public string Count { get => _count; private set => Set(ref _count, value); }

    /// <summary>
    /// What the controls behind the fold raise, ranked across them.
    /// A closed fold hides the lines under each control, so the disclosure carries their mark.
    /// </summary>
    public ValueList<Impact> Impacts
    {
        get => _impacts;
        private set
        {
            if (Set(ref _impacts, value))
            {
                OnPropertyChanged(nameof(HasImpacts));
                OnPropertyChanged(nameof(ImpactState));
                OnPropertyChanged(nameof(ImpactBlocks));
                OnPropertyChanged(nameof(ImpactWarns));
                OnPropertyChanged(nameof(ImpactTip));
            }
        }
    }

    public bool HasImpacts => Impacts.Count > 0;

    public CheckState ImpactState => HasImpacts ? Impacts[0].State : CheckState.Passed;

    public bool ImpactBlocks => ImpactState == CheckState.Blocking;

    public bool ImpactWarns => ImpactState == CheckState.Warned;

    public string ImpactTip => string.Join("\n", Impacts.Select(impact => impact.Text));

    /// <summary>Render-pass input. Idempotent, and it never writes <see cref="Shown"/>.</summary>
    /// <param name="folded">Controls behind the fold.</param>
    public void Apply(IReadOnlyCollection<FieldViewModel> folded)
    {
        Assert.NotNull(folded, "a fold counts the controls behind it");

        HasAny = folded.Count > 0;
        Count = folded.Count > 0 ? folded.Count.ToString(System.Globalization.CultureInfo.InvariantCulture) : "";
        Impacts = Model.Impacts.Ranked(folded.SelectMany(field => field.Diagnostics));

        Assert.That(HasAny == (Count.Length > 0), "the disclosure and its figure agree", HasAny, Count);
        Assert.That(HasAny || !HasImpacts, "an empty fold raises nothing", Impacts.Count);
    }
}
