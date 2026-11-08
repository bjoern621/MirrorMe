using MirrorMe.Api.V1;
using MirrorMe.App.Copy;

namespace MirrorMe.App.Features.Viewer.Model;

/// <summary>
/// What the empty stage says, one sentence per cause, each naming the reader's next step.
///
/// Quiet while the strip holds anything, while the first presence read is out,
/// while the relay's own notice already speaks, and while the join card asks the way in:
/// an empty stage is not told twice why.
/// </summary>
public static class GridEmpty
{
    /// <summary>
    /// Whether the stage asks how to join a group.
    /// A linked machine in Discord mode has picked its way in, and what it lacks is a voice channel.
    /// </summary>
    public static bool AsksToJoin(MembersState? members, DiscordState? discord, bool discordMode, int shown)
        => shown == 0 && members is { Joined: false } && !(discordMode && discord is { Linked: true });

    /// <summary>
    /// Whether a failed relay reading says anything about the relay.
    /// A machine outside a group has no list to read, so its reading fails with the relay up.
    /// </summary>
    public static bool RelayNoticeApplies(MembersState? members) => members is not { Joined: false };

    /// <param name="shown">Items in the strip: tiles, stopped streams, members sharing nothing.</param>
    public static string For(
        MembersState? members, DiscordState? discord, bool discordMode, int shown, bool relayReady)
    {
        if (shown > 0 || members is null || AsksToJoin(members, discord, discordMode, shown))
        {
            return "";
        }

        if (!members.Joined)
        {
            return Cards.GridNoChannel;
        }

        return relayReady ? Cards.GridIdle : "";
    }
}
