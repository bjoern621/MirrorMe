namespace MirrorMe.App.Features.Viewer.WatchSettings.Model;

/// <summary>
/// Where the watch panel places its controls.
/// Over the fold stands one card per way a stream can arrive (<c>Copy/Arrivals.cs</c>), each writing a protocol.
/// Every control waits in the fold, the protocol among them under its own name.
/// </summary>
public static class WatchLayout
{
    /// <summary>Field the cards write.</summary>
    public const string ProtocolKey = "viewer.tile_watch_transport";

    public static bool OnFloor(string key) => false;
}
