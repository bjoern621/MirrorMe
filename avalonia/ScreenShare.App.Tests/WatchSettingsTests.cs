using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Features.Fields.Model;
using ScreenShare.App.Features.Fields.ViewModel;
using ScreenShare.App.Features.Setup.ViewModel;
using ScreenShare.App.Features.Viewer.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Where the settings about receiving live, and what they do.
///
/// The watch group is drawn over the stage and applied, so a choice there is kept as it is made,
/// and watching without publishing needs neither setup nor a <c>Set publish</c>.
/// </summary>
public sealed class WatchSettingsTests
{
    private static readonly Action<Action> Inline = action => action();

    /// <summary>A viewer and a wizard on one draft, built the way the window builds them.</summary>
    private sealed record Both(
        ViewerViewModel Viewer, SetupViewModel Setup, FormSession Form, SeededBackend Backend);

    private static async Task<Both> BothAsync(SeededBackend? seed = null)
    {
        var backend = seed ?? new SeededBackend("linux");
        var session = new Session(backend, Inline);
        var form = new FormSession(backend, session, Inline);

        var viewer = new ViewerViewModel(backend, form, session, Inline);
        var setup = Flows.Setup(backend, form, session);

        // Stands in for the shell's own render pass: without it a relay snapshot reaches the session
        // and never the roster drawn from it (Features/Shell/ViewModel/ShellViewModel.cs).
        session.Changed += viewer.Apply;

        // Reads every running state once and stops before the reconnect delay, so nothing is left dialling
        // behind the assertions.
        // Puts the relay's paths on the roster a decode is opened from.
        _ = session.Start();
        session.Stop();

        await form.Settled;
        return new Both(viewer, setup, form, backend);
    }

    private static FieldViewModel Field(Both both, string key)
        => both.Viewer.Watch.Group.Fields.Single(field => field.Key == key);

    /// <summary>Chooses an option and waits for the form that answers it.</summary>
    private static async Task ChooseAsync(Both both, string key, string value)
    {
        Field(both, key).Options.Single(option => option.Value == value).Choose.Execute(null);
        await both.Form.Settled;
    }

    /// <summary>The group the wizard does not draw, with every field the form marked visible.</summary>
    [Fact]
    public async Task TheViewerDrawsTheWatchingGroup()
    {
        var both = await BothAsync();
        var group = (await both.Backend.ResolveFormAsync(await both.Backend.SettingsAsync()))
            .Groups.Single(g => GroupPlacement.InViewer(g.Key));

        Assert.True(both.Viewer.Watch.Group.IsResolved);
        Assert.Equal(
            group.Fields.Where(field => field.Visible).Select(field => field.Key),
            both.Viewer.Watch.Group.Fields.Select(field => field.Key));
    }

    /// <summary>
    /// The panel opens on its question and the ways a stream can arrive, named for what a viewer notices.
    /// Every control waits in the fold, the protocol among them.
    /// </summary>
    [Fact]
    public async Task ThePanelAsksHowStreamsArriveAndFoldsTheRest()
    {
        var both = await BothAsync();
        var watch = both.Viewer.Watch;

        Assert.Equal("How should streams arrive?", watch.Group.Heading);
        Assert.Equal(["rtsp", "srt", "hls"], watch.Arrivals.Select(card => card.Value));
        Assert.Equal(["Low delay", "Steady", "Compatible"], watch.Arrivals.Select(card => card.Label));
        Assert.Empty(watch.Group.Floor);
        Assert.Contains(watch.Group.Folded, field => field.Key == "viewer.tile_watch_transport");
    }

    /// <summary>A card writes the protocol it stands for, and the card for the protocol in the draft is the picked one.</summary>
    [Fact]
    public async Task ACardPicksTheProtocolItStandsFor()
    {
        var both = await BothAsync();

        both.Viewer.Watch.Arrivals.Single(card => card.Label == "Steady").Choose.Execute(null);
        await both.Form.Settled;

        Assert.Equal("srt", both.Form.Draft!.Viewer.TileWatchTransport);
        Assert.Equal(["srt"], both.Viewer.Watch.Arrivals.Where(card => card.IsSelected).Select(card => card.Value));
    }

