using ScreenShare.App.Controls;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Fields.ViewModel;
using ScreenShare.App.Features.Setup.Model;
using ScreenShare.App.Features.Setup.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// The card at the column's foot lists what the step on screen does not already draw under its controls.
/// A line about a control in front of the reader stands under that control and nowhere else.
/// </summary>
public sealed class ChecksCardTests
{
    private static async Task<SetupViewModel> FlowAsync(SeededBackend backend)
    {
        var flow = Flows.Setup(backend);
        await flow.Settled;
        return flow;
    }

    /// <summary>Under the predicted rate, what the fixture warns on. The field sits on the network step.</summary>
    private static async Task<SetupViewModel> StarvedAsync()
    {
        var flow = await FlowAsync(new SeededBackend("linux"));
        flow.CurrentStep = "network";
        flow.CurrentGroup!.Fields.Single(field => field.Key == UplinkLayout.UplinkKey).Number = 4;
        await flow.Settled;
        return flow;
    }

    [Fact]
    public async Task ALineAboutAnotherStepNamesTheStepThatOwnsIt()
    {
        var flow = await StarvedAsync();
        flow.CurrentStep = flow.Steps[0].Key;

        var warned = flow.Checks.Lines.Single(check => check.State == CheckState.Warned);
        var owner = flow.Steps.Single(step => step.Key == "network");

        Assert.Contains("upload", warned.Text, StringComparison.OrdinalIgnoreCase);
        Assert.Contains(owner.Label, warned.Anchor.Label);
        Assert.True(flow.Checks.IsShown);
    }

    /// <summary>The viewer draws the watch legs, so a line about one stands beside its tiles.</summary>
    [Fact]
    public async Task ALineAboutAControlTheViewerDrawsStaysOutOfSetup()
    {
        var flow = await FlowAsync(new SeededBackend("linux"));

        Assert.DoesNotContain(flow.Checks.Lines, check => check.FieldKey == "viewer.tile_watch_transport");
    }

    [Fact]
    public async Task ALineOnTheStepOnScreenStandsUnderItsControlAlone()
    {
        var flow = await StarvedAsync();

        Assert.DoesNotContain(flow.Checks.Lines, check => check.Anchor.StepKey == "network");
        Assert.Contains(
            flow.CurrentGroup!.Fields.Single(field => field.Key == UplinkLayout.UplinkKey).Diagnostics,
            impact => impact.State == CheckState.Warned);
    }

    /// <summary>A control behind a closed fold says nothing on screen, so the fold carries the mark of what it hides.</summary>
    [Fact]
    public async Task AClosedFoldCarriesTheMarkOfWhatItsControlsRaise()
    {
        var flow = await StarvedAsync();
        var group = flow.CurrentGroup!;

        Assert.False(group.Fold.Shown);
        Assert.True(group.Fold.HasImpacts);
        Assert.Equal(CheckState.Warned, group.Fold.ImpactState);
        Assert.Contains("upload", group.Fold.ImpactTip, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public async Task AFoldOverControlsRaisingNothingCarriesNoMark()
    {
        var flow = await FlowAsync(new SeededBackend("linux"));
        flow.CurrentStep = "network";

        Assert.False(flow.CurrentGroup!.Fold.HasImpacts);
    }

    [Fact]
    public async Task AFormWithNothingToSayListsNoLine()
    {
        var flow = await FlowAsync(new SeededBackend("linux"));

        Assert.Empty(flow.Checks.Lines);
    }

    /// <summary>Why no pipeline builds is about the settings as a whole, so it stands on the card on every step.</summary>
    [Fact]
    public async Task TheCardCarriesWhyNoPipelineBuildsOnEveryStep()
    {
        var backend = new SeededBackend("linux");
        var form = await backend.ResolveFormAsync(await backend.SettingsAsync());
        var flow = await FlowAsync(backend);

        var refusal = Refusals.Of(form.Summary.CommandRefusal);
        Assert.Contains("ffmpeg", refusal);

        foreach (var step in flow.Steps.ToList())
        {
            flow.CurrentStep = step.Key;

            Assert.True(flow.Checks.IsShown);
            Assert.Equal(refusal, flow.Checks.Refusal);
        }
    }

    /// <summary>A pass over an unchanged list produces lines that compare equal, so the bound rows are left alone.</summary>
    [Fact]
    public async Task TheSameFaultProducesTheSameLine()
    {
        var flow = await StarvedAsync();
        flow.CurrentStep = flow.Steps[0].Key;

        var first = flow.Checks.Lines.Single(check => check.State == CheckState.Warned);
        flow.Apply();

        Assert.Equal(first, flow.Checks.Lines.Single(check => check.State == CheckState.Warned));
    }
}
