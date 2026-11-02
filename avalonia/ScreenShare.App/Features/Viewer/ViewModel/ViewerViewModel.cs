using System.Collections.ObjectModel;
using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Shell.Model;
using ScreenShare.App.Features.Shell.ViewModel;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Viewer.Model;
using ScreenShare.App.Features.Viewer.Tile.Model;
using ScreenShare.App.Features.Viewer.Tile.ViewModel;
using ScreenShare.App.Features.Viewer.WatchSettings.ViewModel;
using ScreenShare.App.Mvvm;
using TablerIcons;

namespace ScreenShare.App.Features.Viewer.ViewModel;

/// <summary>
/// Every stream the relay carries, played on a stage with a strip under it.
/// Each ready stream plays with no press, and one the reader stopped waits in the strip until a press.
///
/// <b>Holds no snapshot of its own.</b>
/// <see cref="Backend.Session"/> owns the running state and <see cref="Apply"/> reads it through on every pass.
///
/// <b>The arrangement is this shell's alone.</b>
/// Which streams are decoded is the backend's list; which are drawn, in what order and in which window crosses
/// no message, the backend describing decodes and a decode not being a tile (<c>docs/ipc-api.md</c>).
/// The tile set, the stops, the focus, the pop-outs and the fullscreen states are that arrangement, owned here.
///
/// Legs come from the backend as the options of the form's watch-leg field,
/// so this module holds no list of protocols.
/// Whether a leg carries a given stream is settled as the viewer opens (<see cref="WatchLegViewModel"/>).
///
/// The settings behind those legs are edited in the panel beside the stage.
/// They govern how this machine receives and say nothing about what it sends, hence their placement here
/// (<c>Features/Fields/Model/GroupPlacement.cs</c>).
/// </summary>
public sealed class ViewerViewModel : Observable
{
    private readonly IBackend _backend;

    /// <summary>
    /// Draft and the form it resolves to, owned by the window.
    /// Read through on every pass and never copied,
    /// so a leg changed in the wizard reaches the rows and the next decode with nothing here being told
    /// (<c>docs/development-principles.md</c>, "A reader reads through").
    /// </summary>
    private readonly FormSession _form;

    private readonly Session _session;
    private readonly Action<Action> _dispatch;
    private readonly Dictionary<string, StreamRowViewModel> _rows = [];

    /// <summary>
    /// Tiles on screen, by stream name.
    /// Written rather than read through, the one departure here:
    /// the contract describes decodes and never a window, so there is no tile list to read back
    /// (<c>docs/ipc-api.md</c>).
    /// </summary>
    private readonly Dictionary<string, TileViewModel> _tiles = [];

    /// <summary>
    /// Streams drawn in windows of their own, by name.
    ///
    /// <b>Names and not windows.</b>
    /// Keeps a toolkit type out of a view model (<c>avalonia/README.md</c>).
    ///
    /// A popped stream keeps its tile in the grid, drawn as a plate at its own shape,
    /// so nothing reflows when a stream pops out or comes back.
    /// </summary>
    private readonly HashSet<string> _popped = [];

    /// <summary>
    /// Popped-out streams whose own window should be fullscreen.
    /// A second set rather than a flag on the first: which windows exist, and which fill their screen,
    /// are different questions.
    /// Several fill theirs at once, each on the monitor its window sits on,
    /// which one app-wide fullscreen could not express.
    /// </summary>
    private readonly HashSet<string> _poppedFullscreen = [];

    /// <summary>
    /// Whether the window is on screen (<see cref="SetWindowShown"/>).
    /// True until a window says otherwise, so a viewer with no window around it draws.
    /// </summary>
    private bool _shown = true;

    /// <summary>
    /// Streams the hidden window was watching, in tile order, empty while it is on screen.
    /// The one fact a hide keeps: the tiles go with their decodes,
    /// and the names bring both back with the window.
    /// </summary>
    private readonly List<string> _parked = [];

    /// <summary>
    /// Streams the reader stopped, which wait in the strip until a press plays them.
    /// A name goes once the relay stops listing its stream: a share that ends and starts again plays like any other.
    /// </summary>
    private readonly HashSet<string> _stopped = [];

    /// <summary>Streams whose start is out, so a pass before the answer asks for none a second time.</summary>
    private readonly HashSet<string> _opening = [];

    /// <summary>Closes the last hide asked for, complete where it asked for none. A quit waits on it.</summary>
    private Task _parting = Task.CompletedTask;

    /// <param name="dispatch">
    /// Hands work to the UI loop.
    /// An effect answers on whichever thread the transport completed on,
    /// and everything written here is read by a binding that tolerates one thread only.
    /// </param>
    public ViewerViewModel(IBackend backend, FormSession form, Session session, Action<Action> dispatch)
    {
        Assert.NotNull(backend, "a viewer asks the backend to open and close viewers");
        Assert.NotNull(form, "a viewer draws the settings that govern how it receives");
        Assert.NotNull(session, "a viewer renders the session's running state");
        Assert.NotNull(dispatch, "a viewer needs a UI loop to marshal an answer back to");

        _backend = backend;
        _form = form;
        _session = session;
        _dispatch = dispatch;

        Streams = [];
        Tiles = [];

        PickDiscord = new DelegateCommand(() => PickJoin(JoinRoute.Discord));
        PickKey = new DelegateCommand(() => PickJoin(JoinRoute.Key));
        Join = new DelegateCommand(() => JoinRequested?.Invoke(PickedJoin));

        ToggleGrid = new DelegateCommand(() =>
        {
            _gridAsked = !IsGrid;
            Apply();
        });
        Switch = new GridSwitch(ToggleGrid);
        Strip = [];
        HintsBelow = [];

        LeaveFullscreen = new DelegateCommand(() =>
        {
            Fullscreen = "";
            Apply();
        });

        // Whether the panel is open is this screen's state, so the panel is handed only a way to say it is done.
        // A panel holding the flag would be the arrangement written in two places.
        Watch = new WatchSettingsViewModel(form, session, dispatch, CloseWatchSettings);

        WatchPanel = new OverPanelViewModel("How streams arrive", "Close how streams arrive");

        // News that the draft or the form behind it moved: a row's legs and a tile's leg are both read off it.
        // Raised on the UI loop by the form session, so nothing to marshal here.
        _form.Changed += Apply;

        Apply();
    }

