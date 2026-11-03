using System.Text.RegularExpressions;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// TablerIcon registers Brush as two-way.
/// An icon reading its host's Foreground two-way writes every frame of its own colour fade back onto the host
/// as a local value, which outranks every style: a hover, a pick or a greyed state never reaches the label again.
/// </summary>
public sealed class IconBrushBindingTests
{
    private static readonly Regex TwoWayHostRead =
        new(@"Brush=""\{Binding \$parent\[[^\]]+\]\.Foreground(?![^}]*Mode=OneWay)[^}]*\}""", RegexOptions.Compiled);

    [Fact]
    public void EveryIconReadsItsHostForegroundOneWay()
    {
        var writing = Markup.Views()
            .SelectMany(view => TwoWayHostRead.Matches(File.ReadAllText(view)).Select(match => $"{Path.GetFileName(view)}: {match.Value}"))
            .ToList();

        Assert.True(writing.Count == 0, $"an icon reads its host's Foreground with Mode=OneWay: {string.Join(", ", writing)}");
    }
}
