using ScreenShare.Api.V1;
using ScreenShare.App.Copy;

namespace ScreenShare.App.Features.Viewer.Members.Model;

/// <summary>
/// What the member list says in place of rows, one sentence per cause, each naming the reader's next step.
///
/// Quiet while there are rows to show.
/// </summary>
public static class MembersEmpty
{
    public static string For(MembersState? members, DiscordState? discord, bool discordMode, int rows)
    {
        if (rows > 0)
        {
            return "";
        }

        if (members is null)
        {
            return Cards.MembersUnread;
        }

        // A linked machine in Discord mode lacks a voice channel.
        // Every other machine outside a group is asked the way in by the grid's join card.
        if (!members.Joined)
        {
            return discordMode && discord is { Linked: true } ? Cards.MembersNoChannel : Cards.MembersOutside;
        }

        return Cards.MembersNone;
    }
}
