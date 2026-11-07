using MirrorMe.App.Features.Shell.ViewModel;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// A panel opened over a surface, read against the width the window has
/// (<c>docs/design-language.md</c>, "Narrow windows").
/// The surface under it keeps its whole width at every size.
/// </summary>
public sealed class OverPanelTests
{
    private static OverPanelViewModel Panel() => new("Open it.", "Close it.");

    [Fact]
    public void AWideWindowFloatsTheOpenPanelBesideItsEdge()
    {
        var panel = Panel();
        panel.SetWindowWidth(1040);

        panel.Toggle.Execute(null);

        Assert.True(panel.IsOpen);
        Assert.False(panel.IsSheet);
        Assert.Equal("Close it.", panel.ToggleTip);
    }

    /// <summary>A window with no room beside the panel raises it from the foot, the crops and cards taking the full width.</summary>
    [Fact]
    public void ANarrowWindowRaisesThePanelAsASheet()
    {
        var panel = Panel();
        panel.SetWindowWidth(360);

        panel.Toggle.Execute(null);

        Assert.True(panel.IsSheet);
    }

    [Fact]
    public void ClosingAClosedPanelLeavesItClosed()
    {
        var panel = Panel();

        panel.Close();

        Assert.False(panel.IsOpen);
        Assert.Equal("Open it.", panel.ToggleTip);
    }
}
