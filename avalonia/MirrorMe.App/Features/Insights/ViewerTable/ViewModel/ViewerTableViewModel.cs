using System.Collections.ObjectModel;
using MirrorMe.App.Contracts;
using MirrorMe.App.Copy;
using MirrorMe.App.Features.Insights.Model;
using MirrorMe.App.Mvvm;

namespace MirrorMe.App.Features.Insights.ViewerTable.ViewModel;

/// <summary>
/// Who is watching, and which of them is having a bad time.
/// Severity is carried by colour, weight and fill, never by a status glyph column needing its own legend.
/// <see cref="ViewerRow.IsLast"/> is the one thing derived here rather than rendered: the last row sits flush
/// against the card's rounded edge and carries no separator.
/// </summary>
public sealed class ViewerTableViewModel : Observable
{
    public ViewerTableViewModel()
    {
        Rows = [];
        Apply();
    }

    // --- Inputs -------------------------------------------------------------------

    private IReadOnlyList<ViewerRow> _reported = [];
    private int? _readers;
    private bool _isLive;

    /// <summary>
    /// Viewers as watchd last reported them, in member order.
    /// <see cref="ViewerRow.IsLast"/> is never set on it: which row ends a table is not a fact a report has.
    /// </summary>
    public IReadOnlyList<ViewerRow> Reported
    {
        get => _reported;
        set
        {
            Assert.NotNull(value, "a viewer table renders the reported viewers");

            if (Set(ref _reported, value))
            {
                Apply();
            }
        }
    }

    /// <summary>
    /// Whether a stream is in force.
    /// The card is drawn on a destination reachable with nothing publishing, and an empty table with no stream
    /// behind it is a different absence from a stream nobody has joined.
    /// </summary>
    public bool IsLive
    {
        get => _isLive;
        set
        {
            if (Set(ref _isLive, value))
            {
                Apply();
            }
        }
    }

    /// <summary>
    /// How many readers the relay counts on this stream's path. Null while nothing has been read, or nothing
    /// is publishing.
    /// Carried rather than taken off the row count, the two coming from different services: the count is what
    /// the relay serves, the rows what the viewers' apps reported.
    /// A browser or an external player counts and reports nothing, so the two can differ.
    /// </summary>
    public int? Readers
    {
        get => _readers;
        set
        {
            if (Set(ref _readers, value))
            {
                Apply();
            }
        }
    }

    // --- Outputs ------------------------------------------------------------------

    private int _strugglingCount;
    private string _notice = "";
    private bool _hasRows;

    public ObservableCollection<ViewerRow> Rows { get; }

    /// <summary>
    /// How many rows crossed a limit in <see cref="ViewerRow"/>'s severity table.
    /// The card's own summary of its rows, read by a test and by nothing else on screen: the design draws no
    /// figures on the insights destination's status band, which would otherwise echo a count this card already
    /// carries in colour (<c>Features/Shell/StatusBar</c>).
    /// </summary>
    public int StrugglingCount { get => _strugglingCount; private set => Set(ref _strugglingCount, value); }

    /// <summary>
    /// Why there are no rows, empty while there are some.
    /// The absences are asked in the order they stop being true as a stream comes up.
    /// A count standing with no rows behind it means viewers whose app sends no figures,
    /// which reads as an empty path unless the sentence says otherwise.
    /// </summary>
    public string Notice { get => _notice; private set => Set(ref _notice, value); }

    public bool HasRows { get => _hasRows; private set => Set(ref _hasRows, value); }

    /// <summary>
    /// The one render function.
    /// Stamps the separator onto every row but the last, and leaves the bound list alone where nothing differs:
    /// rows are records, so a table pushed unchanged does not repaint under the pointer.
    /// </summary>
    public void Apply()
    {
        var reported = Reported;
        var rendered = new ViewerRow[reported.Count];
        for (var i = 0; i < reported.Count; i++)
        {
            rendered[i] = reported[i] with { IsLast = i == reported.Count - 1 };
        }

        Reconcile.Onto(Rows, rendered);
        StrugglingCount = Rows.Count(row => row.IsStruggling);
        HasRows = Rows.Count > 0;

        Notice = HasRows ? ""
            : !IsLive ? Cards.ViewersIdle
            : Readers is null ? Cards.ViewersUnasked
            : Readers > 0 ? Cards.ViewersUnreported
            : Cards.ViewersNone;

        Assert.That(Rows.Count == reported.Count, "a row per reported viewer", Rows.Count, reported.Count);
        Assert.That(Rows.Count(row => row.IsLast) == (Rows.Count == 0 ? 0 : 1),
            "exactly one row ends the table", Rows.Count);
        Assert.That(HasRows == (Notice.Length == 0), "rows and the sentence standing in for them are never both on screen", HasRows);
    }
}
