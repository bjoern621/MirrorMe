using System.Runtime.CompilerServices;
using ScreenShare.App.Features.Fields.Model;
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
                Assert.NotEmpty(crop.Caption);
                Assert.True(File.Exists(Path.Combine(CropsDirectory(), crop.File)), crop.File);
            });
        }
    }

    [Fact]
    public void AFieldTheManifestDoesNotNameComparesNothing()
    {
        Assert.Empty(Comparisons.Of("publish.gop"));
    }

    /// <summary>The detail question carries the crops beside its heading, read off the field it asks.</summary>
    [Fact]
    public async Task TheDetailQuestionOffersItsComparison()
    {
        var flow = Flows.Setup(new SeededBackend("linux"));
        await flow.Settled;

        var detail = flow.Quality.Questions.Single(question => question.Field.Key == "publish.cq").Field;

        Assert.True(detail.HasComparison);
        Assert.Equal(Comparisons.Of("publish.cq"), detail.Comparison);
    }
}
