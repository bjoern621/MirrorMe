using MirrorMe.App.Controls;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// A fold's body opens through a <see cref="Reveal"/>, which unrolls it.
/// Bound to <c>IsVisible</c> it would appear in one frame, the rows under it jumping down.
/// Fold state ends in <c>Shown</c> on every view model holding one, <c>IsShown</c> naming a card that rises on its own.
/// </summary>
public sealed class FoldBodyTests
{
    private static readonly System.Text.RegularExpressions.Regex Snapping =
        new(@"IsVisible=""\{Binding [^}]*(?<!Is)Shown\}""", System.Text.RegularExpressions.RegexOptions.Compiled);

    [Fact]
    public void EveryFoldBodyUnrollsThroughAReveal()
    {
        var snapping = Markup.Views()
            .SelectMany(view => Snapping.Matches(File.ReadAllText(view)).Select(match => $"{Path.GetFileName(view)}: {match.Value}"))
            .ToList();

        Assert.True(snapping.Count == 0, $"a fold body binds Reveal.IsOpen, never IsVisible: {string.Join(", ", snapping)}");
    }

    private static readonly System.Text.RegularExpressions.Regex Negated =
        new(@"<controls:Reveal(?![^>]*FollowsOpening=""False"")[^>]*IsOpen=""\{Binding ![^}]*\}""", System.Text.RegularExpressions.RegexOptions.Compiled);

    /// <summary>Opening one encode row closes another, whose heading lines then come back wherever that row stands.</summary>
    [Fact]
    public void ARevealThatOpensAsItsFoldClosesLeavesTheScrollAlone()
    {
        var following = Markup.Views()
            .SelectMany(view => Negated.Matches(File.ReadAllText(view)).Select(match => $"{Path.GetFileName(view)}: {match.Value}"))
            .ToList();

        Assert.True(following.Count == 0, $"a reveal bound to a closed state sets FollowsOpening=\"False\": {string.Join(", ", following)}");
    }
}
