using ScreenShare.Api.V1;
using ScreenShare.App.Features.Viewer.Tile.Model;
using ScreenShare.App.Features.Viewer.Tile.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>The figure in a tile's lower right: the picture's height and rate as the decode reads them.</summary>
public sealed class TileFigureTests
{
    private static TileViewModel Tile()
        => new(TileSource.Relay("desk", "rtsp"), new SeededBackend("linux"), static action => action(), static _ => { });

    private static TilePipeline Live => new(
        Live: true,
        HasAudio: false, Volume: 1, Muted: false,
        Transfer: "bt709", Hdr: false,
        ToneMap: false, CanToneMap: false, ToneMapMissing: "");

    [Fact]
    public void ATileStatesThePictureHeightAndTheRoundedRate()
    {
        var tile = Tile();

        tile.Apply(Live, new ReceiveStreamStats { Width = 1920, Height = 1080, VideoFps = 59.94 });

        Assert.Equal("1080p · 60 fps", tile.Figure);
        Assert.True(tile.HasFigure);
    }

    [Fact]
    public void ARateNotYetMeasuredLeavesTheHeightAlone()
    {
        var tile = Tile();

        tile.Apply(Live, new ReceiveStreamStats { Width = 1280, Height = 720 });

        Assert.Equal("720p", tile.Figure);
    }

    /// <summary>A decode that has drawn nothing has no picture to measure.</summary>
    [Fact]
    public void ADecodeWithNoPictureYetStatesNoFigure()
    {
        var tile = Tile();

        tile.Apply(Live, new ReceiveStreamStats { VideoFps = 60 });

        Assert.Equal("", tile.Figure);
        Assert.False(tile.HasFigure);
    }

    [Fact]
    public void ATileWithNoSampleStatesNoFigure()
    {
        var tile = Tile();

        tile.Apply(Live, sample: null);

        Assert.False(tile.HasFigure);
    }
}
