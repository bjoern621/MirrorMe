using MirrorMe.Api.V1;
using MirrorMe.App.Copy;
using MirrorMe.App.Features.Viewer.Model;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// What the empty grid says, one sentence per cause: a group with nobody sharing or a linked machine outside a voice channel.
/// Outside a group the join card asks instead.
/// Quiet while the strip holds anything, while the first read is out, and while the relay notice already speaks.
/// </summary>
public sealed class GridEmptyTests
{
    private static MembersState Group(bool joined) => new() { Joined = joined };

    private static string For(
        MembersState? members, int shown, bool relayReady,
        DiscordState? discord = null, bool discordMode = false)
        => GridEmpty.For(members, discord, discordMode, shown, relayReady);

    [Fact]
    public void OutsideAGroupTheRelayNoticeStaysQuiet()
    {
        Assert.False(GridEmpty.RelayNoticeApplies(Group(joined: false)));
        Assert.True(GridEmpty.RelayNoticeApplies(Group(joined: true)));
        Assert.True(GridEmpty.RelayNoticeApplies(null));
    }

    /// <summary>Outside a group the join card asks the way in, and the line leaves the asking to it.</summary>
    [Fact]
    public void OutsideTheGroupAsksToJoin()
    {
        Assert.True(GridEmpty.AsksToJoin(Group(joined: false), null, discordMode: false, shown: 0));
        Assert.Equal("", For(Group(joined: false), shown: 0, relayReady: true));
    }

    [Fact]
    public void InDiscordModeAnUnlinkedMachineAsksToJoin()
    {
        Assert.True(GridEmpty.AsksToJoin(Group(joined: false), null, discordMode: true, shown: 0));
        Assert.Equal("", For(Group(joined: false), shown: 0, relayReady: true, discordMode: true));
    }

    /// <summary>A linked machine has picked its way in, and what is missing happens in Discord.</summary>
    [Fact]
    public void ALinkedMachineOutsideAChannelIsNotAskedAgain()
    {
        Assert.False(GridEmpty.AsksToJoin(
            Group(joined: false), new DiscordState { Linked = true }, discordMode: true, shown: 0));
    }

    [Fact]
    public void AJoinedMachineIsNotAsked()
    {
        Assert.False(GridEmpty.AsksToJoin(Group(joined: true), null, discordMode: false, shown: 0));
        Assert.False(GridEmpty.AsksToJoin(null, null, discordMode: false, shown: 0));
    }

    [Fact]
    public void InDiscordModeALinkedMachineNamesTheChannel()
    {
        Assert.Equal(
            Cards.GridNoChannel,
            For(
                Group(joined: false), shown: 0, relayReady: true,
                discord: new DiscordState { Linked = true }, discordMode: true));
    }

    [Fact]
    public void AJoinedGroupWithNothingLiveSaysSo()
    {
        Assert.Equal(Cards.GridIdle, For(Group(joined: true), shown: 0, relayReady: true));
    }

    [Fact]
    public void AnythingInTheStripSilencesTheLine()
    {
        Assert.Equal("", For(Group(joined: true), shown: 1, relayReady: true));
    }

    [Fact]
    public void AnUnansweredReadSaysNothing()
    {
        Assert.Equal("", For(members: null, shown: 0, relayReady: true));
    }

    /// <summary>An unreachable relay is the relay notice's story, and the grid does not tell it twice.</summary>
    [Fact]
    public void AnUnreadyRelaySaysNothing()
    {
        Assert.Equal("", For(Group(joined: true), shown: 0, relayReady: false));
    }
}
