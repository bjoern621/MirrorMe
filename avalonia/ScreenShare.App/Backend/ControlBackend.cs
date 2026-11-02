using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using Grpc.Core.Interceptors;
using Grpc.Net.Client;
using ScreenShare.Api.V1;
using ScreenShare.App.Contracts;

namespace ScreenShare.App.Backend;

/// <summary>
/// Control plane over the local socket: what answers <see cref="IBackend"/>, and the shell's one door to what
/// the Go backend computes.
///
/// Evaluates nothing.
/// No codec name, no encoder family, no table and no rule lives here,
/// and a greyed option arrives greyed carrying the sentence saying why (<c>docs/ipc-api.md</c>, "The rule").
/// What it owns is a transport's three things: the channel, the handshake in front of it,
/// and the translation of a failure into something the caller above can act on.
///
/// <c>Hello</c> runs before any other method and settles the contract major, so a major this backend does not
/// implement is a sentence naming both numbers rather than fields that silently arrive empty.
/// Re-run after a failure, which lets a window that opened before the backend did reach it later: nothing here
/// caches a dead connection, and the channel calls <see cref="ControlEndpoint.ConnectAsync"/> again whenever it
/// needs one.
///
/// The encoder probe is asked for once, which is why this class has an event.
/// <c>Ask form</c> reads what has been probed rather than probing, a resolve running on every keystroke
/// and the probe costing seconds, so a machine nothing has probed greys no codec for missing hardware
/// and goes on offering QSV where there is no Intel GPU.
/// This asks, in the background, and raises <see cref="Changed"/> when the answer lands.
/// What the probe decides is still the backend's; what arrives here is the news that the answer moved.
/// </summary>
public sealed class ControlBackend : IBackend
{
    /// <summary>
    /// Contract major this shell was generated against, matching Go's <c>control.ProtocolMajor</c>.
    /// Sent on every handshake, and the backend refuses a major it does not implement.
    /// </summary>
    private const uint ProtocolMajor = 2;

    /// <summary>What this shell calls itself in the backend's log. No behaviour rides on it.</summary>
    private const string ClientName = "avalonia";

    /// <summary>
    /// Bounds the handshake, shorter than every other call's bound, <c>Hello</c> answering off state the backend
    /// already holds.
    /// A backend that took the connection and did not answer it is not worth waiting on.
    /// </summary>
    private static readonly TimeSpan HandshakeDeadline = TimeSpan.FromSeconds(5);

    /// <summary>
    /// Bounds every other call, through <see cref="CallDeadline"/>.
    /// The slowest honest effect rather than a guess: taking a decode down waits on the pipeline reaching NULL,
    /// which the receive package gives two attempts of three seconds each.
    /// </summary>
    private static readonly TimeSpan CallDeadlineSpan = TimeSpan.FromSeconds(20);

    /// <summary>
    /// Probe's own bound, longer because it test-encodes on every engine the machine has
    /// and would exceed the general bound where there are several GPUs.
    /// A probe that timed out is asked for again by the next read (<see cref="GreetAsync"/>),
    /// so the general bound would mean re-running it for as long as the window is open and never finishing one.
    /// </summary>
    private static readonly TimeSpan ProbeDeadline = TimeSpan.FromMinutes(3);

    private readonly ControlService.ControlServiceClient _client;

    /// <summary>Frame channel client, over the connection the control client uses.</summary>
    private readonly FrameService.FrameServiceClient _frames;

    /// <summary>Serialises the handshake and the probe flag, so reads starting at once produce one <c>Hello</c>.</summary>
    private readonly Lock _gate = new();

    /// <summary>
    /// Handshake, kept so it is awaited rather than repeated.
    /// A faulted one is dropped and started again on the next read, which is the whole of the reconnect:
    /// the failure was an absent backend, and the next read may find one.
    /// </summary>
    private Task? _handshake;

    /// <summary>
    /// Whether the probe has been asked for.
    /// One per instance: the backend caches what it found for its own process lifetime,
    /// so a second request is a second wait for an answer already given.
    /// </summary>
    private bool _probeAsked;