    // --- Outputs --------------------------------------------------------------------

    private string _notice = "";
    private bool _hasNotice;
    private bool _isReading;
    private string _gridEmptyLine = "";
    private bool _showsGridEmpty;
    private bool _noticeIsFailure;
    private bool _isDialling;
    private string _refusal = "";
    private bool _hasRefusal;

    /// <summary>One row per path the relay carries, in the order it listed them.</summary>
    public ObservableCollection<StreamRowViewModel> Streams { get; }

    /// <summary>
    /// Tiles on screen, in the order they were added.
    /// A stream leaves when the reader stops it and stays while the relay drops it,
    /// so a stream that dropped out and came back keeps its tile.
    /// </summary>
    public ObservableCollection<TileViewModel> Tiles { get; }

    /// <summary>
    /// What the tile panel arranges, in order: the playing tiles, the streams waiting for a press,
    /// a line per member sharing nothing, and the Grid switch while a stage stands over the strip.
    /// Converged in place, so a playing tile keeps its control and its frame subscription across passes.
    /// </summary>
    public ObservableCollection<IStripItem> Strip { get; }

    /// <summary>The members sharing nothing, listed under a stacked strip, empty elsewhere.</summary>
    public ObservableCollection<MemberHint> HintsBelow { get; }

    private bool _hasHintsBelow;

    public bool HasHintsBelow { get => _hasHintsBelow; private set => Set(ref _hasHintsBelow, value); }

    /// <summary>Strip.s last item while a stage stands over it.</summary>
    public GridSwitch Switch { get; }

    private LayoutMode _mode;
    private string _focused = "";
    private bool _isGrid;
    private bool _showsGridToggle;

    /// <summary>Whether the reader asked for equal cells over the stage.</summary>
    private bool _gridAsked;

    /// <summary>
    /// Stream the reader last put on the stage, empty before any pick.
    /// Kept while its stream is gone, so one that drops out and comes back takes the stage again.
    /// </summary>
    private string _staged = "";
    private string _fullscreen = "";

    /// <summary>
    /// How the tiles are arranged: the stage with the strip under it wherever the strip holds anything and
    /// the reader has not asked for the grid, stacked on a narrow window.
    /// </summary>
    public LayoutMode Mode { get => _mode; private set => Set(ref _mode, value); }

    /// <summary>
    /// Stream on the stage, empty in the grid.
    /// The reader's pick where its tile is on screen, the first tile otherwise.
    /// </summary>
    public string Focused { get => _focused; private set => Set(ref _focused, value); }

    /// <summary>Whether the tiles sit in equal cells, the state the Grid toggle shows.</summary>
    public bool IsGrid { get => _isGrid; private set => Set(ref _isGrid, value); }

    /// <summary>Whether there is a choice to make: one stream fills the window either way.</summary>
    public bool ShowsGridToggle { get => _showsGridToggle; private set => Set(ref _showsGridToggle, value); }

    /// <summary>Swaps between the stage and equal cells.</summary>
    public DelegateCommand ToggleGrid { get; }

    /// <summary>
    /// Stream the main window is drawing fullscreen, empty when it is not.
    ///
    /// <b>Fullscreen is a property of a window, not of the app.</b>
    /// The main window's here, each popped-out window carrying its own, so several windows fill several monitors
    /// at once.
    /// Not a member of <see cref="LayoutMode"/>: a mode says how tiles sit relative to each other,
    /// this says which window one of them fills.
    ///
    /// The stage and the strip go, the shell takes its own bands off the window
    /// (<c>Features/Shell/ViewModel/ShellViewModel.cs</c>),
    /// and the picture is letterboxed on black at its stream's shape rather than stretched to the monitor's.
    /// </summary>
    public string Fullscreen { get => _fullscreen; private set => Set(ref _fullscreen, value); }

    /// <summary>
    /// Streams that should be drawn in windows of their own, as of this pass.
    /// A view opens and closes windows to match, an idempotent apply: an unchanged set opens and closes nothing.
    /// </summary>
    public IReadOnlyCollection<string> PoppedOut => _popped;

    /// <summary>Which of those windows should be filling their screen.</summary>
    public IReadOnlyCollection<string> PoppedFullscreen => _poppedFullscreen;

    private bool _hasFullscreen;
    private TileViewModel? _fullscreenTile;

    /// <summary>
    /// Width the window last stated, infinite until it states one (<see cref="SetWindowWidth"/>).
    /// </summary>
    private double _window = double.PositiveInfinity;

