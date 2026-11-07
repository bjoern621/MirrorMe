using MirrorMe.App.Features.Shell.Model;
using MirrorMe.App.Features.Shell.NavStrip.ViewModel;
using MirrorMe.App.Features.Shell.ViewModel;
using Xunit;

namespace MirrorMe.App.Tests;

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

    /// <summary>The watch settings belong to the viewer, so the strip offers them on the viewer alone.</summary>
    [Fact]
    public void TheWatchSettingsOpenFromTheStripOnTheViewer()
    {
        var watch = new OverPanelViewModel("open", "close");
        var strip = new NavStripViewModel(_ => { }, watch: watch);

        strip.Show(Destination.Setup);
        Assert.False(strip.ShowsWatch);

        strip.Show(Destination.Viewer);
        Assert.True(strip.ShowsWatch);
        Assert.Same(watch, strip.Watch);
    }
}
