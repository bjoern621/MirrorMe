using ScreenShare.Api.V1;
using ScreenShare.App.Contracts;

namespace ScreenShare.App.Features.Setup.Model;

/// <summary>One step: the form group it draws, its place in the strip, and what the chip calls it.</summary>
public sealed record SetupStepRow
{
    /// <summary>Resolved form's group this step draws.</summary>
    public required string Key { get; init; }

    /// <summary>Place in the strip, 1-based. Also the glyph an unwalked chip's badge wears.</summary>
    public required int Number { get; init; }

    public required string Label { get; init; }
}

/// <summary>
/// Steps of one publish setup, derived from the form the backend answered with.
///
/// <b>The list is not written down here.</b>
/// Which steps exist is which groups the form carries,
/// so a group added to the contract is a step that appears with nothing here to edit, and one renamed leaves no hole
/// (<c>docs/ipc-api.md</c>, "The rule").
///
/// What the shell owns is placement: the order is the form's, one group is drawn by a layout of its own
/// (<see cref="QualityLayout"/>), one by another destination entirely (<see cref="Fields.Model.GroupPlacement"/>,
/// applied by the caller).
/// The last step carries the commit at its foot (<c>Setup/View/SetupView.axaml</c>), so no step of its own holds it.
/// </summary>
public static class SetupSteps
{
    /// <summary>
    /// Steps for one resolved form: a step per group in the form's order.
    /// Empty for a form that has not arrived.
    /// </summary>
    public static IReadOnlyList<SetupStepRow> For(IReadOnlyList<FieldGroup> groups)
    {
        Assert.NotNull(groups, "building the strip needs the groups the form carried");

        var rows = new List<SetupStepRow>(groups.Count);
        foreach (var group in groups)
        {
            rows.Add(new SetupStepRow
            {
                Key = group.Key,
                Number = rows.Count + 1,
                // The backend names the group and this side names the step:
                // the chip has a width the contract cannot see.
                Label = Copy.Fields.Group(group.Key).Title,
            });
        }

        Assert.That(rows.Count == groups.Count, "a step per group", rows.Count, groups.Count);
        return rows;
    }

    /// <summary>
    /// Step this key names, null where the current form has none.
    /// Null is a real answer: the reader can be standing on a step the newest form dropped.
    /// </summary>
    public static SetupStepRow? Of(IReadOnlyList<SetupStepRow> steps, string key)
    {
        Assert.NotNull(steps, "looking a step up needs the steps to look in");

        foreach (var row in steps)
        {
            if (row.Key == key)
            {
                return row;
            }
        }

        return null;
    }

    public static int IndexOf(IReadOnlyList<SetupStepRow> steps, string key)
    {
        Assert.NotNull(steps, "looking a step up needs the steps to look in");

        for (var i = 0; i < steps.Count; i++)
        {
            if (steps[i].Key == key)
            {
                return i;
            }
        }

        return -1;
    }

    /// <summary>Step after this one, null on the last step and on a key the form dropped.</summary>
    public static SetupStepRow? After(IReadOnlyList<SetupStepRow> steps, string key)
    {
        var index = IndexOf(steps, key);
        return index >= 0 && index + 1 < steps.Count ? steps[index + 1] : null;
    }

    /// <summary>Step before this one, null on the first and on a key the form dropped.</summary>
    public static SetupStepRow? Before(IReadOnlyList<SetupStepRow> steps, string key)
    {
        var index = IndexOf(steps, key);
        return index > 0 ? steps[index - 1] : null;
    }
}