    /// <summary>Whether a tile fills this window, taking the stage and the strip off it.</summary>
    public bool HasFullscreen { get => _hasFullscreen; private set => Set(ref _hasFullscreen, value); }

    public TileViewModel? FullscreenTile { get => _fullscreenTile; private set => Set(ref _fullscreenTile, value); }

    /// <summary>
    /// Gives the main window back to its grid, whether or not a stream was filling it.
    /// The window's key rather than a tile's: a filled window draws no strip, no menu and no band,
    /// so the keyboard is what a reader can still reach (<c>Features/Shell/View/ShellWindow.axaml</c>).
    /// </summary>
    public DelegateCommand LeaveFullscreen { get; }

    /// <summary>
    /// How this machine receives: the legs, the jitter buffers and the render chain.
    /// One group of the same resolved form the setup wizard draws its steps from, on the screen its settings govern.
    /// </summary>
    public WatchSettingsViewModel Watch { get; }

    /// <summary>Whether the watch settings are open over the stage, and how they stand at this width.</summary>
    public OverPanelViewModel WatchPanel { get; }

    /// <summary>
    /// Shuts the settings panel, whether or not it was open.
    /// Names the state rather than a transition, so the panel's close button and its commit can both run it:
    /// a commit that closed by toggling would reopen a panel the reader had already dismissed.
    /// </summary>
    private void CloseWatchSettings()
    {
        WatchPanel.Close();
        Apply();
    }

    /// <summary>
    /// States the width the window has, which decides whether the strip stacks and where the panel stands.
    /// Idempotent: the same width twice moves nothing.
    /// </summary>
    public void SetWindowWidth(double width)
    {
        Assert.That(width > 0, "a window states a width it has", width);

        _window = width;
        WatchPanel.SetWindowWidth(width);
        Apply();
    }

    /// <summary>
    /// States whether the window is on screen.
    /// A decode runs while a window draws its tile, and a window closed to the tray draws nothing:
    /// every tile in its grid leaves and its decode stops, and a stream in a window of its own keeps both
    /// (<c>docs/viewer-architecture.md</c>, "A decode runs while a window draws it").
    /// The streams are kept, so the window comes back watching them,
    /// each through the start a press runs and on the leg the settings name then.
    /// Idempotent: the same answer twice moves nothing.
    /// </summary>
    public void SetWindowShown(bool shown)
    {
        if (_shown == shown)
        {
            return;
        }

        _shown = shown;
        if (shown)
        {
            Unpark();
        }
        else
        {
            _parting = Park();
        }

        Apply();
    }

    /// <summary>
    /// Hides ahead of a quit, answering once the backend has closed every decode the hide took down.
    /// A stop still on its way as the process exits is a decode left running on a backend that outlives
    /// the shell.
    /// </summary>
    public Task PartAsync()
    {
        SetWindowShown(false);
        return _parting;
    }

    /// <summary>For a view that has to hand a tile to a window it is opening.</summary>
    public TileViewModel? TileOf(string stream) => _tiles.GetValueOrDefault(stream);

    /// <summary>
    /// Raised after a pass in which the windows a view should be showing changed.
    /// The one thing a view is told imperatively, nothing binding a window into existence.
    /// </summary>
    public event Action? WindowsChanged;

    /// <summary>What keeps the relay from listing streams: an absent backend, an unread relay, an unreachable one.</summary>
    public string Notice { get => _notice; private set => Set(ref _notice, value); }

    public bool HasNotice { get => _hasNotice; private set => Set(ref _hasNotice, value); }

    private bool _noticeCentred;
    private bool _noticeBelow;

    /// <summary>Whether the notice stands on the empty stage.</summary>
    public bool NoticeCentred { get => _noticeCentred; private set => Set(ref _noticeCentred, value); }

    /// <summary>Whether the notice stands under a strip that kept its items.</summary>
    public bool NoticeBelow { get => _noticeBelow; private set => Set(ref _noticeBelow, value); }

    /// <summary>What the empty stage says, empty while the strip holds anything or another notice speaks (<see cref="Model.GridEmpty"/>).</summary>
    public string GridEmptyLine { get => _gridEmptyLine; private set => Set(ref _gridEmptyLine, value); }

    public bool ShowsGridEmpty { get => _showsGridEmpty; private set => Set(ref _showsGridEmpty, value); }

    private bool _showsJoinCard;

    /// <summary>Whether the empty grid asks how to join a group (<see cref="GridEmpty.AsksToJoin"/>).</summary>
    public bool ShowsJoinCard { get => _showsJoinCard; private set => Set(ref _showsJoinCard, value); }

    /// <summary>
    /// Raised by the join card's press, for the way in its rows picked.
    /// The press leads to the control that sets it, which the shell reaches.
    /// The card writes nothing, the setting having one editor.
    /// </summary>
    public event Action<JoinRoute>? JoinRequested;

    private JoinRoute _pickedJoin = JoinRoute.Discord;
    private string _joinLabel = "";
    private Icons _joinGlyph;

    /// <summary>Way in the card's rows picked. Discord until a row says otherwise, the recommended way.</summary>
    public JoinRoute PickedJoin { get => _pickedJoin; private set => Set(ref _pickedJoin, value); }

    public bool PicksDiscord => PickedJoin == JoinRoute.Discord;

    public bool PicksKey => PickedJoin == JoinRoute.Key;