    public ControlBackend()
    {
        // gRPC needs an origin on the request and a pipe or Unix socket has no host to name,
        // so the address is a placeholder and the connect callback decides where the bytes go.
        // The channel outlives this constructor through the client it is handed to: the window's connection,
        // and the window is the process.
        var channel = GrpcChannel.ForAddress("http://localhost", new GrpcChannelOptions
        {
            HttpHandler = new SocketsHttpHandler
            {
                ConnectCallback = async (_, cancellation) => await ControlEndpoint.ConnectAsync(cancellation).ConfigureAwait(false),
                // The event stream is one long-lived call and a resolve is a short one.
                // On a single connection a resolve queues behind whatever the stream is doing.
                EnableMultipleHttp2Connections = true,
            },
        });

        // Control client goes through the deadline interceptor, the frame client does not:
        // a control call is a question with an answer, a frame call stays open for as long as a tile is drawn.
        _client = new ControlService.ControlServiceClient(channel.Intercept(new CallDeadline(CallDeadlineSpan)));
        // Second service on the one connection, as the contract asks: riding the same socket avoids reinventing
        // framing, versioning and cancellation for a stream of handle metadata, and a second connection would be
        // discovered and torn down separately from the one the handshake settled.
        _frames = new FrameService.FrameServiceClient(channel);
    }

    /// <inheritdoc />
    public event Action? Changed;

    /// <inheritdoc />
    public async Task<Settings> SettingsAsync(CancellationToken cancellation = default)
    {
        var state = await ReadStateAsync(EventKind.Settings, cancellation).ConfigureAwait(false);
        return Assert.NotNull(state.Settings?.Settings, "the backend answers a settings read with settings");
    }

    /// <inheritdoc />
    public async Task<Catalog> CatalogAsync(CancellationToken cancellation = default)
    {
        var state = await ReadStateAsync(EventKind.Catalog, cancellation).ConfigureAwait(false);
        return Assert.NotNull(state.Catalog, "the backend answers a catalog read with a catalog");
    }

