using ScreenShare.App.Contracts;

namespace ScreenShare.App.Controls;

/// <summary>
/// How many of a question's choices sit across one line (<see cref="ChoiceRowPanel"/>).
/// All of them where each keeps its readable width, as many as fit where not, the rest wrapping under them.
/// </summary>
public static class ChoiceRowLayout
{
    /// <summary>px. Narrowest a choice is set at, its name and its figure each on one line.</summary>
    public const double MinChoiceWidth = 168;

    /// <summary>px. Between two choices, across and down: the tight gap of <c>Design/Metrics.axaml</c>.</summary>
    public const double Gap = 7;

    public static int Columns(double width, int count)
    {
        Assert.That(width >= 0, "a row is laid out across a width", width);
        Assert.That(count >= 0, "a row is laid out for a count of choices", count);

        var fit = (int)Math.Floor((width + Gap) / (MinChoiceWidth + Gap));
        var columns = Math.Clamp(Math.Min(count, fit), 1, Math.Max(1, count));

        Assert.That(columns >= 1, "a row holds at least one column", columns);
        return columns;
    }
}
