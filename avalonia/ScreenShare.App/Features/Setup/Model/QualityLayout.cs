using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Fields.ViewModel;

namespace ScreenShare.App.Features.Setup.Model;

/// <summary>
/// Which of the encode group's controls the step asks as a question, and which fall to the expert fold under them.
///
/// One table for both, the two being a partition:
/// a field in neither is a setting nobody can reach, and a field in both is one setting edited from two places.
/// The questions ask what a viewer notices of the encode: its smoothness, its size and its detail.
/// Naming them is placement (<c>docs/ipc-api.md</c>, "The rule"): what each offers and greys stays the form's,
/// their order is the form's, and a key the form stops sending drops out of the step.
/// </summary>
public static class QualityLayout
{
    /// <summary>
    /// The group drawn by a layout of its own rather than the generic renderer.
    /// A group the backend renames falls back to the generic renderer.
    /// </summary>
    public const string GroupKey = "quality";

    /// <summary>Controls asked as questions, each one that offers entries.</summary>
    public static readonly IReadOnlyList<string> Questions = ["publish.fps", "publish.output_resolution", "publish.cq"];

    public static bool AsQuestion(FieldViewModel field)
    {
        Assert.NotNull(field, "placing a control needs the control being placed");

        return Questions.Contains(field.Key);
    }

    /// <summary>Complement of <see cref="AsQuestion"/>, so between them the group's fields are drawn exactly once.</summary>
    public static bool InFold(FieldViewModel field) => !AsQuestion(field);
}
