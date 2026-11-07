using MirrorMe.Api.V1;

namespace MirrorMe.App.Features.Shell.Model;

/// <summary>
/// Where the window opens.
///
/// A stream on the air outranks the viewer's first paint: the reader launching over a live stream is
/// its publisher, and the live surface is what they are coming back for.
/// Everything else opens on the viewer, the one screen the design states end to end.
/// </summary>
public static class Opening
{
    public static Destination For(PublishState? publish)
        => publish?.Live is not null ? Destination.Insights : Destination.Viewer;
}
