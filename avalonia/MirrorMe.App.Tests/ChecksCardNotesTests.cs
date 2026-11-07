using MirrorMe.App.Controls;
using MirrorMe.App.Features.Setup.Checks.ViewModel;
using MirrorMe.App.Features.Setup.Model;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// A note reports nothing wrong, so the card folds its notes under one closed disclosure
/// and what needs the reader reads first.
/// </summary>
public sealed class ChecksCardNotesTests
{
    private static PreflightCheckRow Row(CheckState state, string text) =>
        new() { Text = text, State = state, FieldKey = "", Anchor = CheckAnchor.Nowhere };

    [Fact]
    public void NotesFoldAwayFromTheLinesThatNeedTheReader()
    {
        var card = new ChecksCardViewModel();
        card.Apply([Row(CheckState.Warned, "w"), Row(CheckState.Note, "a"), Row(CheckState.Note, "b")], _ => false, "");

        Assert.Equal(["w"], card.Lines.Select(line => line.Text));
        Assert.Equal(["a", "b"], card.Notes.Select(line => line.Text));
        Assert.Equal("2", card.NotesCount);
        Assert.False(card.NotesShown);
        Assert.True(card.IsShown);
    }

    [Fact]
    public void ACardOfNotesAloneShowsTheClosedFold()
    {
        var card = new ChecksCardViewModel();
        card.Apply([Row(CheckState.Note, "a")], _ => false, "");

        Assert.Empty(card.Lines);
        Assert.True(card.HasNotes);
        Assert.True(card.IsShown);
    }

    [Fact]
    public void ARenderPassKeepsTheFoldTheReaderOpened()
    {
        var card = new ChecksCardViewModel();
        card.Apply([Row(CheckState.Note, "a")], _ => false, "");
        card.ToggleNotes.Execute(null);
        card.Apply([Row(CheckState.Note, "a")], _ => false, "");

        Assert.True(card.NotesShown);
    }
}