    /// <summary>
    /// A protocol that no card stands for is picked in the fold, and then no card is picked:
    /// a card marked for it would claim a trade that the protocol does not make.
    /// </summary>
    [Fact]
    public async Task AProtocolPickedInTheFoldLeavesEveryCardUnpicked()
    {
        var both = await BothAsync();

        await ChooseAsync(both, "viewer.tile_watch_transport", "webrtc");

        Assert.DoesNotContain(both.Viewer.Watch.Arrivals, card => card.IsSelected);
    }

    /// <summary>
    /// One draft behind both screens.
    /// The publish persists the whole settings message, so a draft per screen would overwrite the other's half.
    /// </summary>
    [Fact]
    public async Task AWriteInTheViewerIsInWhatTheWizardCommits()
    {
        // A backend the commit can be pressed against: the seeded fixture has no reachable relay,
        // and a reachable one is what unlocks the button.
        var backend = new PublishingBackend();
        var session = new Session(backend, Inline);
        var form = new FormSession(backend, session, Inline);
        var viewer = new ViewerViewModel(backend, form, session, Inline);
        var setup = Flows.Setup(backend, form, session);

        // Reads every running state once and stops before the reconnect delay.
        _ = session.Start();
        session.Stop();
        await form.Settled;

        var chain = viewer.Watch.Group.Fields.Single(field => field.Key == "viewer.render_chain");
        chain.Options.Single(option => option.Value == "sys").Choose.Execute(null);
        await form.Settled;

        setup.Commit.StartSharingCommand.Execute(null);

        var started = Assert.Single(backend.Started);
        Assert.Equal("sys", started.Viewer.RenderChain);
    }

    /// <summary>
    /// The group is applied, so a choice is kept as it is made, as a settings write and never as a publish.
    /// A reader who never sends keeps what they chose.
    /// </summary>
    [Fact]
    public async Task AChoiceIsKeptAsItIsMade()
    {
        var both = await BothAsync();

        await ChooseAsync(both, "viewer.render_chain", "sys");

        Assert.Equal("sys", Assert.Single(both.Backend.Saved).Viewer.RenderChain);
        Assert.Empty(both.Backend.Started);
        Assert.False(both.Viewer.Watch.HasNotice);
    }

    /// <summary>The refusal is the backend's own sentence, and the next write that lands clears it.</summary>
    [Fact]
    public async Task ARefusedWriteShowsTheBackendsSentenceAndClearsOnTheNextOne()
    {
        var backend = new SeededBackend("linux") { SaveRefusal = "the settings file could not be written" };
        var both = await BothAsync(backend);

        await ChooseAsync(both, "viewer.render_chain", "sys");
        Assert.Equal("the settings file could not be written", both.Viewer.Watch.Notice);

        backend.SaveRefusal = "";
        await ChooseAsync(both, "viewer.render_chain", "gl");
        Assert.False(both.Viewer.Watch.HasNotice);
    }

    /// <summary>
    /// The leg a tile receives on is read through the shared draft on every pass,
    /// so a leg changed elsewhere reaches the next decode without the window being reopened.
    /// </summary>
    [Fact]
    public async Task TheTileLegIsReadThroughRatherThanReadOnce()
    {
        var both = await BothAsync();

        await ChooseAsync(both, "viewer.tile_watch_transport", "srt");

        Assert.Equal("srt", Field(both, "viewer.tile_watch_transport").Readback);
        Assert.Equal("srt", both.Form.Draft!.Viewer.TileWatchTransport);
    }

    // --- The leg a decode opens on ------------------------------------------------------

    /// <summary>A backend with one ready path, so the roster has a row to open a tile from.</summary>
    private static SeededBackend Carrying(string path)
    {
        var backend = new SeededBackend("linux");
        var relay = new RelayStatus { Reachable = true };
        relay.Paths.Add(new RelayPath { Name = path, Ready = true });
        backend.Relay = relay;
        return backend;
    }

