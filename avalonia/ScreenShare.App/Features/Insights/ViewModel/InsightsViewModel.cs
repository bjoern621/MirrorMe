using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Contracts;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Insights.Chart.Model;
using ScreenShare.App.Features.Insights.Chart.ViewModel;
using ScreenShare.App.Features.Insights.Model;
using ScreenShare.App.Features.Insights.Preview.ViewModel;
using ScreenShare.App.Features.Insights.Verdict.ViewModel;
using ScreenShare.App.Features.Insights.ViewerTable.ViewModel;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Insights.ViewModel;

/// <summary>
/// Insights screen: is the stream reaching people as it should, and if not, why (<c>docs/insights.md</c>).
/// The verdict answers in a sentence, the value cards hold the three figures that explain it,
/// the chart the history of the one card picked, and the Viewers tab what each receiving end measures.
///
/// Holds no reading of its own.
/// <see cref="Session"/> owns the running state and the backend's timeline,
/// and <see cref="Apply"/> pushes one composed reading into every part on each pass,
/// so two parts cannot disagree about the stream they describe.
///
/// Stop is owned here and drawn on the tray and the Go-strip
/// (<c>Tray/ViewModel/TrayViewModel.cs</c>, <c>Shell/Go/ViewModel/GoViewModel.cs</c>).
/// </summary>
public sealed class InsightsViewModel : Observable
{
    /// <summary>Window width, px, from which the preview stands beside the cards.</summary>
    private const double WideFrom = 600;

    /// <summary>Preview width beside the cards, px.</summary>
    private const double PreviewBeside = 320;

    private readonly IBackend _backend;
    private readonly Session _session;

    /// <summary>
    /// What each figure last measured, for the passes where nothing did,
    /// so a card holds its figure rather than blinking.
    /// </summary>
    private readonly HeldFigures _held = new();

    /// <summary>Where a refusal is shown. Set from outside, so a screen built with no shell still renders.</summary>
    private Action<string> _report = static _ => { };

    private double _window = double.PositiveInfinity;

    /// <param name="form">Settings the backend holds, which the preview's end-to-end route reads its leg from.</param>
    /// <param name="dispatch">Hands work to the UI loop, where every binding here is written.</param>
    public InsightsViewModel(IBackend backend, FormSession form, Session session, Action<Action> dispatch)
    {
        Assert.NotNull(backend, "an insights screen asks the backend to act");
        Assert.NotNull(form, "an insights screen hands the preview the settings it reads its leg from");
        Assert.NotNull(session, "an insights screen renders the session's running state");
        Assert.NotNull(dispatch, "an insights screen needs a UI loop to marshal an answer back to");

        _backend = backend;
        _session = session;

        Preview = new PreviewViewModel(backend, form, session, dispatch);
        Verdict = new VerdictViewModel(OpenLogAsync, dispatch);
        Chart = new ChartViewModel();
        Viewers = new ViewerTableViewModel();

        StreamTab = new InsightsTabRow(InsightsTab.Stream, InsightsCopy.StreamTab);
        ViewersTab = new InsightsTabRow(InsightsTab.Viewers, InsightsCopy.ViewersTab(null));
        Tabs = [StreamTab, ViewersTab];
        _selectedTab = StreamTab;

        StopCommand = new PendingCommand(() => PerformAsync(_backend.StopPublishAsync), dispatch, () => IsLive);
        Refused = reason => dispatch(() =>
        {
            Refusal = reason;
            _report(reason);
            Apply();
        });

        Apply();
    }

    /// <summary>Hands the screen somewhere to put a refusal. Idempotent, and renders after.</summary>
    public void Attach(Action<string> report)
    {
        Assert.NotNull(report, "a refusal needs a window to report it in");

        _report = report;
        Apply();
    }

    // --- Outputs ------------------------------------------------------------------

    private InsightsSnapshot _snapshot = InsightsSnapshot.Unread;
    private InsightsTabRow _selectedTab;
    private string _title = "";
    private string _subtitle = "";
    private string _refusal = "";

    public PreviewViewModel Preview { get; }

    public VerdictViewModel Verdict { get; }

    public ChartViewModel Chart { get; }

    public ViewerTableViewModel Viewers { get; }

    public InsightsTabRow StreamTab { get; }

    public InsightsTabRow ViewersTab { get; }

    public IReadOnlyList<InsightsTabRow> Tabs { get; }

    public InsightsTabRow SelectedTab
    {
        get => _selectedTab;
        set
        {
            Assert.NotNull(value, "one tab is always open");

            if (Set(ref _selectedTab, value))
            {
                OnPropertyChanged(nameof(ShowsStream));
                OnPropertyChanged(nameof(ShowsViewers));
            }
        }
    }

    public bool ShowsStream => SelectedTab.Value == InsightsTab.Stream;

    public bool ShowsViewers => SelectedTab.Value == InsightsTab.Viewers;

    /// <summary>What is shared and with whom, empty while nothing is.</summary>
    public string Title { get => _title; private set => Set(ref _title, value); }

    /// <summary>Way and running time: "Text and detail over SRT · 42:17".</summary>
    public string Subtitle { get => _subtitle; private set => Set(ref _subtitle, value); }

    public bool HasTitle => Title.Length > 0;

    public static string ViewersNote => InsightsCopy.ViewersNote;

