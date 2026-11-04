namespace ScreenShare.App.Copy;

/// <summary>
/// What a card says in its own right: what it is, and what it shows in place of the thing it usually draws.
///
/// The other tables here are keyed on what the backend sent:
/// an identifier (<see cref="Words"/>), a choice (<see cref="Descriptions"/>), a field (<see cref="Fields"/>),
/// a statement (<see cref="Statements"/>).
/// These are keyed on nothing.
/// Here rather than in markup or spelled twice at two render functions (<c>avalonia/README.md</c>, "Layout").
///
/// A refusal takes its sentence from <see cref="Refusals"/> (<c>docs/ipc-api.md</c>, "Errors").
/// These cover the states no call failed in.
/// </summary>
public static class Cards
{
    /// <summary>
    /// Names where the picture is taken, the whole difference between the two routes:
    /// both carry the same encode (<see cref="Features.Insights.Preview.Model.PreviewRoute"/>).
    /// The off segment takes none, and the question still reads over it.
    /// </summary>
    public const string PreviewRouteChoice = "Where this picture is taken:";

    /// <summary>Nowhere: draws no picture, holds no decode open.</summary>
    public const string PreviewOffLabel = "Off";

    public const string PreviewLocalLabel = "Local";

    /// <summary>Spelled out: "E2E" is a word for whoever built the pipeline.</summary>
    public const string PreviewEndToEndLabel = "End to end";

    /// <summary>
    /// What the off segment costs:
    /// the end-to-end route's reader slot and downstream bandwidth go back with the picture.
    /// </summary>
    public const string PreviewOffCost =
        "No picture is drawn and no decode is held open. The stream is unaffected. Viewers keep "
        + "receiving it while this card is off.";

    /// <summary>
    /// Local route's cost, and its warning.
    /// The picture is taken before the relay,
    /// so a congested uplink, a dropping relay and a bad viewer link all leave this card looking perfect
    /// (<c>docs/viewer-architecture.md</c>, "What the insights preview draws").
    /// Opposite claim to <see cref="PreviewEndToEndCost"/>, so neither stands in for the other.
    /// </summary>
    public const string PreviewLocalCost =
        "This is what is being sent, decoded on this computer from a copy the encoder makes. It "
        + "never reaches the relay, costs one local decode and no bandwidth, and adds no viewer "
        + "to the counts. It shows nothing about what viewers receive. Switch to end to end for "
        + "that, or read the Viewers tab.";

    /// <summary>
    /// End-to-end route's cost: decode opened with <c>Set receive</c> like any tile,
    /// so the relay counts it among the viewers beside this card and it pays a viewer's downstream bandwidth.
    /// </summary>
    public const string PreviewEndToEndCost =
        "This is what a viewer receives: this computer's own stream, pulled back off the relay. "
        + "It crosses the uplink, the relay, and the way back, so it shows the whole path. It "
        + "pays for that as a viewer does, spending downstream bandwidth and counting as one of "
        + "the viewers beside it.";

    public const string PreviewNotPublishing = "Nothing is publishing, so there is nothing to show.";

    /// <summary>
    /// Leads with the stream being untouched: a dark card is what a reader could take for a stopped stream.
    /// Names the picture and not the stream, the header's red control being what stops the stream.
    /// </summary>
    public const string PreviewOff = "The preview is off. The stream is unaffected.";

    /// <summary>
    /// A screen the backend was asked to read and has produced no picture from.
    /// A capture element takes a moment to start, and a tile with nothing in it and nothing to say reads as broken.
    /// </summary>
    public const string ScreenOpening = "Opening.";

    /// <summary>
    /// Leads with nothing being shared: in an app whose purpose is sending a screen,
    /// a live picture of one is what a reader could take for a stream already live.
    /// The picture is the rectangle the stream would carry, read by the same element
    /// (<c>backend/internal/screensrc</c>).
    /// </summary>
    public const string ScreenPickerCost =
        "Nothing is being shared yet. Each picture shows what that screen displays now, read the "
        + "same way the stream would read it. The one picked is what viewers would see.";

