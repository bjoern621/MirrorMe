using Google.Protobuf;
using MirrorMe.App.Backend;
using MirrorMe.App.Contracts;
using MirrorMe.App.Copy;
using MirrorMe.App.Features.Fields.Model;
using MirrorMe.App.Features.Fields.ViewModel;
using MirrorMe.App.Features.Shell.Settings.Model;
using MirrorMe.App.Features.Shell.Update.ViewModel;
using MirrorMe.App.Mvvm;

namespace MirrorMe.App.Features.Shell.Settings.ViewModel;

/// <summary>
/// What the app does for itself: what it reports, what it reads on start, where its logs are,
/// whether an icon sits in the tray, which account it follows, which build it is,
/// and whether the synthetic set runs beside it.
///
/// <b>Drawn over the window rather than in a destination.</b> None of it is about a stream, so no surface
/// configuring one owns it, and a reader who only watches reaches it all the same
/// (<see cref="GroupPlacement"/>).
///
/// <b>No commit.</b> The group is applied, so a control is the write and the next start reads it
/// (<c>docs/settings-editing.md</c>, "Staged and applied").
///
/// <b>The draft belongs to the window.</b> <see cref="FormSession"/> holds it, as it does for the wizard
/// and the watch panel, so this dialog's writes and a publish commit are writes of one message.
///
/// <b>Every read-back is read through.</b> The build, the link and the published release are the session's
/// and the update view model's, so the dialog and the status band cannot disagree.
///
/// <b>Outputs</b> only, written by <see cref="Apply"/> on every pass.
/// </summary>
public sealed class AppSettingsViewModel : Observable
{
    /// <summary>
    /// The keys this dialog places by name,
    /// under the heading each belongs to rather than in the group's own order (<see cref="Apply"/>).
    /// </summary>
    private const string CrashReportsKey = "app.send_crash_reports";
    private const string CheckUpdatesOnStartKey = "app.check_updates_on_start";
    private const string TrayIconKey = "app.tray_icon";
    private const string AutostartKey = "app.autostart";
    private const string TestStreamsKey = "app.test_streams";
    private const string RichPresenceKey = "app.discord_rich_presence";

    private readonly FormSession _form;
    private readonly Session _session;
    private readonly UpdateViewModel _updates;
    private readonly Action<Action> _dispatch;

    /// <summary>One reset command per group key, kept so a pass produces an action equal to the last.</summary>
    private readonly Dictionary<string, DelegateCommand> _reset = [];

    /// <summary>
    /// Links this install to a Discord account.
    /// The round trip holds while the person clicks through the browser leg,
    /// so the button waits it out rather than sitting still.
    /// </summary>
    private readonly PendingCommand _linkDiscord;

    /// <summary>
    /// Hands this install's link back, the secret leaving the stored settings.
    /// A round trip that reaches nothing outside this machine, so the wait is one call long.
    /// </summary>
    private readonly PendingCommand _unlinkDiscord;

    /// <summary>What the last press on either button answered where it failed, empty otherwise.</summary>
    private string _linkFailed = "";

    /// <summary>
    /// What the last report press answered: the name it was stored under, or the refusal it came back with.
    /// Both empty before the first press, and cleared by the next one, a line about a send being about
    /// the send it is standing under.
    /// </summary>
    private string _reportId = "";
    private string _reportFailure = "";