    public string JoinLabel { get => _joinLabel; private set => Set(ref _joinLabel, value); }

    public Icons JoinGlyph { get => _joinGlyph; private set => Set(ref _joinGlyph, value); }

    public DelegateCommand PickDiscord { get; }

    public DelegateCommand PickKey { get; }

    public DelegateCommand Join { get; }

    private void PickJoin(JoinRoute route)
    {
        PickedJoin = route;
        OnPropertyChanged(nameof(PicksDiscord));
        OnPropertyChanged(nameof(PicksKey));
        Apply();
    }

    /// <summary>
    /// Whether the notice reports a failure rather than a state of the relay.
    /// True for the backend's own sentence, which is the one thing here that is broken:
    /// a relay carrying nothing answered (<c>docs/design-language.md</c>, "Palette").
    /// </summary>
    public bool NoticeIsFailure { get => _noticeIsFailure; private set => Set(ref _noticeIsFailure, value); }

    /// <summary>
    /// Whether the window is still dialling behind the notice.
    /// False where the notice is not about the backend.
    /// The window opens on this screen, so a shell launched before its backend draws it,
    /// and a sentence that never moves makes one still dialling look stuck.
    /// </summary>
    public bool IsDialling { get => _isDialling; private set => Set(ref _isDialling, value); }

    /// <summary>
    /// Whether the first answer is still out, which is the state this screen opens in.
    /// The sentence beside it already says what is being read, so the arc carries no tip of its own.
    /// </summary>
    public bool IsReading { get => _isReading; private set => Set(ref _isReading, value); }

    /// <summary>Backend's own sentence when it refused to open or close something, empty otherwise.</summary>
    public string Refusal { get => _refusal; private set => Set(ref _refusal, value); }

    public bool HasRefusal { get => _hasRefusal; private set => Set(ref _hasRefusal, value); }

    private string _groupRefusal = "";
    private bool _hasGroupRefusal;

    /// <summary>Why the group service or this machine.s own settings refused its presence, empty otherwise.</summary>
    public string GroupRefusal { get => _groupRefusal; private set => Set(ref _groupRefusal, value); }

    public bool HasGroupRefusal { get => _hasGroupRefusal; private set => Set(ref _hasGroupRefusal, value); }

    // --- Lifecycle ------------------------------------------------------------------

