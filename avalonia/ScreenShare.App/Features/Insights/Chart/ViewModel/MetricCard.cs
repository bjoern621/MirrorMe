using ScreenShare.App.Contracts;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Insights.Chart.Model;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Insights.Chart.ViewModel;

/// <summary>
/// One value card: the figure now, against its limit.
/// Pressing it picks the figure the chart draws.
/// </summary>
public sealed class MetricCard : Observable
{
    public MetricCard(Metric metric, Action<MetricKey> pick)
    {
        Assert.NotNull(metric, "a card states one metric");
        Assert.NotNull(pick, "a card hands its press to the chart");

        Metric = metric;
        Pick = new DelegateCommand(() => pick(metric.Key));
    }

    private string _value = "";
    private string _limitCaption = "";
    private bool _isPicked;
    private bool _isPast;

    public Metric Metric { get; }

    public string Label => Metric.Label;

    public string Unit => Metric.Unit;

    public string Value { get => _value; private set => Set(ref _value, value); }

    /// <summary>"target 60", "budget 16.7", "cap 20" or "no cap".</summary>
    public string LimitCaption { get => _limitCaption; private set => Set(ref _limitCaption, value); }

    public bool IsPicked { get => _isPicked; private set => Set(ref _isPicked, value); }

    /// <summary>Whether the figure now sits past its limit, which the card draws in amber.</summary>
    public bool IsPast { get => _isPast; private set => Set(ref _isPast, value); }

    public DelegateCommand Pick { get; }

    public void Apply(double? value, Limits limits, MetricKey picked)
    {
        var limit = Metric.Limit(limits);

        Value = Metric.Number(value);
        LimitCaption = limit is { } bound
            ? $"{Metric.LimitWord} {bound.ToString(bound < 100 && bound % 1 != 0 ? "0.0" : "0")}"
            : Metric.Key == MetricKey.Bitrate ? InsightsCopy.NoCap : "";
        IsPicked = picked == Metric.Key;
        IsPast = value is { } v && limit is { } l && Metric.IsPast(v, l);
    }
}
