using MirrorMe.Api.V1;
using MirrorMe.App.Backend;

namespace MirrorMe.App.Tests;

/// <summary>
/// A backend carrying the named relay paths.
/// Leg is stated in the stored settings and in the form both:
/// settings are what a decode opens on, the form is what the panel draws,
/// and stating it once leaves a screen showing a leg no decode uses
/// (Features/Viewer/Tile/Model/TileLeg.cs).
/// Everything outside the relay and the viewer group is the seed's, and every effect answers at once.
/// </summary>
internal sealed class ViewerBackend(params string[] paths) : IBackend
{
    private readonly SeededBackend _seed = new("windows");

    private const string Leg = "rtsp";

    /// <summary>Group and publish state the seed answers with, for a test that states either.</summary>
    public SeededBackend Seed => _seed;

    public Task<string> VersionAsync(CancellationToken cancellation = default) => _seed.VersionAsync(cancellation);

    public event Action? Changed
    {
        add { }
        remove { }
    }

    private readonly List<StreamRef> _decoding = [];

    private string[] _paths = paths;

    /// <summary>
    /// Answer every start waits on, null while starts answer at once.
    /// The decode is open the moment it is asked for; the answer is what is late.
    /// </summary>
    private TaskCompletionSource? _heldStart;

    public void HoldStarts() => _heldStart = new TaskCompletionSource();

    /// <summary>Lets every held start answer, off the test's own context (<see cref="Answers.Now"/>).</summary>
    public void AnswerStarts()
    {
        var held = _heldStart;
        _heldStart = null;
        held?.SetResult();
    }

    /// <summary>
    /// Answers the held stops wait on, one per call, empty while stops answer at once.
    /// One per call rather than one shared: the runtime resumes a second awaiter of one task off the thread,
    /// and a test then reads a state still being written.
    /// </summary>
    private readonly Queue<TaskCompletionSource> _heldStops = [];

    private bool _holdsStops;

    public void HoldStops() => _holdsStops = true;

    /// <summary>Lets every held stop answer, in order, off the test's own context (<see cref="Answers.Now"/>).</summary>
    public void AnswerStops()
    {
        _holdsStops = false;
        while (_heldStops.TryDequeue(out var held))
        {
            held.SetResult();
        }
    }

    /// <summary>Starts asked for, a repeat included, so a pass that asks again is caught.</summary>
    public int Starts { get; private set; }

    /// <summary>Replaces what the relay lists, read on the next load.</summary>
    public void Carry(params string[] paths) => _paths = paths;

    /// <summary>Decodes still open, for asserting a stop reached the backend.</summary>
    public IReadOnlyList<StreamRef> Decoding => _decoding;

    /// <summary>
    /// Ends one decode on its own, the way a publisher stopping ends it: gone from the decodes,
    /// announced on the next subscription.
    /// </summary>
    public void End(string streamName)
    {
        var streamRef = new StreamRef { StreamName = streamName, Transport = Leg };
        _decoding.Remove(streamRef);
        _seed.Announcements.Add(new Event { ReceiveExit = new ReceiveExit { Stream = streamRef, Message = "stream ended" } });
    }

    public Task<RelayStatus> RelayStatusAsync(CancellationToken cancellation = default)
    {
        var relay = new RelayStatus { Reachable = true };
        foreach (var path in _paths)
        {
            relay.Paths.Add(new RelayPath { Name = path, Ready = true });
        }

        return Task.FromResult(relay);
    }

    public Task<IReadOnlyList<ReceiveStream>> ReceivingAsync(CancellationToken cancellation = default)
        => Task.FromResult<IReadOnlyList<ReceiveStream>>(
            _decoding.Select(streamRef => new ReceiveStream { Stream = streamRef, Live = true }).ToList());

    public Task<Form> ResolveFormAsync(Settings draft, CancellationToken cancellation = default)
    {
        var form = new Form { Settings = draft.Clone() };
        var group = new FieldGroup { Key = "viewer" };
        group.Fields.Add(new Field
        {
            Key = "viewer.tile_watch_transport",
            Control = ControlKind.Select,
            Visible = true,
            Enabled = true,
            Value = new FieldValue { Text = Leg },
        });

        form.Groups.Add(group);
        return Task.FromResult(form);
    }

    public Task StartReceiveAsync(
        string streamName, string transport, bool toneMap = false, CancellationToken cancellation = default)
    {
        var streamRef = new StreamRef { StreamName = streamName, Transport = transport };
        if (!_decoding.Contains(streamRef))
        {
            _decoding.Add(streamRef);
        }

        Starts++;
        return _heldStart?.Task ?? Task.CompletedTask;
    }

    public Task StopReceiveAsync(string streamName, string transport, CancellationToken cancellation = default)
    {
        _decoding.Remove(new StreamRef { StreamName = streamName, Transport = transport });
        if (!_holdsStops)
        {
            return Task.CompletedTask;
        }

        var held = new TaskCompletionSource();
        _heldStops.Enqueue(held);
        return held.Task;
    }

    public async Task<Settings> SettingsAsync(CancellationToken cancellation = default)
    {
        var settings = await _seed.SettingsAsync(cancellation);
        settings.Viewer.TileWatchTransport = Leg;
        return settings;
    }

    public Task<Catalog> CatalogAsync(CancellationToken cancellation = default)
        => _seed.CatalogAsync(cancellation);

    public Task<IReadOnlyList<ShareWindow>> ShareWindowsAsync(CancellationToken cancellation = default)
        => _seed.ShareWindowsAsync(cancellation);

    public Task<PublishState> PublishStateAsync(CancellationToken cancellation = default)
        => _seed.PublishStateAsync(cancellation);

    public Task<IReadOnlyList<StreamRef>> WatchingAsync(CancellationToken cancellation = default)
        => _seed.WatchingAsync(cancellation);

