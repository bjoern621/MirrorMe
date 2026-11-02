using Avalonia.Controls;
using ScreenShare.App.Backend;
using ScreenShare.App.Features.Insights.ViewModel;
using ScreenShare.App.Features.Shell.Model;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Each destination against the width the window handed it.
/// A tiling desktop hands a window whatever the layout leaves, a quarter of the screen among them,
/// so the arrangement is read off that width on every pass rather than off the size the app opened at.
///
/// Defect locked out: a screen whose controls stand outside the window they were drawn in.
/// </summary>
public sealed class NarrowWindowTests
{
    private static readonly Action<Action> Inline = action => action();

    /// <summary>Quarter of a laptop screen, where no surface fits two columns.</summary>
    private const double Quarter = 640;

    private static InsightsViewModel Insights(IBackend backend)
    {
        var session = new Session(backend, Inline);
        return new InsightsViewModel(backend, new FormSession(backend, session, Inline), session, Inline);
    }

    /// <summary>The preview stands beside the value cards on a wide window and over them in a row on a narrow one.</summary>
    [Fact]
    public void ANarrowWindowPutsTheValueCardsInARowUnderThePreview()
    {
        var insights = Insights(new SeededBackend("linux"));

        insights.SetWindowWidth(WindowSize.Opening);
        Assert.True(insights.IsWide);
        Assert.Equal(1, insights.CardColumns);

        insights.SetWindowWidth(360);
        Assert.False(insights.IsWide);
        Assert.Equal(3, insights.CardColumns);
        Assert.Equal(1, insights.CardsRow);
    }

    [Fact]
    public void ANarrowWindowTakesTheWatchPanelOverTheGrid()
    {
        var backend = new SeededBackend("linux");
        var viewer = Flows.Viewer(backend, new Session(backend, Inline));

        viewer.SetWindowWidth(Quarter);
        viewer.WatchColumn.Toggle.Execute(null);

        Assert.True(viewer.WatchColumn.IsShowing);
        Assert.False(viewer.WatchColumn.IsBeside);
    }

    /// <summary>A width the window states twice changes nothing the second time.</summary>
    [Fact]
    public void TheSameWidthTwiceIsOnePass()
    {
        var insights = Insights(new SeededBackend("linux"));

        insights.SetWindowWidth(360);
        insights.SetWindowWidth(360);

        Assert.False(insights.IsWide);
    }
}
