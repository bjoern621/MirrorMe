using ScreenShare.App.Controls;
using ScreenShare.App.Features.Fields.Model;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Fields.ViewModel;

/// <summary>
/// One entry of a select or radio field, as the form described it and the screen draws it.
/// A dropdown carries one row that is not an entry: the disclosure over the refused ones (<see cref="IsReveal"/>).
///
/// Every string here is written on this side: the name and the paragraph are looked up by field
/// and <see cref="Value"/>, the note and the refusal composed from the codes the backend sent.
/// Whether the entry exists and whether it can be picked are not: <see cref="IsEnabled"/>
/// and the code behind <see cref="Reason"/> arrive already decided (<c>docs/ipc-api.md</c>, "The rule").
///
/// A record, so a render pass over an unchanged field produces rows that compare equal
/// and the bound collection is left alone.
/// <see cref="Choose"/> holds the field's own command instance for this value,
/// so two passes compare equal rather than merely alike.
/// </summary>
public sealed record OptionViewModel
{
    /// <summary>
    /// What the settings carry once this entry is picked.
    /// The identifier the backend sent, as <c>hevc_nvenc</c>.
    /// </summary>
    public required string Value { get; init; }

    public required string Label { get; init; }

    /// <summary>Trailing annotation naming what the entry was derived from, empty where there is none.</summary>
    public required string Note { get; init; }

    /// <summary>Paragraph a radio card shows under its title, empty on a select.</summary>
    public required string Detail { get; init; }

    public required bool IsSelected { get; init; }

    /// <summary>False for an entry this combination rules out. It keeps its place in the list, greyed.</summary>
    public required bool IsEnabled { get; init; }

    /// <summary>
    /// Names the limit and which side has it, so a greyed entry says what to change.
    /// Empty while the entry is live.
    /// </summary>
    public required string Reason { get; init; }

    /// <summary>Whether the backend marked the entry worth emphasising for this combination.</summary>
    public required bool IsRecommended { get; init; }

    /// <summary>What the settings cost with this entry picked: "8.7 Mb/s". Empty where the form sent no price.</summary>
    public string Price { get; init; } = "";

    /// <summary>What picking this entry would raise that the field's other entries do not, ranked by the form.</summary>
    public ValueList<Impact> Impacts { get; init; } = ValueList<Impact>.Empty;

    /// <summary>
    /// Disclosure row rather than an entry: no value, <see cref="Choose"/> lists the refused ones instead of picking,
    /// <see cref="IsSelected"/> reads whether they are listed.
    /// Also what keeps the menu open on the press (<c>Controls/Select/Select.axaml</c>).
    /// </summary>
    public required bool IsReveal { get; init; }

    public required DelegateCommand Choose { get; init; }

    /// <summary>
    /// Both lines under the label are drawn where both are present, the refusal first.
    /// A refused entry drawn without its paragraph states a limit and drops what the entry is,
    /// leaving the reader nothing to weigh the limit against.
    /// </summary>
    public bool HasReason => Reason.Length > 0;

    public bool HasDetail => Detail.Length > 0;

    public bool HasNote => Note.Length > 0;

    /// <summary>What a choice prints beside its name: the price, or what the entry was derived from where it carries none.</summary>
    public string Figure => HasPrice ? Price : Note;

    public bool HasPrice => Price.Length > 0;

    public bool HasImpacts => Impacts.Count > 0;

    /// <summary>
    /// Whether the entry carries a mark for what picking it raises.
    /// The picked entry's statements already stand under its control, so it carries none.
    /// </summary>
    public bool ShowsImpactMark => HasImpacts && !IsSelected;

    /// <summary>Rank of the gravest statement, the form ranking them gravest first.</summary>
    public CheckState ImpactState => HasImpacts ? Impacts[0].State : CheckState.Passed;

    public bool ImpactBlocks => ImpactState == CheckState.Blocking;

    public bool ImpactWarns => ImpactState == CheckState.Warned;

    public string ImpactTip => string.Join("\n", Impacts.Select(impact => impact.Text));
}
