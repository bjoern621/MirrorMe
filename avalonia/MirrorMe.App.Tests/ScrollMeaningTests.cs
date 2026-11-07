using MirrorMe.App.Controls;
using MirrorMe.App.Features.Viewer.Tile.Model;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>What a scroll over a zoomable card does, decided by what the scroll came from.</summary>
public sealed class ScrollMeaningTests
{
    [Fact]
    public void AMouseWheelZooms() => Assert.Equal(ScrollMeaning.Zoom, ScrollMeanings.Of(ScrollSource.Wheel));

    [Fact]
    public void TwoFingersOnATouchpadPan() => Assert.Equal(ScrollMeaning.Pan, ScrollMeanings.Of(ScrollSource.Finger));

    /// <summary>A pointing stick or a held scroll button moves smoothly, as fingers do.</summary>
    [Fact]
    public void AContinuousScrollPans() => Assert.Equal(ScrollMeaning.Pan, ScrollMeanings.Of(ScrollSource.Continuous));

    /// <summary>X11, Windows and macOS hand a touchpad scroll over as a wheel, which zooms as one.</summary>
    [Fact]
    public void AScrollFromAnUnnamedSourceZooms() => Assert.Equal(ScrollMeaning.Zoom, ScrollMeanings.Of(ScrollSource.Unknown));
}
