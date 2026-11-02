using ScreenShare.Api.V1;
using ScreenShare.App.Controls;
using ScreenShare.App.Features.Fields.Model;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// What a control draws of the form's statements about it:
/// a warning or an error as a sentence under it, a note as a chip,
/// and no chip the picked card already carries.
/// </summary>
public sealed class ImpactSplitTests
{
    private static Diagnostic Said(TextCode code, Severity severity)
        => new() { Severity = severity, Text = new Text { Code = code } };

    private static readonly Diagnostic Warning = Said(TextCode.FpsAboveRefresh, Severity.Warning);
    private static readonly Diagnostic Note = Said(TextCode.EncodesOnGpu, Severity.Info);
    private static readonly Diagnostic OtherNote = Said(TextCode.JoinWait, Severity.Info);

    [Fact]
    public void AWarningIsALineAndANoteIsAChip()
    {
        var all = Impacts.Of([Warning, Note]);

        Assert.Equal([TextCode.FpsAboveRefresh], Impacts.Lines(all).Select(impact => impact.Code));
        Assert.Equal([TextCode.EncodesOnGpu], Impacts.Notes(all, []).Select(impact => impact.Code));
    }

    [Fact]
    public void ANoteThePickedCardCarriesIsNotDrawnTwice()
    {
        var all = Impacts.Of([Note, OtherNote]);
        var picked = Impacts.Of([Note]);

        Assert.Equal([TextCode.JoinWait], Impacts.Notes(all, picked).Select(impact => impact.Code));
    }

    [Fact]
    public void AnImpactCarriesItsChipAndItsSentence()
    {
        var impact = Impacts.Of([Note]).Single();

        Assert.Equal(CheckState.Note, impact.State);
        Assert.Equal(Copy.Chips.Of(Note.Text), impact.Chip);
        Assert.Equal(Copy.Statements.Of(Note.Text), impact.Text);
    }
}
