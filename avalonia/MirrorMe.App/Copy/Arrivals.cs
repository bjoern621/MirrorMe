namespace MirrorMe.App.Copy;

/// <summary>
/// Ways a stream can arrive, as the watch panel's cards name them, each standing for one watch protocol.
/// Card order. A protocol absent here gets picked in the panel's fold, under its own name.
/// </summary>
public static class Arrivals
{
    /// <param name="Value">Watch protocol the card writes, as the form offers it.</param>
    public sealed record Arrival(string Value, string Name, string Sentence);

    public static readonly IReadOnlyList<Arrival> All =
    [
        new("rtsp", "Low delay", "Adds the least delay and plays every format."),
        new("srt", "Steady", "Asks again for lost packets, at the cost of a short wait."),
        new("hls", "Compatible", "Gets through proxies and firewalls, and runs seconds behind."),
    ];
}