    /// <summary>
    /// A stream live with no preview behind it.
    /// The reason is in the backend's log rather than on the contract, so the sentence names the fact alone.
    /// </summary>
    public const string PreviewNotPreviewed = "This stream is live and not being previewed locally.";

    /// <summary>
    /// No leg to receive on.
    /// A decode is keyed by stream and protocol together,
    /// so there is nothing to ask for until the settings name one (<c>Features/Viewer/Tile/Model/TileLeg.cs</c>).
    /// </summary>
    public const string PreviewNoWatchLeg =
        "The settings name no protocol for a viewer to receive on. Pick one under Watching.";

    /// <summary>
    /// Its own state, apart from the tile's "Connecting.":
    /// that one is a pipeline up with no frame out of it, this the round trip before there is a pipeline at all.
    /// </summary>
    public const string PreviewOpening = "Opening a decode of this stream off the relay.";

    /// <summary>
    /// What a preset covers, said where the reader is about to make one.
    /// The relay and the watch settings are outside it, one a deployment's and the other this machine's drivers
    /// (<c>docs/presets.md</c>).
    /// </summary>
    public const string PresetsCovers =
        "A preset is one way of publishing: the source, the quality, the audio, and the transport. "
        + "The relay and watch settings stay as they are. They belong to this computer rather "
        + "than to what it sends.";

    /// <summary>
    /// What the viewer table says in place of rows.
    /// One sentence per absence, three of them leaving the publisher something to do next:
    /// start a stream, wait for the relay to be asked, or send somebody the link.
    /// The fourth has no next step: the relay counts viewers whose app sends no figures.
    /// </summary>
    public const string ViewersIdle = "Nothing is publishing, so nobody can be watching this computer.";

    public const string ViewersUnasked = "The relay has not been asked yet, so there is nobody to list.";

    public const string ViewersNone = "Nobody is connected to this stream yet.";

    public const string ViewersUnreported =
        "No viewer has sent figures yet. A browser or an external player never sends any.";

    /// <summary>
    /// What the wizard says before the backend has described this machine.
    /// The steps are drawn out of that answer, so this is what a launch shows in their place,
    /// worded as the relay reading is.
    /// </summary>
    public const string SetupUnread = "Reading what this computer can send.";

    // --- The grid --------------------------------------------------------------------

    /// <summary>
    /// What the empty grid says, one sentence per cause and each naming the next step.
    /// Quiet where the relay notice already speaks (<c>Features/Viewer/Model/GridEmpty.cs</c>).
    /// </summary>
    public const string GridIdle = "Nobody in the group is sharing a screen.";

    public const string GridNoChannel =
        "Not in a voice channel. Join one in Discord, and every member's stream shows here.";

    /// <summary>A member sharing nothing, in the strip where their stream would stand.</summary>
    public static string NotSharing(string name) => $"{name} is in the group, not sharing";

    public const string GridSwitch = "Grid";

    public const string GridSwitchTip = "Show every stream at the same size";

    /// <summary>On a stream the reader stopped, which waits in the strip.</summary>
    public const string PlayAgainTip = "Stopped. Click to play it again.";

    public const string StopWatching = "Stop watching";

    /// <summary>Under the stats overlay's greyed menu row, on a tile too narrow for the panel.</summary>
    public const string StatsNoRoom = "This tile is too narrow for it. Widen the window to turn it on.";

    public const string HideStrip = "Hide other streams";

    public const string HideStripTip = "Hide the other streams (H)";

    public const string ShowStripTip = "Show the other streams (H)";

    public const string JoinTitle = "Join a group to watch and share";

    public const string JoinBody = "Everyone in a group sees the others' streams.";

    public const string JoinDiscord = "Use my Discord voice channel";

    public const string JoinDiscordDetail = "The group follows the channel you sit in";

    public const string JoinKey = "Paste a group key";

    public const string JoinKeyDetail = "For a group outside Discord";