    /// <summary>
    /// Plays one stream afresh and waits for the decode it opens.
    /// The stream started playing as the viewer opened, so it is stopped first.
    /// </summary>
    private static async Task TileAsync(Both both, string stream)
    {
        both.Viewer.Tiles.SingleOrDefault(tile => tile.Name == stream)?.Close.Execute(null);
        var row = both.Viewer.Streams.Single(entry => entry.Name == stream);
        row.Show.Execute(null);
        await both.Form.Settled;
    }

    /// <summary>A leg picked in the panel is what the next tile opens on, with nothing else pressed.</summary>
    [Fact]
    public async Task APickedLegIsWhatTheNextTileOpensOn()
    {
        var backend = Carrying("desk");
        var both = await BothAsync(backend);

        await ChooseAsync(both, "viewer.tile_watch_transport", "srt");
        await TileAsync(both, "desk");

        Assert.Equal("srt", Assert.Single(backend.Decoded).Transport);
    }

    // --- What a roster row offers -------------------------------------------------------

    /// <summary>
    /// The roster is the catalog's player list, in its order.
    /// No setting stands behind it: the leg is picked per press, so a row offers every leg a player here opens.
    /// </summary>
    [Fact]
    public async Task ARowOffersTheCatalogsPlayerLegs()
    {
        var both = await BothAsync(Carrying("desk"));
        var legs = both.Viewer.Streams.Single(row => row.Name == "desk").Legs;

        Assert.Equal(SeededBackend.PlayerLegs, legs.Select(leg => leg.Value));
    }

    /// <summary>The press on an open leg is what closes the player, so the row naming it stays on the menu.</summary>
    [Fact]
    public async Task AnOpenLegIsTickedOnTheMenu()
    {
        var backend = Carrying("desk");
        backend.Watching.Add(new StreamRef { StreamName = "desk", Transport = "hls" });
        var both = await BothAsync(backend);

        var open = both.Viewer.Streams.Single(row => row.Name == "desk").Legs.Single(leg => leg.Value == "hls");

        Assert.True(open.IsOpen);
    }

    // --- The panel, open and closed -----------------------------------------------------

    /// <summary>A choice leaves the panel open, so the reader can try the next one.</summary>
    [Fact]
    public async Task AChoiceLeavesThePanelOpen()
    {
        var both = await BothAsync();
        both.Viewer.WatchPanel.Toggle.Execute(null);

        await ChooseAsync(both, "viewer.render_chain", "sys");

        Assert.True(both.Viewer.WatchPanel.IsOpen);
    }

    /// <summary>Closing a closed panel changes nothing, so a dismissal can run it unconditionally.</summary>
    [Fact]
    public async Task ClosingAClosedPanelIsNotAnOpen()
    {
        var both = await BothAsync();
        Assert.False(both.Viewer.WatchPanel.IsOpen);

        both.Viewer.Watch.CloseCommand.Execute(null);

        Assert.False(both.Viewer.WatchPanel.IsOpen);
    }

    /// <summary>
    /// The panel puts its own group back, and the reset is kept as it is pressed.
    /// The values are the form's, stated per field, so no default is named here.
    /// </summary>
    [Fact]
    public async Task TheWatchPanelPutsItsGroupBack()
    {
        var both = await BothAsync();
        var fresh = both.Form.Draft!.Viewer.Clone();

        await ChooseAsync(both, "viewer.tile_watch_transport", "srt");
        await ChooseAsync(both, "viewer.render_chain", "sys");
        Assert.NotEqual(fresh, both.Form.Draft!.Viewer);

        Assert.True(both.Viewer.Watch.Group.HasAction);
        both.Viewer.Watch.Group.Action!.Command.Execute(null);
        await both.Form.Settled;

        Assert.Equal(fresh, both.Form.Draft!.Viewer);
        Assert.Equal(fresh, both.Backend.Saved[^1].Viewer);
    }
}
