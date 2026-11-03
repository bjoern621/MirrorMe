using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Insights.Model;
using ScreenShare.App.Mvvm;
using TablerIcons;

namespace ScreenShare.App.Features.Viewer.Tile.Model;

/// <summary>
/// One row of the stats panel: the figure's name, its reading and what it means.
/// Name and meaning stay fixed while the reading moves, so a row lives across samples.
/// Replacing it once a second would close its tooltip once a second.
/// </summary>
public sealed class StatLine : Observable
{
    private string _value;
    private bool _alarm;

    public StatLine(string key, string label, string value, string tip, bool alarm)
    {
        Assert.That(key.Length > 0, "a row carries the key it was read off");
        Assert.That(label.Length > 0, "a row names the figure it prints", key);

        Key = key;
        Label = label;
        Tip = tip;
        _value = value;
        _alarm = alarm;
    }

    /// <summary>Contract field or element counter that the row prints.</summary>
    public string Key { get; }

    public string Label { get; }

    /// <summary>What a reading tells the reader, shown on the row's tooltip.</summary>
    public string Tip { get; }

    /// <summary>
    /// What it reads, or <see cref="Figure.NoValue"/> where nothing has measured it.
    /// A zero never stands in for an absence.
    /// </summary>
    public string Value { get => _value; set => Set(ref _value, value); }

    /// <summary>Set on a loss or drop counter reading above zero, which prints at emphasis weight.</summary>
    public bool Alarm { get => _alarm; set => Set(ref _alarm, value); }
}

/// <summary>
/// One block of the panel: one stage of the pipeline and the figures read off it.
/// Readings get written into the rows that are there.
/// A block whose rows change gets replaced whole (<see cref="TileStats.Merge"/>).
/// Open state and toggle belong to the fold (<c>Features/Viewer/Tile/ViewModel/StatsFold.cs</c>).
/// </summary>
public sealed class StatSection : Observable
{
    private bool _open;
    private DelegateCommand? _toggle;

    public StatSection(string heading, string tip, IEnumerable<StatLine> lines, string leadKey)
    {
        Assert.That(heading.Length > 0, "a block names the stage it describes");

        Heading = heading;
        Tip = tip;
        Lines = [.. lines];
        Assert.That(Lines.Count > 0, "a block holds at least one figure", heading);

        // A block whose lead counter the element never reported still leads with something.
        Lead = Lines.FirstOrDefault(line => line.Key == leadKey) ?? Lines[0];

        foreach (var line in Lines)
        {
            line.PropertyChanged += (_, change) =>
            {
                if (change.PropertyName == nameof(StatLine.Alarm))
                {
                    OnPropertyChanged(nameof(HasAlarm));
                }
            };
        }
    }

    public string Heading { get; }

    /// <summary>What this stage of the pipeline does, shown on the heading.</summary>
    public string Tip { get; }

    /// <summary>Figures, in the order they are read off the pipeline.</summary>
    public IReadOnlyList<StatLine> Lines { get; }

    /// <summary>Row that a closed block prints beside its heading.</summary>
    public StatLine Lead { get; }

    /// <summary>Set while any row stands out, so a closed block says where to look.</summary>
    public bool HasAlarm => Lines.Any(line => line.Alarm);

    public bool IsOpen
    {
        get => _open;
        set
        {
            if (Set(ref _open, value))
            {
                OnPropertyChanged(nameof(Glyph));
            }
        }
    }

    public Icons Glyph => IsOpen ? Icons.IconChevronDown : Icons.IconChevronRight;

    public DelegateCommand? Toggle { get => _toggle; set => Set(ref _toggle, value); }
}
