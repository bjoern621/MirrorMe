using MirrorMe.App.Mvvm;

namespace MirrorMe.App.Features.Insights.Model;

public enum InsightsTab
{
    Stream,
    Viewers,
}

/// <summary>One segment of the tab switch. The Viewers label carries the count, so the label moves.</summary>
public sealed class InsightsTabRow(InsightsTab value, string label) : Observable
{
    private string _label = label;

    public InsightsTab Value { get; } = value;

    public string Label { get => _label; set => Set(ref _label, value); }
}