    public const string JoinLinkDiscord = "Link Discord";

    public const string JoinSetUpChannel = "Set up my voice channel";

    public const string JoinEnterKey = "Enter a group key";

    public const string Recommended = "Recommended";

    /// <summary>Card at the step column's foot, holding what no control on screen already says.</summary>
    public const string ChecksTitle = "Checks before sharing";

    /// <summary>Disclosure on that card over the lines that report nothing wrong.</summary>
    public const string ChecksNotes = "Notes";

    /// <summary>Beside the predicted rate on the cost bar.</summary>
    public static string OfUplink(string uplink) => $"of {uplink} uplink";

    public const string NoUplink = "No uplink stated";

    /// <summary>
    /// The relay check: its heading, the button that runs it, what the button does,
    /// and the line standing where the results go before one has run.
    ///
    /// The heading names what is reached rather than what is run,
    /// the reader having come to this card from the address and the ports above it.
    /// The unchecked-relay line keeps the card from reading as broken before a check runs.
    /// </summary>
    public const string RelayCheckTitle = "What answers on this relay";

    public const string RelayCheckAction = "Check";

    public const string RelayCheckHint =
        "Opens a connection to each of the relay's listeners and says which answered. "
        + "Sends no stream and changes nothing.";

    public const string RelayCheckUnrun = "No check has run on this relay.";

    /// <summary>Encode step's first question, answered by the built-in presets.</summary>
    public const string PresetQuestion = "How should the picture look?";

    /// <summary>Disclosure over the saved presets and the box that saves the draft as one.</summary>
    public const string SavedPresetsTitle = "Saved presets";

    /// <summary>Question the encode step asks of one control, keyed by its settings field.</summary>
    public static string Question(string key) => key switch
    {
        "publish.fps" => "How smooth should motion be?",
        "publish.output_resolution" => "How large?",
        "publish.cq" => "How much detail?",
        _ => Fields.Of(key).Label,
    };

    /// <summary>
    /// What the turning arc beside an unreachable backend means: waiting is enough, nothing has to be restarted.
    /// Shared by every screen drawing the arc, so it names no button only one of them offers.
    /// </summary>
    public const string Redialling =
        "This window is dialing the backend, and keeps doing so until it answers or the window closes.";

    /// <summary>
    /// What the button over the audio list says.
    /// It carries the form's own row past the end of the list, whose value is the absent kind,
    /// so the face names what pressing it does rather than what the row holds
    /// (<see cref="Features.Setup.AudioStep.ViewModel.AudioStepViewModel.Add"/>).
    /// </summary>
    public const string AudioAdd = "Add a source";

    /// <summary>What a stream with an empty audio list sends, in place of the rows.</summary>
    public const string AudioEmpty = "Nothing is listed, so the stream goes out silent.";

    /// <summary>
    /// Which of the list's controls reach a running stream,
    /// said once over the columns rather than beside every entry's control.
    /// The controls are named off the form, <c>Field.live</c> deciding which are in the sentence at all:
    /// the engine behind the capture backend answers it, and it moves with the settings
    /// (<c>docs/field-availability.md</c>, "A live stream blocks no field").
    /// </summary>
    public static string AudioLive(string control) =>
        $"{control} reaches a running stream, so moving it costs no viewer a reconnect.";

    public static string AudioLive(string first, string second) =>
        $"{first} and {second} reach a running stream, so moving either costs no viewer a reconnect.";

    /// <summary>
    /// An empty store, apart from <see cref="Statements"/>' unreadable-store notice:
    /// that one is the store failing, this one a reader who has saved nothing.
    /// </summary>
    public const string PresetsEmpty = "Nothing saved yet. Name the configuration below to keep it.";

    /// <summary>
    /// Why the list can be behind: presets are a file the backend does not run on, so no event says one appeared
    /// (<c>Backend/IBackend.cs</c>, <c>PresetsAsync</c>).
    /// </summary>
    public const string PresetsReread = "Read the saved presets again. Nothing announces a preset another window saved.";
}
