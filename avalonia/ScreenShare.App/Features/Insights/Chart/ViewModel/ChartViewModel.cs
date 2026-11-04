using Avalonia;
using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Contracts;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Insights.Chart.Model;
using ScreenShare.App.Features.Insights.Model;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Insights.Chart.ViewModel;

/// <summary>
/// The three value cards and the one chart they pick.
/// The picked card's figure is drawn over the range picked beside the chart, against its limit,
/// with the timeline's marks under it and every card's figure at the moment the pointer rests on.
/// Holds no reading: each pass draws from the session's readings again.
/// </summary>
public sealed class ChartViewModel : Observable
{
    private IReadOnlyList<PublishReading> _readings = [];

    public ChartViewModel()
    {
        Cards = Metrics.All.Select(metric => new MetricCard(metric, Pick)).ToList();
        Ranges = ChartRanges.All;
        _selectedRange = Ranges[1];
    }

    public IReadOnlyList<MetricCard> Cards { get; }

    public IReadOnlyList<ChartRangeRow> Ranges { get; }

    private MetricKey _picked = MetricKey.FrameRate;
    private ChartRangeRow _selectedRange;
    private ChartPlot _plot = ChartPlot.Empty;
    private string _limitLabel = "";
    private string _average = "";
    private string _worstWord = "";
    private string _worst = "";
    private bool _isPast;
    private double _hover = double.NaN;
    private Readout? _readout;

    public MetricKey Picked { get => _picked; private set => Set(ref _picked, value); }

    public ChartRangeRow SelectedRange
    {
        get => _selectedRange;
        set
        {
            Assert.NotNull(value, "the chart always draws one range");

            if (Set(ref _selectedRange, value))
            {
                Render();
            }
        }
    }

    public ChartPlot Plot { get => _plot; private set => Set(ref _plot, value); }

    public IReadOnlyList<Point> Points => Plot.Points;

    public IReadOnlyList<bool> PastAt => Plot.Past;

    public Rect Extent => new(0, Plot.Bottom, Plot.Span, Plot.Top - Plot.Bottom);

    /// <summary>Limit as a share of the height, 0 at the top, NaN where the figure has none.</summary>
    public double LimitFraction => Plot.Limit is { } limit ? 1 - (limit - Plot.Bottom) / (Plot.Top - Plot.Bottom) : double.NaN;

    public string LimitLabel { get => _limitLabel; private set => Set(ref _limitLabel, value); }

    public IReadOnlyList<ChartMark> Marks => Plot.Marks;

    public bool HasPoints => Plot.Points.Count > 1;

    public static string NoReadings => InsightsCopy.NoReadings;

    public static string FrameRateLabel => Metrics.Of(MetricKey.FrameRate).Label;

    public static string EncodeTimeLabel => Metrics.Of(MetricKey.EncodeTime).Label;

    public static string BitrateLabel => Metrics.Of(MetricKey.Bitrate).Label;

    public string Average { get => _average; private set => Set(ref _average, value); }

    public string WorstWord { get => _worstWord; private set => Set(ref _worstWord, value); }

    public string Worst { get => _worst; private set => Set(ref _worst, value); }

    /// <summary>Whether a reading of the span sits past the limit. The header then draws its low or peak in amber.</summary>
    public bool IsPast { get => _isPast; private set => Set(ref _isPast, value); }

    /// <summary>
    /// Where the pointer rests, s from the chart's left edge, NaN while it rests nowhere.
    /// Written by the chart, and a tap pins it on a screen with no hover.
    /// </summary>
    public double HoverSeconds
    {
        get => _hover;
        set
        {
            if (Set(ref _hover, value))
            {
                Readout = ChartPlots.ReadoutAt(_readings, SelectedRange.Value, value);
            }
        }
    }

    public Readout? Readout
    {
        get => _readout;
        private set
        {
            if (Set(ref _readout, value))
            {
                OnPropertyChanged(nameof(HasReadout));
            }
        }
    }

    public bool HasReadout => Readout is not null;

    private Limits _limits = new(null, null);
    private IReadOnlyList<StreamEvent> _events = [];
    private InsightsSnapshot _reading = InsightsSnapshot.Unread;

    /// <summary>
    /// The one render function.
    /// reading: the figures the cards state now. limits: what the running stream is held to.
    /// </summary>
    public void Apply(
        IReadOnlyList<PublishReading> readings, InsightsSnapshot reading, Limits limits, IReadOnlyList<StreamEvent> events)
    {
        Assert.NotNull(readings, "a chart draws the readings the session kept");
        Assert.NotNull(reading, "the cards state one reading");
        Assert.NotNull(limits, "every figure is read against what the stream is held to");
        Assert.NotNull(events, "the chart marks the timeline's events");

        _readings = readings;
        _reading = reading;
        _limits = limits;
        _events = events;
        Render();
    }

    private void Pick(MetricKey key)
    {
        Picked = key;
        Render();
    }

    private void Render()
    {
        foreach (var card in Cards)
        {
            card.Apply(Now(card.Metric.Key), _limits, Picked);
        }

        var metric = Metrics.Of(Picked);
        Plot = ChartPlots.Of(_readings, metric, _limits, SelectedRange.Value, _events);
        OnPropertyChanged(nameof(Points));
        OnPropertyChanged(nameof(PastAt));
        OnPropertyChanged(nameof(Extent));
        OnPropertyChanged(nameof(LimitFraction));
        OnPropertyChanged(nameof(Marks));
        OnPropertyChanged(nameof(HasPoints));

        var summary = Plot.Summary;
        LimitLabel = Plot.Limit is { } limit ? $"{metric.LimitWord} {metric.Number(limit)}" : "";
        Average = metric.Number(summary.Average);
        WorstWord = metric.WorstWord;
        Worst = metric.Number(summary.Worst);
        IsPast = Plot.Past.Contains(true);
        Readout = ChartPlots.ReadoutAt(_readings, SelectedRange.Value, HoverSeconds);

        Assert.That(Cards.Count(card => card.IsPicked) == 1, "one card picks the chart", Picked);
    }

    private double? Now(MetricKey key) => key switch
    {
        MetricKey.FrameRate => _reading.Fps,
        MetricKey.EncodeTime => _reading.EncodeMs,
        MetricKey.Bitrate => _reading.EgressMbps,
        _ => Assert.Never<double?>("every metric has a figure on the reading", key),
    };
}