    /// <summary>
    /// The one render function.
    /// Reads the session and the form through, keeping no copy of either.
    /// Safe to run twice: a row is reused by stream name and renders itself idempotently,
    /// so an unchanged snapshot fires no binding.
    /// </summary>
    public void Apply()
    {
        // Names the state wanted, a form resolved from the draft as it stands,
        // and the sync decides whether a round trip is owed (docs/development-principles.md, "Idempotency").
        _form.Sync();

        var legs = LegsOf(_session.PlayerLegs);
        var browserLegs = LegsOf(_session.BrowserLegs);
        var relay = _session.Relay;
        var rows = Rows(relay, _session.Watching);

        foreach (var row in rows)
        {
            Of(row.Name).Apply(row, legs, browserLegs, _tiles.ContainsKey(row.Name));
        }

        Reconcile.Onto(Streams, rows.Select(row => Of(row.Name)).ToList());

        // Read off a relay that answered, an unread or unreachable one listing nothing about any stream.
        if (relay is { Reachable: true })
        {
            _stopped.RemoveWhere(stream => rows.All(row => row.Name != stream));
        }

        // A window whose stream left the grid has nothing behind it,
        // and a fullscreen state with no window has nothing to answer for.
        _popped.RemoveWhere(stream => !_tiles.ContainsKey(stream));
        _poppedFullscreen.RemoveWhere(stream => !_popped.Contains(stream));
        if (Fullscreen.Length > 0 && !_tiles.ContainsKey(Fullscreen))
        {
            Fullscreen = "";
        }

        // This window's fullscreen names a stream of this window's own grid,
        // so a stream that left for a window of its own gives this one back.
        // Fullscreen does not travel with the stream:
        // one arriving already filling a screen is a state the reader did not ask for.
        if (_popped.Contains(Fullscreen))
        {
            Fullscreen = "";
        }

        var waiting = rows
            .Where(row => !_tiles.ContainsKey(row.Name) && !_opening.Contains(row.Name))
            .Select(row => (IStripItem)Of(row.Name))
            .ToList();
        var hints = HintsOf(_session.Members);
        var narrow = _window < NavWidth.LabelsMinWidth;

        // A stage with an empty strip under it would shrink the one picture for nothing.
        ShowsGridToggle = Tiles.Count >= 2;
        var beside = waiting.Count + (narrow ? 0 : hints.Count);
        Mode = Tiles.Count == 0 || (Tiles.Count == 1 && beside == 0) || (ShowsGridToggle && _gridAsked)
            ? LayoutMode.Grid
            : narrow ? LayoutMode.Stack : LayoutMode.Focus;
        IsGrid = Mode == LayoutMode.Grid;
        Focused = Mode == LayoutMode.Grid ? ""
            : _tiles.ContainsKey(_staged) ? _staged
            : Tiles[0].Name;

        // Rendered from the backend's decode list, joined on the pair the contract keys a decode by.
        // A tile whose decode is not in it draws its own reason rather than disappearing:
        // the reader put it there.
        foreach (var tile in _tiles.Values)
        {
            tile.Apply(TilePipeline.Of(DecodeOf(tile)), _session.StatsOf(tile.Name, tile.Transport));
            tile.IsFocused = tile.Name == Focused;
            tile.IsPoppedOut = _popped.Contains(tile.Name);

            // Which window a stream is drawn in decides which fullscreen state answers for it.
            // Derived here rather than kept on the tile,
            // so the flag the menu ticks and the state the windows obey cannot disagree.
            tile.IsFullscreen = _popped.Contains(tile.Name)
                ? _poppedFullscreen.Contains(tile.Name)
                : Fullscreen == tile.Name;
        }

        var stacked = Mode == LayoutMode.Stack;
        Reconcile.InPlace(Strip, [
            .. Tiles,
            .. waiting,
            .. stacked ? [] : hints,
            .. ShowsGridToggle && !IsGrid ? [Switch] : Array.Empty<IStripItem>(),
        ]);
        Reconcile.Onto(HintsBelow, stacked ? hints : []);
        HasHintsBelow = HintsBelow.Count > 0;

        // An absent backend is why there is no relay reading at all,
        // so it answers before the relay's own states and is the one notice the dialling belongs under.
        var absent = relay is null && _session.Unavailable.Length > 0;

        Notice = absent ? _session.Unavailable : NoticeFor(relay);
        HasNotice = Notice.Length > 0;
        NoticeCentred = HasNotice && Strip.Count == 0;
        NoticeBelow = HasNotice && Strip.Count > 0;
        NoticeIsFailure = absent;

        // Read off the sentence's own verdict: an attempt is coming for the notice that is a failure,
        // and for no other.
        IsDialling = NoticeIsFailure;

        // The other half of the same arc, and the one a launch shows: the sentence above says the relay is being
        // read, and this is what makes it a wait rather than a screen that stopped.
        IsReading = _session.IsReading;

        HasRefusal = Refusal.Length > 0;
        GroupRefusal = Statements.Of(_session.Members?.Refusal);
        HasGroupRefusal = GroupRefusal.Length > 0;

        FullscreenTile = Fullscreen.Length > 0 ? _tiles.GetValueOrDefault(Fullscreen) : null;
        HasFullscreen = FullscreenTile is not null;

        // The way into a group, which the card and the stage each name in their own sentence.
        var discordMode = _form.Stored?.Relay?.GroupSource == "discord";

        GridEmptyLine = GridEmpty.For(
            _session.Members, _session.Discord, discordMode, Strip.Count, relay?.Reachable == true);
        ShowsGridEmpty = GridEmptyLine.Length > 0;
        ShowsJoinCard = GridEmpty.AsksToJoin(_session.Members, _session.Discord, discordMode, Strip.Count);
        var press = JoinPresses.Of(PickedJoin, _session.Discord is { Linked: true });
        JoinLabel = press.Label;
        JoinGlyph = press.Glyph;

        // Draws from the same draft on every pass,
        // so a vocabulary arriving with the catalog reaches the panel's entries without a notification of its own.
        Watch.Apply();

        WindowsChanged?.Invoke();

        Assert.That(Streams.Count == rows.Count, "a row per stream on the relay", Streams.Count, rows.Count);
        Assert.That(HasFullscreen == (FullscreenTile is not null), "a fullscreen tile and the state drawing it agree", HasFullscreen);
        Assert.That(IsGrid == (Focused.Length == 0), "a stage and the mode that draws it agree", Mode, Focused);
        Assert.That(Focused.Length == 0 || _tiles.ContainsKey(Focused), "a focused stream is one of the tiles", Focused);
        Assert.That(Fullscreen.Length == 0 || _tiles.ContainsKey(Fullscreen), "a fullscreen stream is one of the tiles", Fullscreen);
        Assert.That(!_popped.Contains(Fullscreen), "the stream filling this window is one of its own grid's", Fullscreen);
        Assert.That(_tiles.Count == Tiles.Count, "one tile per stream in the grid", _tiles.Count, Tiles.Count);
        Assert.That(Strip.OfType<TileViewModel>().Count() == Tiles.Count, "every tile stands in the strip", Strip.Count, Tiles.Count);
        Assert.That(_tiles.Keys.All(Drawn), "every tile has a window drawing it", _tiles.Count, _shown);
        Assert.That(!_shown || _parked.Count == 0, "a window on screen keeps no streams for its return", _parked.Count);
        Assert.That(!_parked.Any(_tiles.ContainsKey), "a kept stream has no tile", _parked.Count);
        Assert.That(!_stopped.Any(_tiles.ContainsKey), "a stopped stream has no tile", _stopped.Count);
        Assert.That(HasNotice == (Notice.Length > 0), "the notice and its text agree", HasNotice);
        Assert.That(!(NoticeCentred && NoticeBelow), "the notice stands in one place", Strip.Count);
        Assert.That(!NoticeIsFailure || HasNotice, "a failure is marked on the sentence stating it", NoticeIsFailure, HasNotice);
        Assert.That(!IsDialling || HasNotice, "the wait appears under the notice it belongs to", IsDialling, HasNotice);
        Assert.That(HasRefusal == (Refusal.Length > 0), "a refusal and its sentence agree", HasRefusal);

        // Last, after every output holds: a start answering at once renders again from here.
        foreach (var stream in Unplayed(rows))
        {
            _ = StartAsync(stream);
        }
    }

