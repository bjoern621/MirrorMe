using ScreenShare.App.Contracts;

namespace ScreenShare.App.Features.Viewer.Tile.Model;

/// <summary>What a scroll came from, in the terms of Wayland's <c>wl_pointer.axis_source</c>.</summary>
public enum ScrollSource
{
    /// <summary>No backend said: X11, Windows and macOS name none.</summary>
    Unknown,
    Wheel,
    Finger,
    Continuous,
    WheelTilt,
}

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