    /// <param name="backend">Reaches the log directory, which is the backend's own (<c>docs/ipc-api.md</c>).</param>
    /// <param name="form">Draft this window holds, and where a write leaves through.</param>
    /// <param name="session">Running state: the build this window talks to, and the Discord link.</param>
    /// <param name="updates">
    /// What the app knows about the published release, owned once for the window and drawn by the band as well.
    /// </param>
    /// <param name="dispatch">Hands an answer back to the UI loop.</param>
    public AppSettingsViewModel(
        IBackend backend, FormSession form, Session session, UpdateViewModel updates, Action<Action> dispatch)
    {
        Assert.NotNull(backend, "app settings reach the backend for the files it owns");
        Assert.NotNull(form, "app settings draw the draft the window is holding");
        Assert.NotNull(session, "app settings read what the backend answered about this machine");
        Assert.NotNull(updates, "app settings draw the one answer about the published release");
        Assert.NotNull(dispatch, "app settings marshal an answer back to the UI loop");

        _form = form;
        _session = session;
        _updates = updates;
        _dispatch = dispatch;

        // The update toggle's row prints no reason: it and the check button are both inert for one fact,
        // which the Updates heading states once above them (UpdatesRefusal).
        Group = new FieldGroupViewModel(
            _form.Write,
            groupActionOf: ResetOf,
            reasonStatedAbove: key => key == CheckUpdatesOnStartKey);

        // Named states rather than one toggle: a press asks for open or closed, and a second press
        // asks for what already holds (docs/development-principles.md, "Idempotency").
        OpenCommand = new DelegateCommand(() => Show(true));
        CloseCommand = new DelegateCommand(() => Show(false));
        BackCommand = new DelegateCommand(() => Go(_place with { AtList = true }));

        // Pressed from a check line about the link, which is drawn behind this dialog
        // (Features/Setup/ViewModel/SetupViewModel.cs).
        OpenAtDiscordCommand = new DelegateCommand(() => Show(true, SettingsSection.Discord));

        // The logs are the backend's files: it writes them, rotates them, and is the side that knows
        // which still exist.
        OpenLogsFolder = new PendingCommand(() => backend.OpenLogsFolderAsync(), dispatch);

        // The bundle is the backend's to build: it owns the logs, the settings it blanks the secrets out of,
        // and the relay the report goes to.
        // Nothing gates the press. A machine with no relay is refused by the backend, in its own words.
        SendReport = new PendingCommand(() => SendReportAsync(backend), dispatch);

        // The manager runs beside the relay, so a machine pointed at none has nothing to link against.
        _linkDiscord = new PendingCommand(
            () => LinkDiscordAsync(backend), dispatch, () => LinkRefusal().Length == 0);

        // Nothing gates the press: dropping a link reaches no manager, and an install holding none
        // is left holding none. The link it drops is what the button stands beside (IsDiscordLinked).
        _unlinkDiscord = new PendingCommand(() => UnlinkDiscordAsync(backend), dispatch);

        // News that the draft, or the form behind it, moved.
        // The button's own ground is in the draft, so a relay typed on a wizard step reaches this pass.
        _form.Changed += Apply;

        Apply();
    }

    // --- Outputs ------------------------------------------------------------------

    private bool _isOpen;
    private SettingsSection _section;
    private string _version = "";
    private string _discordLine = "";
    private bool _isDiscordLinked;
    private bool _discordLineIsFailure;
    private ByteString _discordAvatar = ByteString.Empty;
    private string _linkLabel = "";
    private string _linkTip = "";
    private string _linkHint = "";
    private string _updatesRefusal = "";
    private bool _updatesRefused;

    /// <summary>Whether the dialog stands over the window.</summary>
    public bool IsOpen { get => _isOpen; private set => Set(ref _isOpen, value); }

    /// <summary>Section on screen. Written by every open, so a plain one cannot land where the last press stood.</summary>
    public SettingsSection Section
    {
        get => _section;
        private set
        {
            if (Set(ref _section, value))
            {
                OnPropertyChanged(nameof(SelectedTab));
            }
        }
    }

    public IReadOnlyList<SettingsTab> Tabs => SettingsSections.All;

    /// <summary>Entries above the divider.</summary>
    public IReadOnlyList<SettingsTab> MainTabs { get; } = [.. SettingsSections.All.Where(tab => !tab.SetApart)];

    /// <summary>Entries under the divider.</summary>
    public IReadOnlyList<SettingsTab> ApartTabs { get; } = [.. SettingsSections.All.Where(tab => tab.SetApart)];

    private bool _showsDevelopment;

    /// <summary>
    /// Whether the divider and the entries under it show.
    /// A released backend hides the test-stream toggle, which leaves the Development section empty.
    /// </summary>
    public bool ShowsDevelopment { get => _showsDevelopment; private set => Set(ref _showsDevelopment, value); }

