using System.Collections.ObjectModel;
using ScreenShare.App.Backend;
using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Fields.Model;
using ScreenShare.App.Features.Fields.ViewModel;
using ScreenShare.App.Features.Viewer.WatchSettings.Model;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Viewer.WatchSettings.ViewModel;

/// <summary>
/// How this machine receives: the protocol a stream comes back on, the windows a receiver holds packets in,
/// and the chain a tile converts decoded frames with.
///
/// <b>Drawn over the tiles it governs.</b> The wizard configures what this machine <i>sends</i>,
/// and a reader who only watches has no business in the sharing flow (<see cref="GroupPlacement"/>).
///
/// <b>Kept as it is set.</b> The form marks the group applied, a receive reading the stored settings
/// and being handed none, so the panel carries no commit (<c>docs/settings-editing.md</c>, "Staged and applied").
/// The draft belongs to the window: <see cref="FormSession"/> holds it and writes an applied field through.
///
/// <b>A change reaches the next decode.</b> A receive pipeline takes no value back once it is built,
/// so a tile on screen keeps the protocol and the chain it opened with.
///
/// <b>Outputs</b> only, written by <see cref="Apply"/> on every pass.
/// </summary>
public sealed class WatchSettingsViewModel : Observable
{
    private readonly FormSession _form;
    private readonly Session _session;

    /// <summary>One reset command per group key, kept so a pass produces an action equal to the last.</summary>
    private readonly Dictionary<string, DelegateCommand> _reset = [];

    /// <param name="form">
    /// Draft this window holds, and where a write leaves through.
    /// The panel keeps no copy: the group renders what the session answers on each pass.
    /// </param>
    /// <param name="session">
    /// Names the entries the group offers, the vocabulary being the backend's and the words this side's.
    /// </param>
    /// <param name="close">Shuts the panel. Owned by the holding screen, whose arrangement the panel is part of.</param>
    public WatchSettingsViewModel(FormSession form, Session session, Action close)
    {
        Assert.NotNull(form, "a watch panel draws the draft the window is holding");
        Assert.NotNull(session, "a watch panel needs the vocabulary that names its entries");
        Assert.NotNull(close, "a watch panel needs a way to say it is done");

        _form = form;
        _session = session;

        Group = new FieldGroupViewModel(
            _form.Write, groupActionOf: ResetOf, sweep: _form.Sweeping, onFloor: WatchLayout.OnFloor);

        // Plain: dismissal costs no round trip and cannot be refused.
        CloseCommand = new DelegateCommand(close);

        // Nothing subscribed to here.
        // The holding screen renders this panel from its own pass: one notification, one render.
        Apply();
    }

    // --- Outputs ------------------------------------------------------------------

    private string _notice = "";
    private bool _hasNotice;

    public FieldGroupViewModel Group { get; }

    /// <summary>
    /// One card per way a stream can arrive, over the fold.
    /// Each is the protocol field's own entry under a plain name, so it is picked, greyed and chosen as that entry is,
    /// and a card and the fold cannot disagree.
    /// </summary>
    public ObservableCollection<OptionViewModel> Arrivals { get; } = [];

    public DelegateCommand CloseCommand { get; }

    public string CloseTip => "Closes this panel. Every setting here is already kept.";

    /// <summary>Backend's sentence about the last write, empty where there is none.</summary>
    public string Notice { get => _notice; private set => Set(ref _notice, value); }

    public bool HasNotice { get => _hasNotice; private set => Set(ref _hasNotice, value); }

    /// <summary>
    /// Reset beside the heading.
    /// An applied group has no other way back, its fields being what this machine already is.
    /// The values are the form's, stated per field, so no default is named here.
    /// Composed per pass around the held command, so two passes produce actions that compare equal.
    /// </summary>
    private GroupAction ResetOf(Api.V1.FieldGroup group) => new(
        "Reset to defaults",
        "Puts every setting here back to the value a fresh installation starts with.",
        ResetCommandOf(group.Key));

    private DelegateCommand ResetCommandOf(string key)
    {
        Assert.That(key.Length > 0, "a reset command is identified by the group it puts back");

        if (_reset.TryGetValue(key, out var command))
        {
            return command;
        }

        command = new DelegateCommand(() => _form.Reset(key));
        _reset[key] = command;
        return command;
    }

    /// <summary>
    /// The one render function.
    /// Safe to run twice: an unmoved draft asks for no resolve,
    /// and the group's own pass produces fields that compare equal, so a repeated pass fires no binding.
    /// </summary>
    public void Apply()
    {
        // Reconciled from the pass rather than performed by it: the pass names the state it wants, a resolved form.
        _form.Sync();

        var form = _form.Form;
        var group = GroupOf(form);
        Group.Apply(group, _session.Words, form?.Settings, _form.IsAnswered, form?.Diagnostics);
        Reconcile.Onto(Arrivals, CardsOf(Group.Visible(WatchLayout.ProtocolKey)));

        // Read through: this panel and the wizard write one settings message down one queue,
        // so what could not be stored is a single sentence and never one per screen (FormSession.Unsaved).
        Notice = _form.Unsaved;
        HasNotice = Notice.Length > 0;

        Assert.That(HasNotice == (Notice.Length > 0), "the notice and its sentence agree", Notice);
        Assert.That(Arrivals.Count(card => card.IsSelected) <= 1, "one card at most is picked", Arrivals.Count);
    }

    /// <summary>
    /// The protocol field's entries that a card stands for, in card order and renamed.
    /// A card whose protocol the form does not offer is absent, a card writing a value the field lacks being a write
    /// the form refuses. Empty while the group draws no protocol field.
    /// </summary>
    private static List<OptionViewModel> CardsOf(FieldViewModel? protocol)
    {
        if (protocol is null)
        {
            return [];
        }

        var cards = new List<OptionViewModel>(Copy.Arrivals.All.Count);
        foreach (var arrival in Copy.Arrivals.All)
        {
            var entry = protocol.Options.FirstOrDefault(option => option.Value == arrival.Value);
            if (entry is not null)
            {
                cards.Add(entry with { Label = arrival.Name, Detail = arrival.Sentence, Note = "" });
            }
        }

        return cards;
    }

    /// <summary>
    /// Watch group of the resolved form, null where the form carries none.
    /// Found by the key deciding which screen draws a group, so this panel and the wizard's filter read one table
    /// (<see cref="GroupPlacement"/>).
    /// </summary>
    private static Api.V1.FieldGroup? GroupOf(Api.V1.Form? form)
    {
        if (form is null)
        {
            return null;
        }

        foreach (var group in form.Groups)
        {
            if (GroupPlacement.InViewer(group.Key))
            {
                return group;
            }
        }

        return null;
    }
}
