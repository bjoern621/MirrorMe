using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Copy;
using ScreenShare.App.Controls;
using ScreenShare.App.Features.Fields.Model;
using ScreenShare.App.Features.Fields.ViewModel;
using ScreenShare.App.Features.Setup.Model;
using ScreenShare.App.Features.Setup.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Strip is the resolved form's groups, never a table held here.
/// A table draws empty columns for keys no backend answers with, leaves the groups it misses unreachable,
/// and asserts nothing where the fixture is written from the same table.
/// </summary>
public sealed class SetupStepsTests
{
    private static async Task<SetupViewModel> FlowAsync()
    {
        var backend = new SeededBackend("linux");
        var flow = Flows.Setup(backend);
        await flow.Settled;
        return flow;
    }

    [Fact]
    public async Task TheStripIsTheFormsGroups()
    {
        var backend = new SeededBackend("linux");
        var form = await backend.ResolveFormAsync(await backend.SettingsAsync());
        var flow = await FlowAsync();

        // Form names the groups and this shell places them.
        var sent = form.Groups.Where(group => GroupPlacement.InSetup(group.Key)).ToList();

        Assert.Equal(sent.Select(group => group.Key), flow.Steps.Select(step => step.Key));
        // Chip titles are this shell's, keyed by the form's group key.
        // What fits on a chip is this strip's decision, the contract not seeing how wide one is.
        Assert.Equal(
            sent.Select(group => Fields.Group(group.Key).Title),
            flow.Steps.Select(step => step.Label));
    }

    /// <summary>
    /// Wizard configures what this machine sends.
    /// Watching settings on a page inside it persist only if the reader goes live.
    /// </summary>
    [Fact]
    public async Task TheWatchingGroupIsNoStepOfTheSendingWizard()
    {
        var backend = new SeededBackend("linux");
        var form = await backend.ResolveFormAsync(await backend.SettingsAsync());
        var flow = await FlowAsync();

        // Fixture carries one, so what follows tests a filter and not a form missing the group.
        Assert.Contains(form.Groups, group => GroupPlacement.InViewer(group.Key));

        Assert.DoesNotContain(flow.Steps, step => GroupPlacement.InViewer(step.Key));

        flow.CurrentStep = GroupPlacement.WatchGroup;
        Assert.Equal(flow.Steps[0].Key, flow.Steps.Single(step => step.IsCurrent).Key);
    }

    /// <summary>
    /// What the stream shares is decided at the press that starts one,
    /// so the wizard carries no step for it.
    /// </summary>
    [Fact]
    public async Task TheShareGroupIsNoStepOfTheWizard()
    {
        var backend = new SeededBackend("linux");
        var form = await backend.ResolveFormAsync(await backend.SettingsAsync());
        var flow = await FlowAsync();

        // Fixture carries one, so what follows tests a filter and not a form missing the group.
        Assert.Contains(form.Groups, group => GroupPlacement.InPicker(group.Key));

        Assert.DoesNotContain(flow.Steps, step => GroupPlacement.InPicker(step.Key));
    }

    [Fact]
    public async Task EveryStepInTheStripDrawsAForm()
    {
        var flow = await FlowAsync();

        foreach (var step in flow.Steps.ToList())
        {
            flow.CurrentStep = step.Key;

            if (step.Key == QualityLayout.GroupKey)
            {
                Assert.True(flow.ShowsQuality);
                Assert.True(flow.Quality.IsResolved);
                continue;
            }

            Assert.True(flow.ShowsFields);
            Assert.NotNull(flow.CurrentGroup);
            Assert.True(flow.CurrentGroup!.IsResolved);
            Assert.NotEmpty(flow.CurrentGroup.Fields);
        }
    }

    /// <summary>The last step's foot carries the commit in place of Continue, and no earlier step's does.</summary>
    [Fact]
    public async Task TheFlowOpensOnTheFirstGroupAndWalksToTheCommit()
    {
        var flow = await FlowAsync();

        Assert.Equal(flow.Steps[0].Key, flow.Steps.Single(step => step.IsCurrent).Key);
        Assert.False(flow.CanGoBack);

        while (flow.CanContinue)
        {
            Assert.False(flow.ShowsCommit);
            flow.ContinueCommand.Execute(null);
        }

        Assert.Equal(flow.Steps[^1].Key, flow.Steps.Single(step => step.IsCurrent).Key);
        Assert.True(flow.ShowsCommit);
    }

    /// <summary>Fallback is read through rather than written back, so a group that returns puts the reader back on it.</summary>
    [Fact]
    public async Task AStepTheFormDoesNotCarryFallsBackToTheFirst()
    {
        var flow = await FlowAsync();

        flow.CurrentStep = "a-group-no-backend-answers-with";

        Assert.Equal(flow.Steps[0].Key, flow.Steps.Single(step => step.IsCurrent).Key);
        Assert.True(flow.ShowsFields);
    }

}
