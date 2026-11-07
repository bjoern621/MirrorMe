namespace MirrorMe.App.Controls;

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
