using ScreenShare.App.Contracts;

namespace ScreenShare.App.Features.Insights.Chart.Model;

public enum ChartRange
{
    OneMinute,
    FiveMinutes,
    Session,
}

public sealed record ChartRangeRow(ChartRange Value, string Label, double? Seconds);

/// <summary>Spans the chart offers. Session reaches back over every reading the session keeps.</summary>
public static class ChartRanges
{
    public static readonly IReadOnlyList<ChartRangeRow> All =
    [
        new(ChartRange.OneMinute, "1 min", 60),
        new(ChartRange.FiveMinutes, "5 min", 300),
        new(ChartRange.Session, "Session", null),
    ];

    public static ChartRangeRow Of(ChartRange value)
    {
        var row = All.FirstOrDefault(row => row.Value == value);
        return row ?? Assert.Never<ChartRangeRow>("every range has a row", value);
    }
}
