using Avalonia;
using Avalonia.Controls;
using ScreenShare.App.Controls;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A fold's body takes the height that the fold's openness gives it.
/// Off the visual tree no transition runs, so the openness lands on the state asked for.
/// </summary>
public sealed class RevealTests
{
    private static Reveal Holding(double bodyHeight) => new() { Child = new Border { Width = 50, Height = bodyHeight } };

    private static double MeasuredHeight(Reveal reveal)
    {
        reveal.Measure(new Size(200, double.PositiveInfinity));
        return reveal.DesiredSize.Height;
    }

    [Fact]
    public void AClosedFoldTakesNoHeight()
    {
        var reveal = Holding(80);

        Assert.Equal(0, MeasuredHeight(reveal));
    }

    [Fact]
    public void AnOpenFoldTakesItsBodysHeight()
    {
        var reveal = Holding(80);
        reveal.IsOpen = true;

        Assert.Equal(80, MeasuredHeight(reveal));
    }

    [Fact]
    public void AFoldHalfwayOpenTakesHalfItsBodysHeight()
    {
        var reveal = Holding(80);
        reveal.IsOpen = true;
        reveal.Openness = 0.5;

        Assert.Equal(40, MeasuredHeight(reveal));
    }

    [Fact]
    public void AFoldClosingAgainTakesNoHeight()
    {
        var reveal = Holding(80);
        reveal.IsOpen = true;
        MeasuredHeight(reveal);
        reveal.IsOpen = false;

        Assert.Equal(0, MeasuredHeight(reveal));
    }
}

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
