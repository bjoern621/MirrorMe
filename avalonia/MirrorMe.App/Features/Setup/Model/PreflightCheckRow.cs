using MirrorMe.Api.V1;
using MirrorMe.App.Contracts;
using MirrorMe.App.Controls;
using MirrorMe.App.Features.Fields.Model;

namespace MirrorMe.App.Features.Setup.Model;

/// <summary>
/// One line of the pre-publish list, as the strip and the checks card read it.
/// A record, so a pass over an unchanged list compares equal and leaves the bound collection alone.
/// </summary>
public sealed record PreflightCheckRow
{
    public required string Text { get; init; }

    public required CheckState State { get; init; }

    /// <summary>Settings field the line is about, empty for a line about the settings as a whole.</summary>
    public required string FieldKey { get; init; }

    /// <summary>Step owning the control at fault, and the press reaching it.</summary>
    public required CheckAnchor Anchor { get; init; }
}

/// <summary>
/// The list, built from the diagnostics the form carried.
/// Every word comes from <c>Copy/</c>, keyed on the statement code the backend sent.
/// The ranking is the form's.
/// The one thing decided here is where a line is anchored: the contract names what a diagnostic is about,
/// and which screen fixes it is placement (<c>docs/ipc-api.md</c>, "The rule").
/// </summary>
public static class PreflightChecks
{
    /// <summary>
    /// The form's diagnostics that a publish answers to.
    /// A line about a control that the viewer draws stands beside the tiles it governs.
    /// </summary>
    public static IReadOnlyList<Diagnostic> About(Form? form)
    {
        if (form is null)
        {
            return [];
        }

        var elsewhere = form.Groups
            .Where(group => GroupPlacement.InViewer(group.Key))
            .SelectMany(group => group.Fields.Select(field => field.Key))
            .ToHashSet();
        return form.Diagnostics.Where(diagnostic => !elsewhere.Contains(diagnostic.FieldKey)).ToList();
    }

    /// <summary>Diagnostics as lines, in the order the form ranked them.</summary>
    /// <param name="anchorOf">
    /// Anchors one diagnostic on the screen fixing it.
    /// Answers <see cref="CheckAnchor.Nowhere"/> where nothing this app draws fixes it,
    /// and for a diagnostic about the combination rather than a field.
    /// </param>
    public static IReadOnlyList<PreflightCheckRow> Of(
        IReadOnlyList<Diagnostic> diagnostics, Func<Diagnostic, CheckAnchor> anchorOf)
    {
        Assert.NotNull(diagnostics, "building the list needs the diagnostics the form carried");
        Assert.NotNull(anchorOf, "a line needs somewhere to look up the screen that fixes it");

        return diagnostics
            .Select(diagnostic => new PreflightCheckRow
            {
                Text = Copy.Statements.Of(diagnostic.Text),
                State = Impacts.StateOf(diagnostic.Severity),
                FieldKey = diagnostic.FieldKey,
                Anchor = anchorOf(diagnostic),
            })
            .ToList();
    }

    /// <summary>
    /// Steps a blocking line names, one entry per line, empty for a line naming no field.
    /// The strip marks its chips from this, so the mark and the list count the same lines.
    /// </summary>
    public static IReadOnlyList<string> BlockedSteps(IReadOnlyList<PreflightCheckRow> checks)
    {
        Assert.NotNull(checks, "the blocked steps come out of the list the form's diagnostics became");

        return checks
            .Where(check => check.State == CheckState.Blocking)
            .Select(check => check.Anchor.StepKey)
            .ToList();
    }
}
