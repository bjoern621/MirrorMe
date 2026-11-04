using System.Collections.ObjectModel;
using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Viewer.Tile.Model;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Viewer.Tile.ViewModel;

/// <summary>
/// Stats panel's disclosure: every block closed, at most one open, so the panel shows one stage at a time.
///
/// The open block belongs to the reader and holds the one state kept here.
/// It clears once its heading leaves the panel, so a block that returns comes back closed.
/// A panel going up again starts empty, which clears it the same way.
/// </summary>
public sealed class StatsFold
{
    private readonly ObservableCollection<StatSection> _panel;
    private readonly Dictionary<string, DelegateCommand> _toggles = [];
    private string _open = "";

    public StatsFold(ObservableCollection<StatSection> panel)
    {
        Assert.NotNull(panel, "a fold opens and closes the blocks of one panel");

        _panel = panel;
    }

    /// <summary>Writes the open state and the toggle onto every block. Idempotent.</summary>
    public void Apply()
    {
        var headings = _panel.Select(section => section.Heading).ToHashSet();
        foreach (var gone in _toggles.Keys.Where(heading => !headings.Contains(heading)).ToList())
        {
            _toggles.Remove(gone);
        }
        if (!headings.Contains(_open))
        {
            _open = "";
        }

        foreach (var section in _panel)
        {
            section.IsOpen = section.Heading == _open;
            section.Toggle = ToggleOf(section.Heading);
        }

        Assert.That(_panel.Count(section => section.IsOpen) <= 1, "the panel holds one block open at most", _open);
    }

    /// <summary>One command per heading across passes, so a write of the same toggle notifies nothing.</summary>
    private DelegateCommand ToggleOf(string heading)
    {
        if (!_toggles.TryGetValue(heading, out var command))
        {
            command = new DelegateCommand(() => Toggle(heading));
            _toggles[heading] = command;
        }
        return command;
    }

    /// <summary>Two presses make two states, the departure from idempotency that every disclosure makes.</summary>
    private void Toggle(string heading)
    {
        _open = _open == heading ? "" : heading;
        Apply();
    }
}