    /// <summary>Beside the cards on a wide window, over them on a narrow one.</summary>
    public bool IsWide => _window >= WideFrom;

    public double PreviewWidth => IsWide ? PreviewBeside : double.NaN;

    public int CardsRow => IsWide ? 0 : 1;

    public int CardsColumn => IsWide ? 1 : 0;

    public int PreviewSpan => IsWide ? 1 : 2;

    /// <summary>Cards stack in one column beside the preview and stand in a row under it.</summary>
    public int CardColumns => IsWide ? 1 : 3;

    /// <summary>Word on the stop control, wherever it is drawn: the tray's row and the Go-strip's button.</summary>
    public const string StopLabel = "Stop sharing";

    /// <summary>That word where the strip has no room for it.</summary>
    public const string StopShort = "Stop";

    public PendingCommand StopCommand { get; }

    public InsightsSnapshot Snapshot { get => _snapshot; private set => Set(ref _snapshot, value); }

    /// <summary>Refusal of something this screen asked for, empty otherwise.</summary>
    public string Refusal
    {
        get => _refusal;
        private set
        {
            if (Set(ref _refusal, value))
            {
                OnPropertyChanged(nameof(HasRefusal));
            }
        }
    }

    public bool HasRefusal => Refusal.Length > 0;

    private bool IsLive => Snapshot.IsLive;

    private Action<string> Refused { get; }

    /// <summary>States the window's width, which decides where the preview stands. Idempotent.</summary>
    public void SetWindowWidth(double width)
    {
        Assert.That(width > 0, "a window states a width it has", width);

        _window = width;
        OnPropertyChanged(nameof(IsWide));
        OnPropertyChanged(nameof(PreviewWidth));
        OnPropertyChanged(nameof(CardsRow));
        OnPropertyChanged(nameof(CardsColumn));
        OnPropertyChanged(nameof(PreviewSpan));
        OnPropertyChanged(nameof(CardColumns));
    }

    /// <summary>States whether the window is on screen, which the preview's picture follows. Idempotent.</summary>
    public void SetWindowShown(bool shown) => Preview.SetWindowShown(shown);

    /// <summary>Hides ahead of a quit, answering once the preview's decode is closed.</summary>
    public Task PartAsync() => Preview.PartAsync();

    /// <summary>
    /// The one render function.
    /// Composes the reading from the session and pushes it into every part.
    /// Safe to run twice: a part whose input did not move writes nothing.
    /// </summary>
    public void Apply()
    {
        var reading = _held.Fill(InsightsSnapshot.Of(_session.Publish, _session.Stats, _session.Relay));
        Snapshot = reading;
        var live = _session.Publish?.Live;

        Title = live is null ? "" : InsightsCopy.Title(Target(live.Publish), Group(_session.Discord));
        OnPropertyChanged(nameof(HasTitle));
        Subtitle = live is null ? "" : Figure.Join(reading.Way, reading.Elapsed);

        Verdict.Apply(_session.Timeline, reading);

        Preview.Snapshot = reading;
        Preview.Apply();

        var limits = new Limits(live?.Publish.Fps is > 0 and var fps ? fps : null, reading.VbvCeilingMbps);
        Chart.Apply(_session.Samples, reading, limits);

        Viewers.Reported = Watching(_session.Relay, reading.Stream);
        Viewers.Readers = reading.Viewers;
        Viewers.IsLive = reading.IsLive;
        Viewers.Apply();
        ViewersTab.Label = InsightsCopy.ViewersTab(reading.Viewers);

        StopCommand.Refresh();

        Assert.That(Preview.Snapshot == reading, "every part describes one reading", reading.Elapsed);
        Assert.That(HasTitle == (live is not null), "the header names a stream while one runs", HasTitle);
    }

    /// <summary>What a running stream shares, in the words the setup's capture step uses.</summary>
    private static string Target(PublishSettings settings) => settings.ShareKind switch
    {
        "window" => "A window",
        "region" => $"Part of Screen {settings.Monitor}",
        _ => $"Screen {settings.Monitor}",
    };

    private static string Group(DiscordState? discord)
        => discord is { InChannel: true, ChannelName.Length: > 0 } ? discord.ChannelName : InsightsCopy.YourGroup;

    /// <summary>
    /// Viewer rows: one per viewer report on this stream's path, in member order,
    /// so a row a reader learned the position of stays there.
    /// </summary>
    private static IReadOnlyList<ViewerRow> Watching(RelayStatus? relay, string stream)
    {
        var path = InsightsSnapshot.PathOf(relay, stream);
        return path is null ? [] : path.Viewers.Select(ViewerRow.Of).ToList();
    }

    // --- The effects ----------------------------------------------------------------

    /// <summary>Opens the stopped run's log. The file sits on the backend's machine, so the backend opens it.</summary>
    private Task OpenLogAsync(string path)
    {
        Assert.That(path.Length > 0, "a log is opened by the path the verdict named");

        return PerformAsync(cancellation => _backend.OpenLogAsync(path, cancellation));
    }

    /// <summary>
    /// Asks the backend for one effect and shows its refusal (<c>docs/ipc-api.md</c>, "Errors").
    /// A success clears what the last one left.
    /// </summary>
    private async Task PerformAsync(Func<CancellationToken, Task> effect)
    {
        try
        {
            await effect(default).ConfigureAwait(false);
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
}