    /// <summary>
    /// Ready streams nobody stopped and nothing is starting, in relay order.
    /// None while no window draws a tile, and none before the settings name the leg a tile opens on.
    /// This machine's own share waits for a press: its picture is the one the reader is sending.
    /// </summary>
    private List<string> Unplayed(IReadOnlyList<StreamRow> rows)
    {
        if (!_shown || TileLeg.Of(_form.Stored).Length == 0)
        {
            return [];
        }

        var own = _session.Publish?.Live?.StreamName ?? "";
        return rows
            .Where(row => row.IsReady && row.Name != own)
            .Select(row => row.Name)
            .Where(stream => !_tiles.ContainsKey(stream) && !_opening.Contains(stream) && !_stopped.Contains(stream))
            .ToList();
    }

    /// <summary>A line per other member of the group publishing nothing, in the group's order.</summary>
    private static List<MemberHint> HintsOf(MembersState? group)
        => group is { Joined: true }
            ? group.Members
                .Where(member => !member.Self && !member.Publishing)
                .Select(member => new MemberHint(
                    Cards.NotSharing(member.DisplayName.Length > 0 ? member.DisplayName : member.MemberId)))
                .ToList()
            : [];

    /// <summary>
    /// One row per relay path, carrying the legs this machine already has open on it.
    /// Joined on the name the backend uses on both sides,
    /// so nothing here has to know what a transport or a format is.
    /// </summary>
    private static IReadOnlyList<StreamRow> Rows(RelayStatus? relay, IReadOnlyList<StreamRef> watching)
    {
        if (relay is null || !relay.Reachable)
        {
            return [];
        }

        var rows = new List<StreamRow>(relay.Paths.Count);
        foreach (var path in relay.Paths)
        {
            var open = new List<string>();
            foreach (var key in watching)
            {
                if (key.StreamName == path.Name)
                {
                    open.Add(key.Transport);
                }
            }

            rows.Add(new StreamRow
            {
                Name = path.Name,
                IsReady = path.Ready,
                Tracks = path.Tracks,
                Format = path.Format,
                Readers = path.Readers,
                InMbps = path.InMbps,
                WatchedOn = open,
            });
        }

        return rows;
    }

    /// <summary>
    /// Why the relay lists nothing, for every reason but an absent backend, which the render pass answers first.
    /// An unreachable relay says why in its own words.
    /// Empty for a relay that answered: an idle group is the empty stage's line (<see cref="GridEmpty"/>).
    /// </summary>
    private static string NoticeFor(RelayStatus? relay)
    {
        if (relay is null)
        {
            return "Reading what the relay is carrying.";
        }

        if (!relay.Reachable)
        {
            return relay.Error.Length > 0 ? relay.Error : "The relay could not be reached.";
        }

        return "";
    }

    private StreamRowViewModel Of(string name)
    {
        if (_rows.TryGetValue(name, out var row))
        {
            return row;
        }

        row = new StreamRowViewModel(name, WatchAsync, TileAsync, BrowseAsync, _dispatch);
        _rows[name] = row;
        return row;
    }

    /// <summary>
    /// Backend's state for one tile's decode, null while nothing is decoding that pair.
    /// Joined on the stream name and the leg together,
    /// the relay re-serving each stream on all its listeners so the name alone is no identity.
    /// </summary>
    private ReceiveStream? DecodeOf(TileViewModel tile)
    {
        foreach (var decode in _session.Receiving)
        {
            if (decode.Stream.StreamName == tile.Name && decode.Stream.Transport == tile.Transport)
            {
                return decode;
            }
        }

        return null;
    }

    // --- The effects ----------------------------------------------------------------

    /// <summary>
    /// Legs of one roster, named for this screen.
    /// The list is the catalog's and the words this side's:
    /// which legs a receiver opens on is a fact, what a protocol is called on a row is a decision about the screen.
    ///
    /// One answer serves every row, and no entry carries a verdict.
    /// A roster names the legs its own receiver reaches and answers nothing about a stream,
    /// so whether a leg carries this one is settled as the viewer opens (<see cref="WatchLegViewModel"/>).
    ///
    /// Empty until the first catalog read lands, a menu with nothing under it rather than one guessing at protocols.
    /// </summary>
    private static IReadOnlyList<WatchLeg> LegsOf(IReadOnlyList<string> legs)
        => legs.Select(leg => new WatchLeg(leg, Copy.Words.Transport(leg))).ToList();

    /// <summary>A press on one stream: plays one waiting in the strip, stops one playing.</summary>
    private Task TileAsync(string stream, bool tiled) => tiled ? StopAsync(stream) : PlayAsync(stream);

    /// <summary>Plays a stream the reader asked for, lifting a stop they made earlier.</summary>
    private Task PlayAsync(string stream)
    {
        _stopped.Remove(stream);
        return StartAsync(stream);
    }

    /// <summary>
    /// Takes one stream off the stage and keeps it in the strip until a press.
    /// The tile goes before the decode closes:
    /// a subscription outliving its decode is a window holding handles to memory the backend has freed.
    /// The stop names the leg the tile opened on, possibly an older setting,
    /// a decode being keyed by the stream and the leg together.
    /// Idempotent: a stream already stopped asks the backend for nothing.
    /// </summary>
    private async Task StopAsync(string stream)
    {
        // Read before the tile goes, the tile being where the answer is.
        var opened = _tiles.TryGetValue(stream, out var tile) ? tile.Transport : "";
        _stopped.Add(stream);
        Drop(stream);
        _parked.Remove(stream);
        Apply();

        if (opened.Length > 0)
        {
            await CloseAsync(stream, opened).ConfigureAwait(false);
        }
    }

