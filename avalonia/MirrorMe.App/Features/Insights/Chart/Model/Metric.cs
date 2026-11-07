using System.Globalization;
using MirrorMe.Api.V1;
using MirrorMe.App.Contracts;
using MirrorMe.App.Features.Insights.Model;

namespace MirrorMe.App.Features.Insights.Chart.Model;

public enum MetricKey
{
    FrameRate,
    EncodeTime,
    Bitrate,
}

/// <summary>What the running stream is held to. Each absent where nothing states it.</summary>
public sealed record Limits(double? TargetFps, double? CeilingMbps);

/// <summary>
/// One figure a value card states and the chart can draw, with the limit it is read against.
/// Frame rate says what a viewer feels, encode time and bitrate the two places it can break: this machine and the line.
/// </summary>
public sealed record Metric
{
    public required MetricKey Key { get; init; }

    public required string Label { get; init; }

    public required string Unit { get; init; }

    /// <summary>Word before the limit on a card: "target 60".</summary>
    public required string LimitWord { get; init; }

    /// <summary>Word before the worst reading in the chart header: "low" or "peak".</summary>
    public required string WorstWord { get; init; }

    public required string Format { get; init; }

    /// <summary>Whether a reading under the limit is the bad side, as a frame rate is.</summary>
    public required bool LowIsBad { get; init; }

    public required Func<PublishStats, double?> Read { get; init; }

    public required Func<Limits, double?> Limit { get; init; }

    public bool IsPast(double value, double limit) => LowIsBad ? value < limit : value > limit;

    /// <summary>Value with its unit, "48 fps", or the ellipsis for none.</summary>
    public string Print(double? value)
        => value is { } v ? $"{v.ToString(Format, CultureInfo.CurrentCulture)} {Unit}" : Figure.NoValue;

    /// <summary>Value alone, the card setting its unit apart.</summary>
    public string Number(double? value) => Figure.Of(value, Format);
}

/// <summary>The three figures, in the order the cards stand.</summary>
public static class Metrics
{
    public static readonly IReadOnlyList<Metric> All =
    [
        new Metric
        {
            Key = MetricKey.FrameRate,
            Label = "Frame rate",
            Unit = "fps",
            LimitWord = "target",
            WorstWord = "low",
            Format = "0",
            LowIsBad = true,
            Read = stats => stats.HasFps ? stats.Fps : null,
            Limit = limits => limits.TargetFps,
        },
        new Metric
        {
            Key = MetricKey.EncodeTime,
            Label = "Encode time",
            Unit = "ms",
            LimitWord = "budget",
            WorstWord = "peak",
            Format = "0",
            LowIsBad = false,
            Read = stats => stats.HasTransitMs ? stats.TransitMs : null,
            // A frame encoding longer than it lasts falls behind the next one.
            Limit = limits => limits.TargetFps is > 0 and var fps ? 1000 / fps : null,
        },
        new Metric
        {
            Key = MetricKey.Bitrate,
            Label = "Bitrate",
            Unit = "Mb/s",
            LimitWord = "cap",
            WorstWord = "peak",
            Format = "0.0",
            LowIsBad = false,
            Read = stats => stats.HasInstMbps ? stats.InstMbps : null,
            Limit = limits => limits.CeilingMbps,
        },
    ];

    public static Metric Of(MetricKey key)
    {
        var metric = All.FirstOrDefault(row => row.Key == key);
        return metric ?? Assert.Never<Metric>("every metric key has a row", key);
    }
}
