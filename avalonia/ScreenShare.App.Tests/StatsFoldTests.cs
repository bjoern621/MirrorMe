using System.Collections.ObjectModel;
using ScreenShare.Api.V1;
using ScreenShare.App.Features.Viewer.Tile.Model;
using ScreenShare.App.Features.Viewer.Tile.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>The stats panel shows one stage at a time: every block closed until a press opens one.</summary>
public sealed class StatsFoldTests
{
    private static ReceiveStreamStats Sample() => new()
    {
        CodecDescription = "H.264",
        VideoMbps = 8,
        Width = 1920,
        Height = 1080,
        Decoder = "vah264dec",
        AudioCodecDescription = "Opus",
        AudioRate = 48_000,
    };

    private static TileReport Report() => new(1920, 1080, Frames: 600, Dropped: 0, Notice: "");

    private static (ObservableCollection<StatSection> Panel, StatsFold Fold) Open(ReceiveStreamStats sample)
    {
        var panel = new ObservableCollection<StatSection>();
        var fold = new StatsFold(panel);
        TileStats.Merge(panel, TileStats.Of(sample, Report()));
        fold.Apply();
        return (panel, fold);
    }

    [Fact]
    public void EveryBlockStartsClosed()
    {
        var (panel, _) = Open(Sample());

        Assert.NotEmpty(panel);
        Assert.All(panel, section => Assert.False(section.IsOpen));
    }

    [Fact]
    public void OpeningABlockClosesTheOneOpenBeforeIt()
    {
        var (panel, _) = Open(Sample());

        panel[0].Toggle!.Execute(null);
        Assert.Equal([panel[0]], panel.Where(section => section.IsOpen));

        panel[1].Toggle!.Execute(null);
        Assert.Equal([panel[1]], panel.Where(section => section.IsOpen));

        panel[1].Toggle!.Execute(null);
        Assert.DoesNotContain(panel, section => section.IsOpen);
    }

    /// <summary>A block that returns comes back closed, the press that opened it belonging to a stage that went away.</summary>
    [Fact]
    public void ABlockThatLeavesComesBackClosed()
    {
        var (panel, fold) = Open(Sample());
        panel.Single(section => section.Heading == "Audio").Toggle!.Execute(null);

        var silent = Sample();
        silent.AudioCodecDescription = "";
        silent.AudioRate = 0;
        TileStats.Merge(panel, TileStats.Of(silent, Report()));
        fold.Apply();
        TileStats.Merge(panel, TileStats.Of(Sample(), Report()));
        fold.Apply();

        Assert.DoesNotContain(panel, section => section.IsOpen);
    }

    /// <summary>A new reading lands in the open block without closing it.</summary>
    [Fact]
    public void AnOpenBlockStaysOpenThroughTheNextSample()
    {
        var (panel, fold) = Open(Sample());
        panel[0].Toggle!.Execute(null);

        var moved = Sample();
        moved.VideoMbps = 9;
        TileStats.Merge(panel, TileStats.Of(moved, Report()));
        fold.Apply();

        Assert.True(panel[0].IsOpen);
    }
}
