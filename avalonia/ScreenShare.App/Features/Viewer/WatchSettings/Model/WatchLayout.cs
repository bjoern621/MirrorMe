namespace ScreenShare.App.Features.Viewer.WatchSettings.Model;

/// <summary>
/// Which control the watch panel shows while its fold is closed: the protocol a tile receives on.
/// Every other field of the group has a default behind it and waits in the fold.
/// </summary>
public static class WatchLayout
{
    public const string FloorKey = "viewer.tile_watch_transport";

    public static bool OnFloor(string key) => key == FloorKey;
}
