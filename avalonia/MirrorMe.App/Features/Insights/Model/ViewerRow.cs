using MirrorMe.Api.V1;
using MirrorMe.App.Contracts;

namespace MirrorMe.App.Features.Insights.Model;

/// <summary>
/// One viewer of this stream, as that viewer's own app reported it (<c>docs/viewer-reports.md</c>).
///
/// <b>Filled from the viewer report, cell for cell.</b> Every figure comes off the viewer's own decode, so every
/// leg carries the same ones.
/// A cell with no measurement behind it prints <see cref="Figure.NoValue"/> and never a zero, so a viewer whose
/// clock could not be compared never reads as one with no latency.
///
/// <see cref="IsLast"/> is nobody's measurement: the render function derives it, the last row sitting flush
/// against the card's rounded edge and carrying no separator.
/// </summary>
/// <param name="Name">Name the group lists this viewer by.</param>
/// <param name="Latency">ms from this computer's encoder to the viewer's decoder.</param>
/// <param name="Fps">Frames a second reaching the viewer's window.</param>
/// <param name="Dropped">Frames a second the viewer's decoder shed to keep up.</param>
/// <param name="Via">Leg this viewer watches over, in the transport vocabulary.</param>
/// <param name="IsStruggling">Whether a measured figure reached a limit in <see cref="Strain"/>.</param>
public sealed record ViewerRow(
    string Name,
    string Latency,
    string Fps,
    string Dropped,
    string Via,
    bool IsStruggling)
{
    /// <summary>
    /// Frames a second. Shedding this many reads as a viewer in trouble.
    /// Latency stays out of the rule: each leg has its own floor, and an RTMP viewer at its floor is not in trouble.
    /// </summary>
    private const double StrainingDroppedFps = 1;

    /// <summary>
    /// The whole severity rule: every figure that can put a row in the struggling state, and the measurement
    /// at which it does.
    ///
    /// A table rather than a condition in the render function or the view, on the rule every static fact here
    /// follows: the escalation lands in four places in the markup, fill, weight, the hot roles and the one amber,
    /// and a rule restated at each is four rules that can disagree (<c>docs/development-principles.md</c>,
    /// "Stateless").
    /// A figure the viewer did not measure takes part in nothing, hence each entry reading presence.
    /// </summary>
    private static readonly (Func<ViewerReport, double?> Figure, double Limit)[] Strain =
    [
        (report => report.HasDroppedFps ? report.DroppedFps : null, StrainingDroppedFps),
    ];

    public bool IsLast { get; init; }

    /// <summary>
    /// One row from one viewer report.
    /// Nothing here is decided: a cell is the viewer's figure, formatted, or the ellipsis saying it measured none.
    /// </summary>
    public static ViewerRow Of(ViewerReport report)
    {
        Assert.NotNull(report, "a viewer row describes a viewer report");

        return new ViewerRow(
            Name: report.DisplayName.Length > 0 ? report.DisplayName : Figure.NoValue,
            Latency: Figure.Of(report.HasPathMs ? report.PathMs : null, "0"),
            Fps: Figure.Of(report.HasShownFps ? report.ShownFps : null, "0"),
            Dropped: Figure.Of(report.HasDroppedFps ? report.DroppedFps : null, "0"),
            Via: report.Transport.Length > 0 ? report.Transport : Figure.NoValue,
            IsStruggling: Strain.Any(rule => rule.Figure(report) is { } measured && measured >= rule.Limit));
    }
}
