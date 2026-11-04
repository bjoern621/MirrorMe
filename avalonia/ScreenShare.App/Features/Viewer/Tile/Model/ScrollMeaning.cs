using ScreenShare.App.Contracts;
using ScreenShare.App.Controls;

namespace ScreenShare.App.Features.Viewer.Tile.Model;

public enum ScrollMeaning
{
    Zoom,
    Pan,
}

public static class ScrollMeanings
{
    public static ScrollMeaning Of(ScrollSource source) => source switch
    {
        ScrollSource.Unknown or ScrollSource.Wheel or ScrollSource.WheelTilt => ScrollMeaning.Zoom,
        ScrollSource.Finger or ScrollSource.Continuous => ScrollMeaning.Pan,
        _ => Assert.Never<ScrollMeaning>("unexpected scroll source", (int)source),
    };
}
