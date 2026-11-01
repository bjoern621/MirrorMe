using TablerIcons;

namespace ScreenShare.App.Copy;

/// <summary>
/// Mark an option card draws before its title, keyed by the field and the value the backend sent.
/// Most entries carry none: a mark earns its place where the two answers differ in kind at a glance.
/// </summary>
public static class Glyphs
{
    private static readonly Dictionary<(string Field, string Value), Icons> Entries = new()
    {
        [("relay.group_source", "discord")] = Icons.IconBrandDiscord,
        [("relay.group_source", "key")] = Icons.IconKey,
    };

    /// <summary>The entry's mark, null for an entry that draws none.</summary>
    public static Icons? Of(string field, string value)
        => Entries.TryGetValue((field, value), out var glyph) ? glyph : null;
}
