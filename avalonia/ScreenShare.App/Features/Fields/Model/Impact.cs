using ScreenShare.Api.V1;
using ScreenShare.App.Contracts;
using ScreenShare.App.Controls;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Fields.Model;

/// <summary>One statement the settings would raise, at the rank the form gave it.</summary>
public sealed record Impact
{
    public required string Text { get; init; }

    public required CheckState State { get; init; }
}

/// <summary>Diagnostics as the lines this shell draws them, in the order the form ranked them.</summary>
public static class Impacts
{
    public static ValueList<Impact> Of(IEnumerable<Diagnostic> diagnostics)
    {
        Assert.NotNull(diagnostics, "drawing what an entry raises needs the diagnostics it carried");

        return new ValueList<Impact>(diagnostics.Select(diagnostic => new Impact
        {
            Text = Copy.Statements.Of(diagnostic.Text),
            State = StateOf(diagnostic.Severity),
        }));
    }

    /// <summary>
    /// A rank as the hue it is drawn in (<c>Controls/CheckItem</c>).
    /// Exhaustive: a severity added to the contract fails here.
    /// </summary>
    public static CheckState StateOf(Severity severity) => severity switch
    {
        Severity.Error => CheckState.Blocking,
        Severity.Warning => CheckState.Warned,
        Severity.Info => CheckState.Note,
        _ => Assert.Never<CheckState>("unexpected diagnostic severity", (int)severity),
    };
}
