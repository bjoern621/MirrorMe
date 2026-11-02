using System.Runtime.CompilerServices;
using ScreenShare.App.Features.Fields.Model;
using ScreenShare.App.Features.Fields.ViewModel;
using ScreenShare.App.Features.Setup.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Comparison crops: pictures rendered offline through the real encoder (<c>tools/crops.sh</c>),
/// listed in one manifest that the script renders from and the shell reads.
/// </summary>
public sealed class ComparisonTests
{
    private static string CropsDirectory([CallerFilePath] string here = "")
        => Path.Combine(Path.GetDirectoryName(here)!, "..", "ScreenShare.App", "Assets", "Crops");

    [Fact]
    public void EveryCropTheManifestListsIsRendered()
    {
        foreach (var key in new[] { "publish.cq", "publish.chroma", "publish.output_resolution" })
        {
            var crops = Comparisons.Of(key);

            Assert.True(crops.Count >= 2, $"{key} compares at least two pictures");
            Assert.All(crops, crop =>
            {
                Assert.NotEmpty(crop.Name);
                Assert.NotEmpty(crop.Figure);
                Assert.True(File.Exists(Path.Combine(CropsDirectory(), crop.File)), crop.File);
            });
        }
    }

    [Fact]
    public void AFieldTheManifestDoesNotNameComparesNothing()
    {
        Assert.Empty(Comparisons.Of("publish.gop"));
    }

    /// <summary>
    /// The detail question carries its comparison beside its heading:
    /// a title, what to look for, and one picture per step, read off the field it asks.
    /// </summary>
    [Fact]
    public async Task TheDetailQuestionOffersItsComparison()
    {
        var flow = Flows.Setup(new SeededBackend("linux"));
        await flow.Settled;
        var detail = Detail(flow);

        Assert.True(detail.HasComparison);
        Assert.Equal("How much detail", detail.Comparison!.Title);
        Assert.NotEmpty(detail.Comparison.Sentence);
        Assert.Equal(
            Comparisons.Of("publish.cq").Select(crop => crop.Name),
            detail.Comparison.Pictures.Select(picture => picture.Name));
    }

    /// <summary>The picture of the entry the field holds is the marked one, read off the field on every pass.</summary>
    [Fact]
    public async Task ThePictureOfThePickedEntryIsMarked()
    {
        var flow = Flows.Setup(new SeededBackend("linux"));
        await flow.Settled;

        Detail(flow).Options.Single(option => option.Value == "18").Choose.Execute(null);
        await flow.Settled;
        var detail = Detail(flow);

        Assert.Equal(["Sharper"], detail.Comparison!.Pictures.Where(picture => picture.IsPicked).Select(picture => picture.Name));
    }

    private static FieldViewModel Detail(SetupViewModel flow)
        => flow.Quality.Questions.Single(question => question.Field.Key == "publish.cq").Field;
}