    /// <summary>
    /// List entry of <see cref="Section"/>, read through.
    /// None while a compact dialog stands on the list, so a press on any entry picks it, the current one included.
    /// A null the list writes while it rebuilds asks for nothing.
    /// </summary>
    public SettingsTab? SelectedTab
    {
        get => _place is { Compact: true, AtList: true } ? null : Tabs.Single(tab => tab.Section == Section);
        set
        {
            if (value is not null)
            {
                Section = value.Section;
                Go(_place with { AtList = false });
            }
        }
    }

    private SettingsPlace _place;
    private bool _showsList = true;
    private bool _showsSection = true;
    private bool _showsBack;
    private string _heading = AppSettingsCopy.Title;
    private string _question = "";
    private string _lead = "";
    private bool _isCompact;

    public bool ShowsList { get => _showsList; private set => Set(ref _showsList, value); }

    public bool ShowsSection { get => _showsSection; private set => Set(ref _showsSection, value); }

    /// <summary>Whether the header carries the arrow back to the list.</summary>
    public bool ShowsBack { get => _showsBack; private set => Set(ref _showsBack, value); }

    /// <summary>Dialog title, or the section's name where the section stands alone.</summary>
    public string Heading { get => _heading; private set => Set(ref _heading, value); }

    /// <summary>Question the section on screen opens on.</summary>
    public string Question { get => _question; private set => Set(ref _question, value); }

    public string Lead { get => _lead; private set => Set(ref _lead, value); }

    /// <summary>Whether the dialog fills a window too narrow for the list beside a section.</summary>
    public bool IsCompact { get => _isCompact; private set => Set(ref _isCompact, value); }

    /// <summary>Returns a compact dialog from its section to the list.</summary>
    public DelegateCommand BackCommand { get; }

    /// <summary>States whether the window is too narrow for the list beside a section. Idempotent.</summary>
    public void SetCompact(bool compact) => Go(_place with { Compact = compact });

    private void Go(SettingsPlace place)
    {
        _place = place;
        var tab = Tabs.Single(candidate => candidate.Section == Section);
        IsCompact = place.Compact;
        ShowsList = place.ShowsList;
        ShowsSection = place.ShowsSection;
        ShowsBack = place.ShowsBack;
        Heading = place.ShowsBack ? tab.Label : AppSettingsCopy.Title;
        Question = tab.Question;
        Lead = tab.Lead;
        OnPropertyChanged(nameof(SelectedTab));

        Assert.That(ShowsList || ShowsSection, "the dialog draws the list, the section or both", place);
    }

    public DelegateCommand OpenCommand { get; }

    /// <summary>Opens at the Discord heading, where the link button stands.</summary>
    public DelegateCommand OpenAtDiscordCommand { get; }

    public DelegateCommand CloseCommand { get; }

    public PendingCommand OpenLogsFolder { get; }

    /// <summary>Sends one report to the relay, and answers under <see cref="ReportLine"/>.</summary>
    public PendingCommand SendReport { get; }

    private string _reportLine = "";
    private bool _reportLineIsFailure;

    /// <summary>
    /// What the last report press answered, empty before the first one.
    /// Drawn selectable, the stored name being what a reader carries into a bug report and a refusal
    /// what they carry into a message about the send itself.
    /// </summary>
    public string ReportLine { get => _reportLine; private set => Set(ref _reportLine, value); }

    /// <summary>Whether <see cref="ReportLine"/> is a refusal rather than a name.</summary>
    public bool ReportLineIsFailure { get => _reportLineIsFailure; private set => Set(ref _reportLineIsFailure, value); }

    /// <summary>The app group of the resolved form, holding the fields below and their write path.</summary>
    public FieldGroupViewModel Group { get; }

    private FieldViewModel? _crashReports;
    private FieldViewModel? _checkUpdatesOnStart;
    private FieldViewModel? _trayIcon;
    private FieldViewModel? _autostart;
    private FieldViewModel? _testStreams;
    private FieldViewModel? _richPresence;

