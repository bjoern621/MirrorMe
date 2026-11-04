using System.Text.RegularExpressions;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A control's help and a card's paragraph open on hover, so a screen shows names, figures and chips.
/// A step's lead under its question stays on screen, the one sentence a step opens on.
/// </summary>
public sealed class HelpOnHoverTests
{
    private static readonly Regex Printed =
        new(@"<TextBlock\b[^>]*\bText=""\{Binding (Help|Group\.Help|Detail)\}""[^>]*>", RegexOptions.Compiled);

    private static readonly string[] StepLeads = ["FieldGroupView.axaml", "AudioStepView.axaml"];

    [Fact]
    public void HelpAndCardParagraphsStandBehindAHover()
    {
        var printed = Markup.Views()
            .SelectMany(view => Printed.Matches(File.ReadAllText(view))
                .Where(match => !(StepLeads.Contains(Path.GetFileName(view)) && match.Value.Contains("Classes=\"secondary\"")))
                .Select(match => $"{Path.GetFileName(view)}: {match.Value}"))
            .ToList();

        Assert.True(printed.Count == 0, $"help and card paragraphs open on hover, never as a line: {string.Join(", ", printed)}");
    }
}
