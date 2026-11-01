using ScreenShare.App.Contracts;
using ScreenShare.App.Copy;
using TablerIcons;

namespace ScreenShare.App.Features.Viewer.Model;

/// <summary>What the join card's press says and draws for the way in that is picked.</summary>
public sealed record JoinPress(string Label, Icons Glyph);

public static class JoinPresses
{
    /// <param name="linked">Whether this machine holds a Discord link, without which no voice channel can be read.</param>
    public static JoinPress Of(JoinRoute route, bool linked) => (route, linked) switch
    {
        (JoinRoute.Discord, false) => new(Cards.JoinLinkDiscord, Icons.IconBrandDiscord),
        (JoinRoute.Discord, true) => new(Cards.JoinSetUpChannel, Icons.IconBrandDiscord),
        (JoinRoute.Key, _) => new(Cards.JoinEnterKey, Icons.IconKey),
        _ => Assert.Never<JoinPress>("every way in has a press", route),
    };
}