    /// <summary>Placed under the Logs heading, beside <see cref="OpenLogsFolder"/>.</summary>
    public FieldViewModel? CrashReports { get => _crashReports; private set => Set(ref _crashReports, value); }

    /// <summary>Placed under the Updates heading, beside <see cref="Updates"/>'s own check.</summary>
    public FieldViewModel? CheckUpdatesOnStart
    {
        get => _checkUpdatesOnStart;
        private set => Set(ref _checkUpdatesOnStart, value);
    }

    /// <summary>
    /// Placed in the Window section, the icon deciding what the window's close button does
    /// (<c>avalonia/README.md</c>, "The tray").
    /// The shell converges the icon on this write, so it comes and goes with the toggle
    /// (<c>Features/Tray/View/TrayIconHost.cs</c>).
    /// </summary>
    public FieldViewModel? TrayIcon { get => _trayIcon; private set => Set(ref _trayIcon, value); }

    /// <summary>
    /// Placed in the Window section above <see cref="TrayIcon"/>.
    /// The shell converges the sign-in entry on this write (<c>Features/Shell/Model/AutostartEntry.cs</c>).
    /// </summary>
    public FieldViewModel? Autostart { get => _autostart; private set => Set(ref _autostart, value); }

    /// <summary>
    /// Placed under the Development heading, the synthetic set being a testing aid rather than something
    /// a stream is configured with.
    /// The backend converges the set on this write, so the streams come and go with the toggle
    /// (<c>backend/internal/app/teststreams.go</c>).
    /// </summary>
    public FieldViewModel? TestStreams { get => _testStreams; private set => Set(ref _testStreams, value); }

    /// <summary>What this machine shares, shown on the linked Discord profile.</summary>
    public FieldViewModel? RichPresence { get => _richPresence; private set => Set(ref _richPresence, value); }

    /// <summary>Published release, the band's answer and this one being the same object.</summary>
    public UpdateViewModel Updates => _updates;

    /// <summary>
    /// Why this install asks about no release, empty where it asks.
    /// The toggle's own reason, hoisted: the check button is inert for the same fact,
    /// and one line over the section says it once.
    /// </summary>
    public string UpdatesRefusal { get => _updatesRefusal; private set => Set(ref _updatesRefusal, value); }

    public bool UpdatesRefused { get => _updatesRefused; private set => Set(ref _updatesRefused, value); }

    /// <summary>Build the backend answered with, empty until the first read lands.</summary>
    public string Version { get => _version; private set => Set(ref _version, value); }

    /// <summary>
    /// Where this install stands with Discord, in one sentence:
    /// what the last link attempt answered where it failed, and the state the backend read otherwise.
    /// </summary>
    public string DiscordLine { get => _discordLine; private set => Set(ref _discordLine, value); }

    /// <summary>
    /// Links this install to a Discord account, the browser leg running on the press.
    /// Nothing is written here: the secret lands in the stored settings on the backend's side,
    /// and the settings-changed announcement is what moves the draft.
    /// </summary>
    public PendingCommand LinkDiscord => _linkDiscord;

    /// <summary>
    /// Hands the link back, the button standing beside a link alone
    /// (<see cref="IsDiscordLinked"/>).
    /// </summary>
    public PendingCommand UnlinkDiscord => _unlinkDiscord;

    /// <summary>What the link button says, which follows the state it is pressed from.</summary>
    public string LinkLabel { get => _linkLabel; private set => Set(ref _linkLabel, value); }

    /// <summary>
    /// What the unlink button says, and what its press does.
    /// Read off the copy rather than written by a pass: one offer whatever the manager answered.
    /// </summary>
    public string UnlinkLabel => Links.UnlinkLabel;

    public string UnlinkTip => Links.UnlinkTip;

    /// <summary>What the press does, since the label is three words.</summary>
    public string LinkTip { get => _linkTip; private set => Set(ref _linkTip, value); }

    /// <summary>Why the press is refused, empty while it is offered.</summary>
    public string LinkHint { get => _linkHint; private set => Set(ref _linkHint, value); }

