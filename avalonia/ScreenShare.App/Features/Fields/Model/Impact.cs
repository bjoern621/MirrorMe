using ScreenShare.Api.V1;
using ScreenShare.App.Contracts;
using ScreenShare.App.Controls;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Fields.Model;

/// <summary>One statement the settings would raise, at the rank the form gave it.</summary>
public sealed record Impact
{
    public required TextCode Code { get; init; }

    /// <summary>Short form a card prints (<see cref="Copy.Chips"/>), empty on a line drawn as a sentence.</summary>
    public required string Chip { get; init; }

    public required string Text { get; init; }

    public required CheckState State { get; init; }

    public bool Warns => State == CheckState.Warned;

    public bool Blocks => State == CheckState.Blocking;
}

/// <summary>Diagnostics as this shell draws them, in the order the form ranked them.</summary>
public static class Impacts
{
    /// <summary>A control's statements: a note carries its chip, a warning or an error its sentence alone.</summary>
    public static ValueList<Impact> Of(IEnumerable<Diagnostic> diagnostics)
    {
        Assert.NotNull(diagnostics, "drawing what a control raises needs the diagnostics it carried");

        return new ValueList<Impact>(diagnostics.Select(diagnostic =>
            ImpactOf(diagnostic, chipped: diagnostic.Severity == Severity.Info)));
    }

    /// <summary>An entry's statements, every one a chip on its card.</summary>
    public static ValueList<Impact> Chipped(IEnumerable<Diagnostic> diagnostics)
    {
        Assert.NotNull(diagnostics, "drawing what an entry raises needs the diagnostics it carried");

        return new ValueList<Impact>(diagnostics.Select(diagnostic => ImpactOf(diagnostic, chipped: true)));
    }

    /// <summary>Warnings and errors, each a sentence under the control.</summary>
    public static ValueList<Impact> Lines(ValueList<Impact> all)
    {
        Assert.NotNull(all, "splitting a control's statements needs them");

        return new ValueList<Impact>(all.Where(impact => impact.State != CheckState.Note));
    }

    /// <summary>Notes as chips, less those the picked card already prints.</summary>
    public static ValueList<Impact> Notes(ValueList<Impact> all, IReadOnlyList<Impact> picked)
    {
        Assert.NotNull(all, "splitting a control's statements needs them");
        Assert.NotNull(picked, "a picked card's chips are read to skip them, an empty list where none is picked");

        var printed = picked.Select(impact => impact.Code).ToHashSet();
        return new ValueList<Impact>(all.Where(impact => impact.State == CheckState.Note && !printed.Contains(impact.Code)));
    }

    private static Impact ImpactOf(Diagnostic diagnostic, bool chipped) => new()
    {
        Code = diagnostic.Text.Code,
        Chip = chipped ? Copy.Chips.Of(diagnostic.Text) : "",
        Text = Copy.Statements.Of(diagnostic.Text),
        State = StateOf(diagnostic.Severity),
    };

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

    /// <summary>Lines of several controls in one list, what stops the stream first and a note last.</summary>
    public static ValueList<Impact> Ranked(IEnumerable<Impact> impacts)
    {
        Assert.NotNull(impacts, "ranking lines needs the lines");

        return new ValueList<Impact>(impacts.OrderBy(impact => Rank(impact.State)));
    }

    private static int Rank(CheckState state) => state switch
    {
        CheckState.Blocking => 0,
        CheckState.Warned => 1,
        CheckState.Note => 2,
        _ => Assert.Never<int>("an impact carries a diagnostic's rank", (int)state),
    };
}
