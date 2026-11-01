using ScreenShare.Api.V1;
using ScreenShare.App.Copy;

namespace ScreenShare.App.Features.Viewer.Model;

/// <summary>
/// What the empty grid says, one sentence per cause, each naming the reader's next step.
///
/// Quiet while a tile is up, while the first presence read is out,
/// while the relay's own notice on the rail already speaks, and while the join card asks the way in:
/// an empty grid is not told twice why.
/// </summary>
public static class GridEmpty
{
    /// <summary>
    /// Whether the grid asks how to join a group.
    /// A linked machine in Discord mode has picked its way in, and what it lacks is a voice channel.
    /// </summary>
    public static bool AsksToJoin(MembersState? members, DiscordState? discord, bool discordMode, int tiles)
        => tiles == 0 && members is { Joined: false } && !(discordMode && discord is { Linked: true });

    public static string For(
        MembersState? members, DiscordState? discord, bool discordMode,
        int streams, int tiles, bool relayReady)
    {
        if (tiles > 0 || members is null || AsksToJoin(members, discord, discordMode, tiles))
        {
            return "";
        }

        if (!members.Joined)
        {
            return Cards.GridNoChannel;
        }

        if (streams > 0)
        {
            return Cards.GridUnwatched;
        }

        return relayReady ? Cards.GridIdle : "";
    }
}