    /// <summary>
    /// Opens one stream's decode, then its tile, there being no frames to subscribe to until something decodes.
    /// Idempotent: a stream playing or starting asks for nothing.
    ///
    /// <b>The leg is the stored one.</b>
    /// It is the only knob of the watch group this call names,
    /// the backend reading the render chain and the jitter buffers out of its own settings as it builds the decode.
    /// Opening on the draft would run an unkept leg against kept buffers
    /// (<c>Features/Viewer/Tile/Model/TileLeg.cs</c>).
    ///
    /// A refused start leaves the stream stopped, so the next pass asks nothing until a press,
    /// and the refusal line says why.
    /// </summary>
    private async Task StartAsync(string stream)
    {
        if (_tiles.ContainsKey(stream) || !_opening.Add(stream))
        {
            return;
        }

        var leg = TileLeg.Of(_form.Stored);
        if (leg.Length == 0)
        {
            _opening.Remove(stream);
            _stopped.Add(stream);
            Refused("The settings name no protocol for a tile to receive on. Pick one under Watching.");
            return;
        }

        try
        {
            await _backend.StartReceiveAsync(stream, leg).ConfigureAwait(false);

            // One step on the loop, so the tile lands before a pass the cleared refusal would run starts another.
            _dispatch(() =>
            {
                _opening.Remove(stream);
                Refusal = "";
                Add(stream, leg);
            });
        }
        catch (BackendUnavailableException e)
        {
            // Shown as it arrived: a refusal names the format and the protocols that would have carried it.
            Refused(e.Message);
            _dispatch(() =>
            {
                _opening.Remove(stream);
                _stopped.Add(stream);
                Apply();
            });
        }
        catch (OperationCanceledException)
        {
            _dispatch(() => _opening.Remove(stream));
        }
    }

    /// <summary>Closes one decode and records what the backend said, an empty answer clearing the last refusal.</summary>
    private async Task CloseAsync(string stream, string leg)
    {
        try
        {
            await _backend.StopReceiveAsync(stream, leg).ConfigureAwait(false);
            Refused("");
        }
        catch (BackendUnavailableException e)
        {
            Refused(e.Message);
        }
        catch (OperationCanceledException)
        {
        }
    }

    /// <summary>
    /// Whether a window draws one stream's tile: the main window while it is on screen, or a window of the stream's
    /// own.
    /// </summary>
    private bool Drawn(string stream) => _shown || _popped.Contains(stream);

    /// <summary>
    /// Takes every tile no window draws off the grid, closes its decode and keeps its stream for the window's
    /// return.
    /// Runs after each write that can leave one: the window hiding, and a pop-out closing into a hidden grid.
    /// The tiles go before any stop is asked for, the order <see cref="StopAsync"/> states.
    /// Answers once every stop has.
    /// </summary>
    private Task Park()
    {
        var undrawn = _tiles.Values.Where(tile => !Drawn(tile.Name)).ToList();
        foreach (var tile in undrawn)
        {
            Drop(tile.Name);
            _parked.Add(tile.Name);
        }

        return Task.WhenAll(undrawn.Select(tile => CloseAsync(tile.Name, tile.Transport)));
    }

    /// <summary>
    /// Plays one stream, for a link the desktop handed this window
    /// (<c>Features/Shell/ViewModel/ShellViewModel.cs</c>).
    ///
    /// The press's own path, so a linked stream plays on the leg every other tile uses, a stop on it is lifted,
    /// and a stream already playing stays as it is.
    /// A refusal lands where a press's would, the screen being where a reader is looking.
    /// </summary>
    public Task OpenAsync(string stream)
    {
        Assert.That(stream.Length > 0, "a link opens a stream it names");

        return PlayAsync(stream);
    }

    /// <summary>Says why a link could not be opened, in the line a press's refusal lands in.</summary>
    public void RefuseLink(string reason) => Refused(reason);

    /// <summary>
    /// Starts every kept stream again and forgets the list: what the answers add back is the grid.
    /// </summary>
    private void Unpark()
    {
        var parked = _parked.ToList();
        _parked.Clear();
        foreach (var stream in parked)
        {
            _ = StartAsync(stream);
        }
    }

    /// <summary>
    /// Hands each tile its own decode's loudness.
    /// <b>Not part of <see cref="Apply"/>.</b>
    /// Levels arrive fifteen times a second, and the render pass at that rate would re-read the relay,
    /// the roster and every decode to move a bar.
    /// Writes only what the meter binds (<c>Backend/Session.cs</c>, <c>Metered</c>).
    /// </summary>
    public void Meter()
    {
        foreach (var tile in _tiles.Values)
        {
            tile.Meter(_session.LevelOf(tile.Name, tile.Transport));
        }
    }

