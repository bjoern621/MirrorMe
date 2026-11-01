using System.Globalization;
using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Setup.Model;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Setup.StepStrip.ViewModel;

/// <summary>How far the flow has got past one chip.</summary>
public enum StepChipState
{
    Done,
    Current,
    Upcoming,
}

/// <summary>
/// One chip of the strip.
/// A record, so an unchanged pass produces rows that compare equal and the bound collection is left alone.
/// <see cref="Select"/> is the owner's own command instance for that step,
/// which makes two passes over one step equal rather than merely equivalent.
/// </summary>
public sealed record StepChipViewModel
{
    /// <summary>Which form group the step draws.</summary>
    public required string Key { get; init; }

    public required StepChipState State { get; init; }

    /// <summary>Step number, drawn in the chip's circle in every state.</summary>
    public required string Badge { get; init; }

    public required string Label { get; init; }

    /// <summary>What this step settled on, named on hover.</summary>
    public required string Value { get; init; }

    /// <summary>False on the leading chip, which joins nothing to its left.</summary>
    public required bool HasConnector { get; init; }

    public required DelegateCommand Select { get; init; }

    /// <summary>
    /// Whether a blocking line names this step.
    /// The strip is on every step, so a fault a reader is standing away from is still visible.
    /// </summary>
    public required bool IsBlocked { get; init; }

    public bool IsDone => State == StepChipState.Done;

    public bool IsCurrent => State == StepChipState.Current;

    /// <summary>Hover line: the step's name, and what it settled on where it has settled.</summary>
    public string Tip => Value.Length > 0 ? $"{Label}: {Value}" : Label;
}

/// <summary>
/// Builds the strip from one set of steps and the step being stood on.
/// Pure and total, so a render pass calls it unconditionally and the reconcile decides whether anything moved.
/// </summary>
public static class StepChips
{
    /// <param name="checks">
    /// Pre-publish list, which decides the marked chips.
    /// The list rather than a set of keys, so the mark is derived where the lines are counted.
    /// </param>
    public static IReadOnlyList<StepChipViewModel> For(
        IReadOnlyList<SetupStepRow> steps,
        string current,
        IReadOnlyList<PreflightCheckRow> checks,
        Func<SetupStepRow, string> valueOf,
        Func<string, DelegateCommand> commandOf)
    {
        Assert.NotNull(steps, "a strip is built from the steps the form described");
        Assert.NotNull(checks, "a chip is marked from the list the form's diagnostics became");
        Assert.NotNull(valueOf, "a chip needs somewhere to read its value from");
        Assert.NotNull(commandOf, "a chip needs a command to carry");

        var currentIndex = SetupSteps.IndexOf(steps, current);
        var blocked = PreflightChecks.BlockedSteps(checks);
        var chips = new List<StepChipViewModel>(steps.Count);

        for (var i = 0; i < steps.Count; i++)
        {
            var row = steps[i];

            chips.Add(new StepChipViewModel
            {
                Key = row.Key,
                State = StateOf(i, currentIndex),
                Badge = row.Number.ToString(CultureInfo.InvariantCulture),
                Label = row.Label,
                Value = valueOf(row),
                HasConnector = i > 0,
                Select = commandOf(row.Key),
                IsBlocked = blocked.Contains(row.Key),
            });
        }

        Assert.That(chips.Count == steps.Count, "a chip per step", chips.Count, steps.Count);
        Assert.That(
            chips.Count == 0 || chips.Count(chip => chip.IsCurrent) == 1,
            "a strip with steps on it has exactly one current chip", chips.Count(chip => chip.IsCurrent));
        return chips;
    }

    private static StepChipState StateOf(int index, int currentIndex)
    {
        if (index == currentIndex)
        {
            return StepChipState.Current;
        }

        if (index < currentIndex)
        {
            return StepChipState.Done;
        }

        return StepChipState.Upcoming;
    }
}
