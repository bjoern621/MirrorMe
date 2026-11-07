using MirrorMe.App.Features.Setup.QualityStep.ViewModel;
using MirrorMe.App.Features.Setup.ViewModel;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>The fold under the questions: every expert control as a closed row, one row open at a time.</summary>
public sealed class ExpertFoldTests
{
    private static async Task<SetupViewModel> FlowAsync()
    {
        var flow = Flows.Setup(new SeededBackend("linux"));
        await flow.Settled;
        return flow;
    }

    private static ExpertRow Row(SetupViewModel flow, string key)
        => flow.Quality.Expert.Rows.Single(row => row.Field.Key == key);

    [Fact]
    public async Task EveryRowStartsClosed()
    {
        var flow = await FlowAsync();

        Assert.NotEmpty(flow.Quality.Expert.Rows);
        Assert.All(flow.Quality.Expert.Rows, row => Assert.False(row.IsOpen));
    }

    [Fact]
    public async Task OpeningARowClosesTheOneOpenBefore()
    {
        var flow = await FlowAsync();

        Row(flow, "publish.mode").Toggle.Execute(null);
        Row(flow, "publish.gop").Toggle.Execute(null);

        Assert.Equal(["publish.gop"], flow.Quality.Expert.Rows.Where(row => row.IsOpen).Select(row => row.Field.Key));
    }

    /// <summary>A replaced row rebuilds its container, and the fold under it then snaps to its end state.</summary>
    [Fact]
    public async Task OpeningARowKeepsEveryRowItsContainer()
    {
        var flow = await FlowAsync();
        var drawn = flow.Quality.Expert.Rows.ToList();
        var edits = 0;
        flow.Quality.Expert.Rows.CollectionChanged += (_, _) => edits++;

        drawn.First(row => row.Field.IsEnabled).Toggle.Execute(null);

        Assert.Equal(0, edits);
        Assert.True(drawn.First(row => row.Field.IsEnabled).IsOpen);
    }

    [Fact]
    public async Task PressingTheOpenRowClosesIt()
    {
        var flow = await FlowAsync();

        Row(flow, "publish.gop").Toggle.Execute(null);
        Row(flow, "publish.gop").Toggle.Execute(null);

        Assert.DoesNotContain(flow.Quality.Expert.Rows, row => row.IsOpen);
    }

    /// <summary>The open row is the reader's, so a pass the form answers keeps it open.</summary>
    [Fact]
    public async Task TheOpenRowSurvivesAResolve()
    {
        var flow = await FlowAsync();

        Row(flow, "publish.gop").Toggle.Execute(null);
        flow.Apply();
        await flow.Settled;

        Assert.True(Row(flow, "publish.gop").IsOpen);
    }

    /// <summary>A greyed field states its reason on the closed row and opens onto nothing.</summary>
    [Fact]
    public async Task AGreyedRowDoesNotOpen()
    {
        var flow = await FlowAsync();
        var greyed = flow.Quality.Expert.Rows.FirstOrDefault(row => !row.Field.IsEnabled);
        Assert.NotNull(greyed);

        greyed.Toggle.Execute(null);

        Assert.False(Row(flow, greyed.Field.Key).IsOpen);
        Assert.NotEmpty(greyed.Field.Reason);
    }
}