    /// <summary>
    /// Does what a tile's menu or key asked for.
    /// <b>Every intent is decided here.</b>
    /// Each is a fact about the whole arrangement, so a tile raises the intent and writes none of it
    /// (<c>Features/Viewer/Model/TileIntent.cs</c>).
    /// </summary>
    private void Arrange(string stream, TileIntent intent)
    {
        Assert.That(stream.Length > 0, "an arrangement is asked for by a tile that names its stream");

        switch (intent)
        {
            case TileIntent.Focus:
                // The staged tile gives the stage up for the grid, any other takes it.
                // No state has two staged, so none has one to take away first.
                _gridAsked = Focused == stream;
                _staged = _gridAsked ? _staged : stream;
                break;

            case TileIntent.PopOut:
                // The decode is untouched either way: closing a pop-out returns the stream to its slot,
                // stopping being an intent of its own.
                if (!_popped.Remove(stream))
                {
                    _popped.Add(stream);
                }

                break;

            case TileIntent.Fullscreen:
                // Which window the stream is drawn in decides which state moves:
                // a popped-out stream fills its own window's screen and several do that at once,
                // a stream in the grid fills the main window, of which there is one.
                if (_popped.Contains(stream))
                {
                    if (!_poppedFullscreen.Remove(stream))
                    {
                        _poppedFullscreen.Add(stream);
                    }
                }
                else
                {
                    Fullscreen = Fullscreen == stream ? "" : stream;
                }

                break;

            case TileIntent.LeavePopOut:
                // The state a closed window reports, so a stream already in the grid is left where it is.
                // The slot it returns to is drawn while the main window is on screen, and the decode runs on.
                // Into a hidden grid, the stream is kept for the window's return instead,
                // that slot being in a window nothing shows.
                _popped.Remove(stream);
                break;

            case TileIntent.Close:
                // Fired without waiting: the tile leaves before the stop's round trip answers.
                _ = StopAsync(stream);
                break;

            case TileIntent.LeaveFullscreen:
                // Names the state a window is to be in,
                // so a stream filling nothing is left alone and the key is safe wherever it fires.
                if (_popped.Contains(stream))
                {
                    _poppedFullscreen.Remove(stream);
                }
                else if (Fullscreen == stream)
                {
                    Fullscreen = "";
                }

                break;
        }

        _ = Park();
        Apply();
    }

    /// <summary>
    /// Adds one tile, on the UI loop, and re-renders.
    /// The leg is passed in rather than read again,
    /// so the tile is keyed by the pair the backend keyed the decode by even where the setting moved in between.
    /// A window that hid while the start's answer was out has no grid for the tile,
    /// so the decode closes again and the stream is kept for the window's return, like the tiles the hide took.
    /// </summary>
    private void Add(string stream, string leg)
    {
        Assert.That(leg.Length > 0, "a tile names the leg its decode was opened on", stream);

        if (_tiles.ContainsKey(stream))
        {
            return;
        }

        if (!Drawn(stream))
        {
            if (!_parked.Contains(stream))
            {
                _parked.Add(stream);
            }

            _ = CloseAsync(stream, leg);
            return;
        }

        var tile = new TileViewModel(
            TileSource.Relay(stream, leg), _backend, _dispatch, intent => Arrange(stream, intent));
        // A tile reports what it drew, which no state the backend owns can carry.
        // The pass it asks for is this screen's,
        // so the figures over a tile and the strip around it stay one render function.
        tile.Changed += Apply;

        _tiles[stream] = tile;
        Tiles.Add(tile);
        Apply();
    }

    /// <summary>
    /// Takes one tile off the grid.
    /// Removing it from the collection ends the frame subscription:
    /// the control drawing it detaches, cancelling the call and disposing every handle it imported
    /// (<c>Features/Viewer/Tile/View/StreamTile.cs</c>).
    /// The caller renders: the pass drops the focus, the pop-out and the fullscreen of a stream that left
    /// the grid, and a caller taking several off runs one pass for all of them.
    /// </summary>
    private void Drop(string stream)
    {
        if (!_tiles.Remove(stream, out var tile))
        {
            return;
        }

        tile.Changed -= Apply;
        Tiles.Remove(tile);
    }

    /// <summary>
    /// Opens or closes one external viewer.
    /// No local copy of the roster is edited, the roster being the backend's list.
    /// </summary>
    private async Task WatchAsync(string stream, string transport, bool open)
    {
        try
        {
            if (open)
            {
                await _backend.StopWatchAsync(stream, transport).ConfigureAwait(false);
            }
            else
            {
                await _backend.StartWatchAsync(stream, transport).ConfigureAwait(false);
            }

            Refused("");

            // The backend announces a viewer that ended and not one that started,
            // so the roster is re-read rather than patched with what was sent.
            var watching = await _backend.WatchingAsync().ConfigureAwait(false);
            _session.Adopt(watching);
        }
        catch (BackendUnavailableException e)
        {
            Refused(e.Message);
        }
        catch (OperationCanceledException)
        {
        }
    }

    /// <summary>
    /// Opens the relay's player page for one stream in the machine's default browser.
    /// Nothing is re-read afterwards, no list having moved.
    /// The page opens in a program this app does not own, so what came of it is a state neither side can report.
    /// The refusal is, and it lands where every other one does.
    /// </summary>
    private async Task BrowseAsync(string stream, string transport)
    {
        try
        {
            await _backend.OpenInBrowserAsync(stream, transport).ConfigureAwait(false);
            Refused("");
        }
        catch (BackendUnavailableException e)
        {
            Refused(e.Message);
        }
        catch (OperationCanceledException)
        {
        }
    }

    /// <summary>
    /// Records what the backend said about the last effect and re-renders, on the UI loop.
    /// An empty reason is a success and clears whatever the last failure left.
    /// </summary>
    private void Refused(string reason)
    {
        _dispatch(() =>
        {
            Refusal = reason;
            Apply();
        });
    }
}