    public Task StartPublishAsync(Settings settings, CancellationToken cancellation = default)
        => _seed.StartPublishAsync(settings, cancellation);

    public Task SaveSettingsAsync(Settings settings, CancellationToken cancellation = default)
        => _seed.SaveSettingsAsync(settings, cancellation);

    public Task ApplyToStreamAsync(Settings settings, CancellationToken cancellation = default)
        => _seed.ApplyToStreamAsync(settings, cancellation);

    public Task StopPublishAsync(CancellationToken cancellation = default)
        => _seed.StopPublishAsync(cancellation);

    public Task<double> MeasureUplinkAsync(CancellationToken cancellation = default)
        => _seed.MeasureUplinkAsync(cancellation);

public Task<IReadOnlyList<RelayLeg>> CheckRelayAsync(Settings settings, CancellationToken cancellation = default)
        => _seed.CheckRelayAsync(settings, cancellation);



    public Task<(string Key, string Id)> CreateGroupAsync(RelaySettings relay, CancellationToken cancellation = default)
        => _seed.CreateGroupAsync(relay, cancellation);

    public Task<MembersState> MembersAsync(CancellationToken cancellation = default)
        => _seed.MembersAsync(cancellation);

    public Task<DiscordState> DiscordAsync(CancellationToken cancellation = default)
        => _seed.DiscordAsync(cancellation);


    public Task<string> ResolveLinkAsync(string url, CancellationToken cancellation = default)

        => _seed.ResolveLinkAsync(url, cancellation);

    public Task LinkDiscordAsync(RelaySettings relay, CancellationToken cancellation = default)
        => _seed.LinkDiscordAsync(relay, cancellation);

    public Task UnlinkDiscordAsync(CancellationToken cancellation = default)
        => _seed.UnlinkDiscordAsync(cancellation);

    public Task<TestStreamState> TestStreamsAsync(CancellationToken cancellation = default)
        => _seed.TestStreamsAsync(cancellation);

    public Task<PresetStore> PresetsAsync(CancellationToken cancellation = default)
        => _seed.PresetsAsync(cancellation);

    public Task SavePresetAsync(string name, PublishSettings settings, CancellationToken cancellation = default)
        => _seed.SavePresetAsync(name, settings, cancellation);

    public Task DeletePresetAsync(string name, CancellationToken cancellation = default)
        => _seed.DeletePresetAsync(name, cancellation);

    public Task StartWatchAsync(string streamName, string transport, CancellationToken cancellation = default)
        => _seed.StartWatchAsync(streamName, transport, cancellation);

    public Task StopWatchAsync(string streamName, string transport, CancellationToken cancellation = default)
        => _seed.StopWatchAsync(streamName, transport, cancellation);

    public Task OpenInBrowserAsync(string streamName, string transport, CancellationToken cancellation = default)
        => _seed.OpenInBrowserAsync(streamName, transport, cancellation);

    public Task SetReceiveAudioAsync(
        string streamName, string transport, double volume, bool muted, CancellationToken cancellation = default)
        => _seed.SetReceiveAudioAsync(streamName, transport, volume, muted, cancellation);

    public Task<FrameChannel> OpenFramesAsync(string streamName, string transport, CancellationToken cancellation = default)
        => _seed.OpenFramesAsync(streamName, transport, cancellation);

    public Task<FrameChannel> OpenPreviewFramesAsync(CancellationToken cancellation = default)
        => _seed.OpenPreviewFramesAsync(cancellation);

    public Task StartMonitorPreviewAsync(int monitor, CancellationToken cancellation = default)
        => _seed.StartMonitorPreviewAsync(monitor, cancellation);

    public Task StopMonitorPreviewAsync(int monitor, CancellationToken cancellation = default)
        => _seed.StopMonitorPreviewAsync(monitor, cancellation);

    public Task<FrameChannel> OpenMonitorFramesAsync(int monitor, CancellationToken cancellation = default)
        => _seed.OpenMonitorFramesAsync(monitor, cancellation);

    public Task<IReadOnlyList<PreviewedMonitor>> PreviewedMonitorsAsync(CancellationToken cancellation = default)
        => _seed.PreviewedMonitorsAsync(cancellation);

    public Task OpenLogAsync(string path, CancellationToken cancellation = default)
        => _seed.OpenLogAsync(path, cancellation);

    public Task OpenLogsFolderAsync(CancellationToken cancellation = default)
        => _seed.OpenLogsFolderAsync(cancellation);

    public Task<string> SendReportAsync(CancellationToken cancellation = default)
        => _seed.SendReportAsync(cancellation);

    public Task<UpdateState> UpdateAsync(CancellationToken cancellation = default)
        => _seed.UpdateAsync(cancellation);

    public Task<StreamTimeline> StreamTimelineAsync(CancellationToken cancellation = default)
        => _seed.StreamTimelineAsync(cancellation);

    public Task CheckUpdateAsync(CancellationToken cancellation = default)
        => _seed.CheckUpdateAsync(cancellation);

    public Task InstallUpdateAsync(CancellationToken cancellation = default)
        => _seed.InstallUpdateAsync(cancellation);

    public IAsyncEnumerable<Event> SubscribeAsync(CancellationToken cancellation = default)
        => _seed.SubscribeAsync(cancellation);

    public IAsyncEnumerable<AudioLevels> SubscribeAudioLevelsAsync(CancellationToken cancellation = default)
        => _seed.SubscribeAudioLevelsAsync(cancellation);

    public IAsyncEnumerable<PointerPosition> SubscribePointerAsync(
        StreamRef? stream = null, CancellationToken cancellation = default)
        => _seed.SubscribePointerAsync(stream, cancellation);
}