    /// <inheritdoc />
    public async Task<IReadOnlyList<ShareWindow>> ShareWindowsAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.ShareWindows, cancellation).ConfigureAwait(false)).ShareWindows.Windows;

    /// <inheritdoc />
    public async Task<Form> ResolveFormAsync(Settings draft, CancellationToken cancellation = default)
    {
        Assert.NotNull(draft, "resolving a form needs the draft it is resolved against");

        var answer = await AskAsync(new AskRequest { Form = draft }, cancellation).ConfigureAwait(false);
        return Assert.NotNull(answer.Form, "the backend answers a draft with its form");
    }

    /// <inheritdoc />
    public async Task<PublishState> PublishStateAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.PublishState, cancellation).ConfigureAwait(false)).PublishState;

    /// <inheritdoc />
    public async Task<RelayStatus> RelayStatusAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.RelayStatus, cancellation).ConfigureAwait(false)).RelayStatus;

    /// <inheritdoc />
    public async Task<IReadOnlyList<StreamRef>> WatchingAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.ViewerState, cancellation).ConfigureAwait(false)).ViewerState.Viewers;

    /// <inheritdoc />
    public async Task<IReadOnlyList<ReceiveStream>> ReceivingAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.ReceiveState, cancellation).ConfigureAwait(false)).ReceiveState.Streams;

    /// <inheritdoc />
    public async Task<IReadOnlyList<PreviewedMonitor>> PreviewedMonitorsAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.MonitorPreviewState, cancellation).ConfigureAwait(false))
            .MonitorPreviewState.Monitors;

    /// <inheritdoc />
    public async Task<MembersState> MembersAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.MembersState, cancellation).ConfigureAwait(false)).MembersState;

    /// <inheritdoc />
    public async Task<TestStreamState> TestStreamsAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.TestStreamState, cancellation).ConfigureAwait(false)).TestStreamState;

    /// <inheritdoc />
    public async Task<PresetStore> PresetsAsync(CancellationToken cancellation = default)
    {
        var presets = (await ReadStateAsync(EventKind.Presets, cancellation).ConfigureAwait(false)).Presets;

        // Both halves: an empty list without the notice is the wrong sentence on a machine whose store could not
        // be read (PresetStore).
        return new PresetStore(presets.Presets, presets.Notice);
    }

    /// <inheritdoc />
    public Task StartPublishAsync(Settings settings, CancellationToken cancellation = default)
    {
        Assert.NotNull(settings, "starting a publish names the settings the encoder runs on");

        return SetAsync(new SetRequest { Publish = new PublishWanted { Live = true, Settings = settings } }, cancellation);
    }

    /// <inheritdoc />
    public Task ApplyToStreamAsync(Settings settings, CancellationToken cancellation = default)
    {
        Assert.NotNull(settings, "applying to the running stream names the settings it restarts on");

        return SetAsync(new SetRequest { Publish = new PublishWanted { Live = true, Settings = settings } }, cancellation);
    }

    /// <inheritdoc />
    public Task SaveSettingsAsync(Settings settings, CancellationToken cancellation = default)
    {
        Assert.NotNull(settings, "saving the settings names the settings to keep");

        return SetAsync(new SetRequest { Settings = settings }, cancellation);
    }

    /// <inheritdoc />
    public Task SavePresetAsync(string name, PublishSettings settings, CancellationToken cancellation = default)
    {
        Assert.That(name.Length > 0, "a preset is saved under a name");
        Assert.NotNull(settings, "a preset is the way of publishing it was saved from");

        return SetAsync(new SetRequest { Preset = new PresetWanted { Name = name, Settings = settings } }, cancellation);
    }

    /// <inheritdoc />
    public Task DeletePresetAsync(string name, CancellationToken cancellation = default)
    {
        Assert.That(name.Length > 0, "a preset is deleted by the name it was saved under");

        return SetAsync(new SetRequest { Preset = new PresetWanted { Name = name } }, cancellation);
    }

    /// <inheritdoc />
    public Task StopPublishAsync(CancellationToken cancellation = default)
        => SetAsync(new SetRequest { Publish = new PublishWanted { Live = false } }, cancellation);

    /// <inheritdoc />
    public async Task<double> MeasureUplinkAsync(CancellationToken cancellation = default)
        => (await DoAsync(new DoRequest { MeasureUplink = new Empty() }, cancellation).ConfigureAwait(false)).UplinkMbps;

    /// <inheritdoc />
    public async Task<IReadOnlyList<RelayLeg>> CheckRelayAsync(
        Settings settings, CancellationToken cancellation = default)
    {
        Assert.NotNull(settings, "a check names the relay it dials");

        var outcome = await DoAsync(new DoRequest { CheckRelay = settings }, cancellation).ConfigureAwait(false);
        return Assert.NotNull(outcome.RelayCheck, "a relay check answers its legs").Legs;
    }

    /// <inheritdoc />
    public async Task<(string Key, string Id)> CreateGroupAsync(
        RelaySettings relay, CancellationToken cancellation = default)
    {
        Assert.NotNull(relay, "a group key is drawn at the relay the draft names");

        var outcome = await DoAsync(new DoRequest { CreateGroup = relay }, cancellation).ConfigureAwait(false);
        var group = Assert.NotNull(outcome.Group, "drawing a group answers the group");
        return (group.Key, group.Id);
    }

    /// <inheritdoc />
    public async Task<DiscordState> DiscordAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.DiscordState, cancellation).ConfigureAwait(false)).DiscordState;

    /// <inheritdoc />
    public async Task<string> ResolveLinkAsync(string url, CancellationToken cancellation = default)
    {
        Assert.That(url.Length > 0, "a link names a stream to open");

        return (await AskAsync(new AskRequest { Link = url }, cancellation).ConfigureAwait(false)).StreamName;
    }

    /// <inheritdoc />
    public Task LinkDiscordAsync(RelaySettings relay, CancellationToken cancellation = default)
    {
        Assert.NotNull(relay, "a link runs against the relay the draft names");

        return SetAsync(
            new SetRequest { DiscordLink = new DiscordLinkWanted { Linked = true, Relay = relay } }, cancellation);
    }

    /// <inheritdoc />
    public Task UnlinkDiscordAsync(CancellationToken cancellation = default)
        => SetAsync(new SetRequest { DiscordLink = new DiscordLinkWanted { Linked = false } }, cancellation);

    /// <inheritdoc />
    public Task StartWatchAsync(string streamName, string transport, CancellationToken cancellation = default)
        => SetAsync(
            new SetRequest { Viewer = new ViewerWanted { Viewer = Keyed(streamName, transport, "opening a viewer"), Open = true } },
            cancellation);

    /// <inheritdoc />
    public Task StopWatchAsync(string streamName, string transport, CancellationToken cancellation = default)
        => SetAsync(
            new SetRequest { Viewer = new ViewerWanted { Viewer = Keyed(streamName, transport, "closing a viewer") } },
            cancellation);

    /// <inheritdoc />
    public Task OpenInBrowserAsync(string streamName, string transport, CancellationToken cancellation = default)
        => DoAsync(
            new DoRequest { OpenInBrowser = Keyed(streamName, transport, "opening a page in the browser") }, cancellation);

    /// <inheritdoc />
    public Task StartReceiveAsync(
        string streamName, string transport, bool toneMap = false, CancellationToken cancellation = default)
        => SetAsync(
            new SetRequest
            {
                Receive = new ReceiveWanted
                {
                    Stream = Keyed(streamName, transport, "opening a decode"), Open = true, ToneMap = toneMap,
                },
            },
            cancellation);

    /// <inheritdoc />
    public Task StopReceiveAsync(string streamName, string transport, CancellationToken cancellation = default)
        => SetAsync(
            new SetRequest { Receive = new ReceiveWanted { Stream = Keyed(streamName, transport, "closing a decode") } },
            cancellation);

    /// <inheritdoc />
    public Task SetReceiveAudioAsync(
        string streamName, string transport, double volume, bool muted, CancellationToken cancellation = default)
    {
        Assert.That(volume >= 0, "a volume is not negative", volume);

        return SetAsync(
            new SetRequest
            {
                ReceiveAudio = new ReceiveAudio
                {
                    Stream = Keyed(streamName, transport, "setting a decode's audio"), Volume = volume, Muted = muted,
                },
            },
            cancellation);
    }

    /// <inheritdoc />
    public async Task<FrameChannel> OpenFramesAsync(string streamName, string transport, CancellationToken cancellation = default)
    {
        // Handshake first, like every other call: a frame channel opened before the contract major was settled
        // would be handles exchanged by two sides that never established they mean one thing by one.
        await GreetAsync().ConfigureAwait(false);

        try
        {
            return await FrameChannel.OpenAsync(_frames, streamName, transport, cancellation).ConfigureAwait(false);
        }
        catch (RpcException e)
        {
            throw Translate(e, cancellation);
        }
    }

    /// <inheritdoc />
    public async Task<FrameChannel> OpenPreviewFramesAsync(CancellationToken cancellation = default)
    {
        await GreetAsync().ConfigureAwait(false);

        try
        {
            return await FrameChannel.OpenPreviewAsync(_frames, cancellation).ConfigureAwait(false);
        }
        catch (RpcException e)
        {
            throw Translate(e, cancellation);
        }
    }

    /// <inheritdoc />
    public Task StartMonitorPreviewAsync(int monitor, CancellationToken cancellation = default)
        => SetAsync(
            new SetRequest { MonitorPreview = new MonitorPreviewWanted { Monitor = monitor, Open = true } }, cancellation);

    /// <inheritdoc />
    public Task StopMonitorPreviewAsync(int monitor, CancellationToken cancellation = default)
        => SetAsync(new SetRequest { MonitorPreview = new MonitorPreviewWanted { Monitor = monitor } }, cancellation);

    /// <inheritdoc />
    public async Task<FrameChannel> OpenMonitorFramesAsync(int monitor, CancellationToken cancellation = default)
    {
        await GreetAsync().ConfigureAwait(false);

        try
        {
            return await FrameChannel.OpenMonitorAsync(_frames, monitor, cancellation).ConfigureAwait(false);
        }
        catch (RpcException e)
        {
            throw Translate(e, cancellation);
        }
    }

    /// <inheritdoc />
    public Task OpenLogAsync(string path, CancellationToken cancellation = default)
    {
        Assert.That(path.Length > 0, "opening a run log names the log the backend handed out");

        return DoAsync(new DoRequest { OpenLog = path }, cancellation);
    }

    /// <inheritdoc />
    public Task OpenLogsFolderAsync(CancellationToken cancellation = default)
        => DoAsync(new DoRequest { OpenLogsFolder = new Empty() }, cancellation);

    /// <inheritdoc />
    public async Task<string> SendReportAsync(CancellationToken cancellation = default)
        => (await DoAsync(new DoRequest { SendReport = new Empty() }, cancellation).ConfigureAwait(false)).ReportId;

    /// <inheritdoc />
    public async Task<UpdateState> UpdateAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.UpdateState, cancellation).ConfigureAwait(false)).UpdateState;

    /// <inheritdoc />
    public async Task<StreamTimeline> StreamTimelineAsync(CancellationToken cancellation = default)
        => (await ReadStateAsync(EventKind.StreamTimeline, cancellation).ConfigureAwait(false)).StreamTimeline;

    /// <inheritdoc />
    public Task CheckUpdateAsync(CancellationToken cancellation = default)
        => DoAsync(new DoRequest { CheckUpdate = new Empty() }, cancellation);

    /// <inheritdoc />
    public Task InstallUpdateAsync(CancellationToken cancellation = default)
        => DoAsync(new DoRequest { InstallUpdate = new Empty() }, cancellation);

    /// <inheritdoc />
    public IAsyncEnumerable<Event> SubscribeAsync(CancellationToken cancellation = default)
        => WatchAsync(new WatchRequest(), cancellation);

    /// <inheritdoc />
    public async IAsyncEnumerable<PointerPosition> SubscribePointerAsync(
        StreamRef? stream = null,
        [System.Runtime.CompilerServices.EnumeratorCancellation] CancellationToken cancellation = default)
    {
        var request = new WatchRequest { Kinds = { EventKind.Pointer } };
        if (stream is not null)
        {
            request.PointerOf = stream;
        }

        await foreach (var e in WatchAsync(request, cancellation).ConfigureAwait(false))
        {
            yield return Assert.NotNull(e.Pointer, "a pointer watch delivers positions");
        }
    }

    /// <inheritdoc />
    public async IAsyncEnumerable<AudioLevels> SubscribeAudioLevelsAsync(
        [System.Runtime.CompilerServices.EnumeratorCancellation] CancellationToken cancellation = default)
    {
        var request = new WatchRequest { Kinds = { EventKind.AudioLevels } };
        await foreach (var e in WatchAsync(request, cancellation).ConfigureAwait(false))
        {
            yield return Assert.NotNull(e.AudioLevels, "a level watch delivers levels");
        }
    }

    /// <summary>
    /// One watch, with the handshake in front of it and every failure translated.
    /// The call opens outside the loop, so a failure to open it is translated like a failed read
    /// rather than surfacing from inside an enumeration the caller is already iterating.
    /// </summary>
    private async IAsyncEnumerable<Event> WatchAsync(
        WatchRequest request,
        [System.Runtime.CompilerServices.EnumeratorCancellation] CancellationToken cancellation)
    {
        await GreetAsync().ConfigureAwait(false);

        using var call = _client.Watch(request, cancellationToken: cancellation);

        while (true)
        {
            bool more;
            try
            {
                more = await call.ResponseStream.MoveNext(cancellation).ConfigureAwait(false);
            }
            catch (RpcException e)
            {
                throw Translate(e, cancellation);
            }

            if (!more)
            {
                yield break;
            }

            yield return call.ResponseStream.Current;
        }
    }

    /// <summary>One state, whole, as the backend holds it now.</summary>
    private async Task<Event> ReadStateAsync(EventKind kind, CancellationToken cancellation)
    {
        var answer = await CallAsync(
            c => c.ReadAsync(new ReadRequest { Kinds = { kind } }, cancellationToken: cancellation), cancellation)
            .ConfigureAwait(false);

        Assert.That(answer.States.Count == 1, "a read of one kind answers one state", kind, answer.States.Count);
        return answer.States[0];
    }

    private Task<AskResponse> AskAsync(AskRequest question, CancellationToken cancellation)
        => CallAsync(c => c.AskAsync(question, cancellationToken: cancellation), cancellation);

    private Task SetAsync(SetRequest state, CancellationToken cancellation)
        => CallAsync(c => c.SetAsync(state, cancellationToken: cancellation), cancellation);

    private Task<DoResponse> DoAsync(DoRequest action, CancellationToken cancellation)
        => CallAsync(c => c.DoAsync(action, cancellationToken: cancellation), cancellation);

    /// <summary>
    /// One call, with the handshake in front of it and the failure translated behind it.
    /// The delegate takes the client rather than closing over it, so nothing here decides which call is made.
    /// </summary>
    private async Task<T> CallAsync<T>(
        Func<ControlService.ControlServiceClient, AsyncUnaryCall<T>> call, CancellationToken cancellation)
    {
        await GreetAsync().ConfigureAwait(false);

        try
        {
            return await call(_client).ResponseAsync.ConfigureAwait(false);
        }
        catch (RpcException e)
        {
            throw Translate(e, cancellation);
        }
    }

    /// <summary>
    /// The stream and leg a viewer and a decode are identified by, asserted where every effect on them builds it.
    /// <paramref name="what"/> names the effect as its assertions name it.
    /// </summary>
    private static StreamRef Keyed(string streamName, string transport, string what)
    {
        Assert.That(streamName.Length > 0, $"{what} names the stream it is for");
        Assert.That(transport.Length > 0, $"{what} names the leg it runs over", streamName);

        return new StreamRef { StreamName = streamName, Transport = transport };
    }

    /// <summary>
    /// Settles the contract version, once, and asks for the encoder probe behind it.
    ///
    /// Handshake shared rather than per call, so two reads racing produce one <c>Hello</c>.
    /// Deliberately not given the caller's token: one caller abandoning its read would cancel the handshake every
    /// other caller is waiting on.
    ///
    /// The probe is asked for here rather than from inside the handshake: a settled handshake is kept and never
    /// re-run, so a flag only it consults is a flag nothing reads again (<see cref="Probe"/>).
    /// <see cref="Probe"/> is idempotent, so every read after the first asks for a state that already holds.
    /// </summary>
    private async Task GreetAsync()
    {
        Task handshake;
        lock (_gate)
        {
            if (_handshake is null || _handshake.IsFaulted || _handshake.IsCanceled)
            {
                _handshake = HelloAsync();
            }

            handshake = _handshake;
        }

        await handshake.ConfigureAwait(false);
        Probe();
    }

    private async Task HelloAsync()
    {
        HelloResponse hello;
        try
        {
            hello = await _client.HelloAsync(
                new HelloRequest { Client = ClientName, ProtocolMajor = ProtocolMajor },
                deadline: DateTime.UtcNow.Add(HandshakeDeadline)).ResponseAsync.ConfigureAwait(false);
        }
        catch (RpcException e)
        {
            throw Translate(e, CancellationToken.None);
        }

        Assert.That(
            hello.ProtocolMajor == ProtocolMajor,
            "a settled handshake leaves both sides on one contract major", ProtocolMajor, hello.ProtocolMajor);

        lock (_gate)
        {
            _backendVersion = hello.BackendVersion;
        }
    }

    /// <summary>
    /// The build behind the socket, which the handshake already carries.
    /// Read through the handshake rather than held by a caller, so a reconnect answers the build now serving.
    /// </summary>
    private string _backendVersion = "";

    /// <inheritdoc />
    public async Task<string> VersionAsync(CancellationToken cancellation = default)
    {
        await GreetAsync().ConfigureAwait(false);

        lock (_gate)
        {
            return _backendVersion;
        }
    }

    /// <summary>
    /// Asks the backend to run the encoder probe, once, without waiting for it.
    ///
    /// Not awaited: the probe test-encodes on every engine and takes seconds, so a read that waited would put
    /// those seconds in front of the window's first form.
    /// The screen paints what is known, greying nothing for missing hardware, and <see cref="Changed"/> tells it
    /// to read again when the answer lands.
    ///
    /// A probe that failed is not remembered as done: nothing was probed, so the next read asks again
    /// (<see cref="GreetAsync"/>).
    /// </summary>
    private void Probe()
    {
        lock (_gate)
        {
            if (_probeAsked)
            {
                return;
            }

            _probeAsked = true;
        }

        _ = ProbeAsync();
    }

    private async Task ProbeAsync()
    {
        try
        {
            await _client.DoAsync(
                new DoRequest { ProbeEncoders = new Empty() }, deadline: DateTime.UtcNow.Add(ProbeDeadline))
                .ResponseAsync.ConfigureAwait(false);
        }
        catch (RpcException)
        {
            lock (_gate)
            {
                _probeAsked = false;
            }

            return;
        }

        // Raised on whichever thread the call completed on (IBackend.Changed).
        Changed?.Invoke();
    }

    /// <summary>
    /// Turns a failed call into what the caller above is written against: a read this shell abandoned,
    /// or a sentence the reader sees.
    /// The two are not gRPC's own division, so the translation is here rather than in a <c>catch</c> upstairs.
    ///
    /// A superseded resolve is nobody's business, the flow cancelling one on every keystroke,
    /// so it becomes an <see cref="OperationCanceledException"/>.
    /// The token is checked alongside the code,
    /// so a <c>CANCELLED</c> the backend produced on its own is not mistaken for one this shell asked for.
    ///
    /// Everything else divides by who wrote the status, whatever code it carries.
    /// A status the backend served carries a <see cref="Refusal"/>, read as this shell's sentence for its code:
    /// a relay that could not be reached
    /// and a child process that would not start are both <c>UNAVAILABLE</c> (<c>docs/ipc-api.md</c>, "Errors"),
    /// so reading that code as "nothing is listening" would answer Start sharing with a sentence about its own connection.
    /// The one thing this side may add is the address, and only where the connection is what failed.
    ///
    /// <see cref="Status.DebugException"/> answers which side wrote it:
    /// the client library sets it on a status it made from a local failure and leaves it null on one that arrived.
    /// A refused connect is told apart from a served refusal by code rather than by matching on a sentence,
    /// the input that changes without anything failing to compile.
    ///
    /// A local failure is the backend not running whatever code it wears:
    /// an absent named pipe arrives as <c>INTERNAL</c> on Windows with nothing said,
    /// an unbound socket as <c>UNAVAILABLE</c>.
    /// Both name the address that was tried (<see cref="ControlEndpoint.Describe"/>).
    ///
    /// A served status carrying no refusal comes from a backend of another build, and names its code:
    /// the exception promises a sentence, and the assertion upstairs refuses a blank one.
    /// </summary>
    internal static Exception Translate(RpcException e, CancellationToken cancellation)
    {
        if (e.StatusCode == StatusCode.Cancelled && cancellation.IsCancellationRequested)
        {
            return new OperationCanceledException(cancellation);
        }

        // This side's own clock running out, so it is named before the check below: the status was written here
        // and carries no prose, and the wording for a prose-less status says the backend answered.
        //
        // Whether the call landed is not visible from here.
        // Repeating a Set finds the work already done.
        if (e.StatusCode == StatusCode.DeadlineExceeded)
        {
            return new BackendUnavailableException(
                $"The backend did not answer in time over {ControlEndpoint.Describe()}. "
                + "Whether it acted before going quiet is not visible from here, "
                + "so anything that names a state (a stream, a viewer or a decode) "
                + "is safe to ask for again.", e);
        }

        if (e.Status.DebugException is not null)
        {
            return new BackendUnavailableException(
                $"The backend is not running: nothing is listening on {ControlEndpoint.Describe()} ({e.StatusCode}).", e);
        }

        var refusal = RefusalOf(e);
        if (refusal is null)
        {
            return new BackendUnavailableException(
                $"The backend answered {e.StatusCode} over {ControlEndpoint.Describe()} and said nothing further.", e);
        }

        return new BackendUnavailableException(Copy.Refusals.Of(refusal), e);
    }

    /// <summary>The Refusal a served status carries in its details trailer, null where it carries none.</summary>
    private static Refusal? RefusalOf(RpcException e)
    {
        var packed = e.Trailers.GetValueBytes("grpc-status-details-bin");
        if (packed is null)
        {
            return null;
        }

        var status = Google.Rpc.Status.Parser.ParseFrom(packed);
        foreach (var detail in status.Details)
        {
            if (detail.TryUnpack<Refusal>(out var refusal))
            {
                return refusal;
            }
        }
        return null;
    }
}
