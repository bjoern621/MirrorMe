using ScreenShare.Api.V1;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Viewer.Model;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// What the empty grid says, one sentence per cause: a group with nobody sharing, streams waiting unwatched,
/// or a linked machine outside a voice channel. Outside a group the join card asks instead.
/// Quiet while a tile is up, while the first read is out, and while the rail's notice already speaks.
/// </summary>
public sealed class GridEmptyTests
{
    private static MembersState Group(bool joined) => new() { Joined = joined };

    private static string For(
        MembersState? members, int streams, int tiles, bool relayReady,
        DiscordState? discord = null, bool discordMode = false)
        => GridEmpty.For(members, discord, discordMode, streams, tiles, relayReady);

    /// <summary>Outside a group the join card asks the way in, and the line leaves the asking to it.</summary>
    [Fact]
    public void OutsideTheGroupAsksToJoin()
    {
        Assert.True(GridEmpty.AsksToJoin(Group(joined: false), null, discordMode: false, tiles: 0));
        Assert.Equal("", For(Group(joined: false), streams: 0, tiles: 0, relayReady: true));
    }

    [Fact]
    public void InDiscordModeAnUnlinkedMachineAsksToJoin()
    {
        Assert.True(GridEmpty.AsksToJoin(Group(joined: false), null, discordMode: true, tiles: 0));
        Assert.Equal("", For(Group(joined: false), streams: 0, tiles: 0, relayReady: true, discordMode: true));
    }

    /// <summary>A linked machine has picked its way in, and what is missing happens in Discord.</summary>
    [Fact]
    public void ALinkedMachineOutsideAChannelIsNotAskedAgain()
    {
        Assert.False(GridEmpty.AsksToJoin(
            Group(joined: false), new DiscordState { Linked = true }, discordMode: true, tiles: 0));
    }

    [Fact]
    public void AJoinedMachineIsNotAsked()
    {
        Assert.False(GridEmpty.AsksToJoin(Group(joined: true), null, discordMode: false, tiles: 0));
        Assert.False(GridEmpty.AsksToJoin(null, null, discordMode: false, tiles: 0));
    }

    [Fact]
    public void InDiscordModeALinkedMachineNamesTheChannel()
    {
        Assert.Equal(
            Cards.GridNoChannel,
            For(
                Group(joined: false), streams: 0, tiles: 0, relayReady: true,
                discord: new DiscordState { Linked = true }, discordMode: true));
    }

    [Fact]
    public void AJoinedGroupWithNothingLiveSaysSo()
    {
        Assert.Equal(Cards.GridIdle, For(Group(joined: true), streams: 0, tiles: 0, relayReady: true));
    }

    [Fact]
    public void WaitingStreamsPointAtTheList()
    {
        Assert.Equal(Cards.GridUnwatched, For(Group(joined: true), streams: 2, tiles: 0, relayReady: true));
    }

    [Fact]
    public void ATileOnScreenSilencesTheLine()
    {
        Assert.Equal("", For(Group(joined: true), streams: 2, tiles: 1, relayReady: true));
    }

    [Fact]
    public void AnUnansweredReadSaysNothing()
    {
        Assert.Equal("", For(members: null, streams: 0, tiles: 0, relayReady: true));
    }

    /// <summary>An unreachable relay is the rail notice's story, and the grid does not tell it twice.</summary>
    [Fact]
    public void AnUnreadyRelaySaysNothing()
    {
        Assert.Equal("", For(Group(joined: true), streams: 0, tiles: 0, relayReady: false));
    }
}
