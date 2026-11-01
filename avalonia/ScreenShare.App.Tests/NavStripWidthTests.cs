using ScreenShare.App.Features.Shell.Model;
using ScreenShare.App.Features.Shell.NavStrip.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Strip names each destination with a mark and a word, and drops the words where the window cannot hold them.
/// </summary>
public sealed class NavStripWidthTests
{
    private static NavStripViewModel Strip() => new(_ => { });

    [Fact]
    public void EveryDestinationCarriesItsOwnMark()
    {
        var marks = Strip().Tabs.Select(tab => tab.Icon).ToList();

        Assert.Equal(marks.Count, marks.Distinct().Count());
    }

    [Fact]
    public void AWideWindowSpellsOutTheDestinations()
    {
        var strip = Strip();

        strip.SetWindowWidth(NavWidth.LabelsMinWidth);

        Assert.True(strip.ShowsLabels);
    }

    [Fact]
    public void ANarrowWindowKeepsTheMarksAlone()
    {
        var strip = Strip();

        strip.SetWindowWidth(NavWidth.LabelsMinWidth - 1);

        Assert.False(strip.ShowsLabels);
    }
}