    /// <summary>
    /// Whether that sentence reports something broken, which draws it in the failure hue
    /// (<c>docs/design-language.md</c>, "Palette").
    /// A link the manager declines is the one state here that is.
    /// </summary>
    public bool DiscordLineIsFailure
    {
        get => _discordLineIsFailure;
        private set => Set(ref _discordLineIsFailure, value);
    }

    public bool IsDiscordLinked { get => _isDiscordLinked; private set => Set(ref _isDiscordLinked, value); }

    /// <summary>
    /// The linked account's Discord picture, drawn in front of the sentence naming it.
    /// Empty where no account is linked, and until the backend has read one.
    /// </summary>
    public ByteString DiscordAvatar { get => _discordAvatar; private set => Set(ref _discordAvatar, value); }

    /// <summary>
    /// Names the state the press asks for, so pressing twice changes nothing the second time.
    /// A plain open lands on the first section, and on the list where the dialog is compact.
    /// </summary>
    public void Show(bool open, SettingsSection? at = null)
    {
        Section = at ?? SettingsSection.Window;
        Go(_place with { AtList = at is null });
        IsOpen = open;
    }

    /// <summary>
    /// The one render function.
    /// Safe to run twice: an unmoved draft asks for no resolve, and the group's own pass produces fields
    /// that compare equal.
    /// </summary>
    public void Apply()
    {
        _form.Sync();

        var form = _form.Form;
        Group.Apply(GroupOf(form), _session.Words, form?.Settings, _form.IsAnswered, form?.Diagnostics);
        CrashReports = Group.Visible(CrashReportsKey);
        CheckUpdatesOnStart = Group.Visible(CheckUpdatesOnStartKey);
        TrayIcon = Group.Visible(TrayIconKey);
        Autostart = Group.Visible(AutostartKey);
        TestStreams = Group.Visible(TestStreamsKey);
        ShowsDevelopment = TestStreams is not null;
        RichPresence = Group.Visible(RichPresenceKey);

        UpdatesRefusal = CheckUpdatesOnStart?.Reason ?? "";
        UpdatesRefused = UpdatesRefusal.Length > 0;

        Version = _session.Version;

        var discord = _session.Discord;
        IsDiscordLinked = discord?.Linked ?? false;
        DiscordLine = _linkFailed.Length > 0 ? _linkFailed : AppSettingsCopy.DiscordLine(discord);
        DiscordLineIsFailure = _linkFailed.Length > 0 || Links.StateIsFailure(discord);
        ReportLine = _reportFailure.Length > 0
            ? _reportFailure
            : _reportId.Length > 0 ? AppSettingsCopy.ReportSent(_reportId) : "";
        ReportLineIsFailure = _reportFailure.Length > 0;

        DiscordAvatar = discord?.Avatar ?? ByteString.Empty;
        LinkLabel = Links.Label(discord);
        LinkTip = Links.Tip(discord);
        LinkHint = LinkRefusal();
        _linkDiscord.Refresh();

        Assert.That(
            Group.IsResolved || Group.Fields.Count == 0,
            "a dialog with no group draws no controls", Group.Fields.Count);
        Assert.That(
            !Group.IsResolved || CrashReports is not null,
            "a resolved app group always carries the crash-report toggle");
        Assert.That(
            !Group.IsResolved || CheckUpdatesOnStart is not null,
            "a resolved app group always carries the update-on-start toggle");
        Assert.That(
            !Group.IsResolved || TrayIcon is not null,
            "a resolved app group always carries the tray-icon toggle");
        Assert.That(
            !Group.IsResolved || Autostart is not null,
            "a resolved app group always carries the autostart toggle");
        Assert.That(
            UpdatesRefused == (CheckUpdatesOnStart?.HasReason ?? false),
            "the section states the toggle's reason exactly where the toggle has one", UpdatesRefusal);
        Assert.That(
            !UpdatesRefused || !Updates.CanCheck,
            "a section stating why nothing is checked offers no check", UpdatesRefusal);
    }

