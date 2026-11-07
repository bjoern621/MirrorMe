using MirrorMe.App.Controls;
using MirrorMe.App.Features.Setup.Model;
using MirrorMe.App.Features.Setup.StepStrip.ViewModel;
using MirrorMe.App.Mvvm;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// A line naming a step is a way to that step, and a blocking one marks that step's chip.
/// A list that only says where a fault is fixed leaves the reader to find the step
/// and gives the strip nothing to show while they are standing somewhere else.
/// </summary>
public sealed class PreflightAnchorTests
{
    [Fact]
    public async Task PressingALineStandsOnTheStepThatFixesIt()
    {
        var flow = Flows.Setup(new SeededBackend("linux"));
        await flow.Settled;

        // Under the predicted rate, what the fixture warns on. The field sits on the network step.
        flow.CurrentStep = "network";
        flow.CurrentGroup!.Fields.Single(field => field.Key == UplinkLayout.UplinkKey).Number = 4;
        await flow.Settled;

        flow.CurrentStep = flow.Steps[0].Key;
        var warned = flow.Checks.Lines.Single(check => check.State == CheckState.Warned);

        Assert.Equal("network", warned.Anchor.StepKey);
        Assert.NotNull(warned.Anchor.Fix);

        warned.Anchor.Fix!.Execute(null);

        Assert.Equal("network", flow.CurrentStep);
    }

    /// <summary>
    /// Most controls stand behind the step's fold, closed on every launch.
    /// A press landing on the step with the control still hidden points at nothing.
    /// </summary>
    [Fact]
    public async Task PressingALineOpensTheFoldHidingTheControl()
    {
        var flow = Flows.Setup(new SeededBackend("linux"));
        await flow.Settled;

        flow.CurrentStep = "network";
        var group = flow.CurrentGroup!;
        group.Fields.Single(field => field.Key == UplinkLayout.UplinkKey).Number = 4;
        await flow.Settled;

        Assert.Contains(group.Folded, field => field.Key == UplinkLayout.UplinkKey);
        Assert.False(group.Fold.Shown);

        flow.CurrentStep = flow.Steps[0].Key;
        flow.Checks.Lines.Single(check => check.State == CheckState.Warned).Anchor.Fix!.Execute(null);

        Assert.Equal("network", flow.CurrentStep);
        Assert.True(group.Fold.Shown);
    }

    [Fact]
    public void ABlockingLineMarksTheChipOfTheStepItNames()
    {
        var chips = Chips([Line(CheckState.Blocking, "network")]);

        Assert.True(Chip(chips, "network").IsBlocked);
        Assert.False(Chip(chips, "video").IsBlocked);
    }

    [Fact]
    public void AWarnedLineMarksNothing()
    {
        var chips = Chips([Line(CheckState.Warned, "network"), Line(CheckState.Passed, "")]);

        foreach (var chip in chips)
        {
            Assert.False(chip.IsBlocked);
        }
    }

    private static readonly IReadOnlyList<SetupStepRow> Steps =
    [
        new() { Key = "video", Number = 1, Label = "Video" },
        new() { Key = "network", Number = 2, Label = "Network" },
    ];

    private static IReadOnlyList<StepChipViewModel> Chips(IReadOnlyList<PreflightCheckRow> checks)
        => StepChips.For(Steps, "video", checks, _ => "", _ => new DelegateCommand(() => { }));

    private static StepChipViewModel Chip(IReadOnlyList<StepChipViewModel> chips, string key)
        => chips.Single(chip => chip.Key == key);

    private static PreflightCheckRow Line(CheckState state, string stepKey) => new()
    {
        Text = "something the form said",
        State = state,
        FieldKey = "",
        Anchor = stepKey.Length == 0
            ? CheckAnchor.Nowhere
            : new CheckAnchor { Label = stepKey, StepKey = stepKey, Fix = new DelegateCommand(() => { }) },
    };
}
