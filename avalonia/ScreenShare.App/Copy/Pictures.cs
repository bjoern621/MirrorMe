namespace ScreenShare.App.Copy;

/// <summary>
/// Heading and sentence over a field's comparison pictures, keyed by field.
/// The sentence names where in the picture the entries differ, so the eye lands there first.
/// </summary>
public static class Pictures
{
    public sealed record Entry(string Title, string Sentence);

    /// <summary>Closes the pictures where a narrow window shows them as a sheet.</summary>
    public const string Done = "Done";

    private static readonly Dictionary<string, Entry> Entries = new()
    {
        ["publish.cq"] = new(
            "How much detail",
            "Same picture at each step. Blocks appear around text first, then across the gradient."),
        ["publish.chroma"] = new(
            "Color detail",
            "Look at the red and blue text. Standard color smears its edges."),
        ["publish.output_resolution"] = new(
            "How large",
            "Same picture at each size, scaled back up. Fine text blurs first."),
    };

    /// <summary>Copy for one field's pictures, null for a field with none written.</summary>
    public static Entry? Of(string key) => Entries.GetValueOrDefault(key);
}