    /// <summary>
    /// Why no link can be drawn, empty while one can.
    /// The manager runs beside the relay, so an unnamed relay leaves nothing to ask
    /// (<c>docs/discord-mode.md</c>, "The manager").
    /// </summary>
    private string LinkRefusal()
        => _form.Draft?.Relay is { Host.Length: > 0 }
            ? ""
            : "Set the relay address under Relay in Setup first. The Discord manager runs beside the relay.";

    /// <summary>
    /// Runs the link.
    /// Nothing is written back: the secret lands in the stored settings on the backend's side,
    /// and the settings-changed announcement is what moves the draft.
    /// </summary>
    private async Task LinkDiscordAsync(IBackend backend)
    {
        _linkFailed = "";
        Apply();

        var relay = _form.Draft?.Relay;
        if (relay is null)
        {
            return;
        }

        try
        {
            await backend.LinkDiscordAsync(relay).ConfigureAwait(false);
            _dispatch(Apply);
        }
        catch (BackendUnavailableException e)
        {
            _dispatch(() => LinkFailed(e.Message));
        }
        catch (OperationCanceledException)
        {
            _dispatch(() => LinkFailed(""));
        }
    }

    /// <summary>
    /// Hands the link back.
    /// Nothing is written here either: the secret leaves the stored settings on the backend's side,
    /// and the settings-changed announcement is what moves the draft.
    /// </summary>
    private async Task UnlinkDiscordAsync(IBackend backend)
    {
        _linkFailed = "";
        Apply();

        try
        {
            await backend.UnlinkDiscordAsync().ConfigureAwait(false);
            _dispatch(Apply);
        }
        catch (BackendUnavailableException e)
        {
            _dispatch(() => LinkFailed(e.Message));
        }
        catch (OperationCanceledException)
        {
            _dispatch(() => LinkFailed(""));
        }
    }

    /// <summary>Takes a link that did not land, on the UI loop.</summary>
    private void LinkFailed(string reason)
    {
        _linkFailed = reason;
        Apply();
    }

    /// <summary>
    /// Sends one report and keeps what came back.
    /// The line is cleared as the press starts, so the name under the button is the name of the send
    /// that is running rather than the one before it.
    /// A cancelled call clears it and says nothing, nothing having been refused.
    /// </summary>
    private async Task SendReportAsync(IBackend backend)
    {
        ReportLanded("", "");

        try
        {
            var id = await backend.SendReportAsync().ConfigureAwait(false);
            _dispatch(() => ReportLanded(id, ""));
        }
        catch (BackendUnavailableException e)
        {
            _dispatch(() => ReportLanded("", e.Message));
        }
        catch (OperationCanceledException)
        {
            _dispatch(() => ReportLanded("", ""));
        }
    }

    /// <summary>Takes what a report press answered, on the UI loop.</summary>
    private void ReportLanded(string id, string failure)
    {
        _reportId = id;
        _reportFailure = failure;
        Apply();
    }

    /// <summary>
    /// The reset beside the heading, which an applied group carries: its fields are already what this machine
    /// is, so a reader has no other way back to what a fresh installation holds
    /// (<c>docs/settings-editing.md</c>, "Who can be put back").
    ///
    /// Composed per pass around the held command, so two passes produce actions that compare equal.
    /// </summary>
    private GroupAction ResetOf(Api.V1.FieldGroup group) => new(
        AppSettingsCopy.ResetLabel, AppSettingsCopy.ResetHelp, ResetCommandOf(group.Key));

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
    /// App group of the resolved form, null where the form carries none.
    /// Found by the key deciding which surface draws a group, so this dialog and the wizard's filter read
    /// one table (<see cref="GroupPlacement"/>).
    /// </summary>
    private static Api.V1.FieldGroup? GroupOf(Api.V1.Form? form)
    {
        if (form is null)
        {
            return null;
        }

        foreach (var group in form.Groups)
        {
            if (GroupPlacement.InAppSettings(group.Key))
            {
                return group;
            }
        }

        return null;
    }
}
