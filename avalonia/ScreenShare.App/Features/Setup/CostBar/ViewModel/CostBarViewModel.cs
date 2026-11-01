using ScreenShare.Api.V1;
using ScreenShare.App.Contracts;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Setup.CostBar.ViewModel;

/// <summary>
/// What the draft costs against the line carrying it, one row above the step column on every step,
/// so a choice is priced while it is being made.
///
/// Every figure is the backend's: the rate and the headroom come off <c>Summary.estimate</c>,
/// whose headroom the form computed against <c>publish.uplink_mbps</c>.
/// Nothing here predicts anything (<c>docs/ipc-api.md</c>, "The rule").
/// </summary>
public sealed class CostBarViewModel : Observable
{
    /// <summary>
    /// Slack the bar's scale leaves above the larger of prediction and uplink,
    /// so a prediction that exactly meets the line keeps the marker off the end cap.
    /// </summary>
    private const double Headroom = 1.15;

    private string _bitrate = "";
    private string _uplinkCaption = "";
    private double _fillShare;
    private double _uplinkShare;
    private bool _isOverUplink;
    private bool _isResolved;

    public CostBarViewModel() => Apply(null);

    /// <summary>Prediction with its unit: "8.7 Mb/s".</summary>
    public string Bitrate { get => _bitrate; private set => Set(ref _bitrate, value); }

    /// <summary>What the marker stands at: "of 20 Mb/s uplink", or the sentence for an uplink nobody stated.</summary>
    public string UplinkCaption { get => _uplinkCaption; private set => Set(ref _uplinkCaption, value); }

    /// <summary>Share of the bar the prediction fills, 0..1.</summary>
    public double FillShare { get => _fillShare; private set => Set(ref _fillShare, value); }

    /// <summary>Share of the bar the uplink marker stands at, 0..1. Zero where no uplink was stated, which draws no marker.</summary>
    public double UplinkShare { get => _uplinkShare; private set => Set(ref _uplinkShare, value); }

    public bool IsOverUplink { get => _isOverUplink; private set => Set(ref _isOverUplink, value); }

    /// <summary>False before the first form lands, when there is nothing to price.</summary>
    public bool IsResolved { get => _isResolved; private set => Set(ref _isResolved, value); }

    /// <summary>The one render function. Every output is read out of the argument, so a second pass notifies nothing.</summary>
    /// <param name="estimate">Backend's prediction for the draft. Null before the first form.</param>
    public void Apply(Estimate? estimate)
    {
        IsResolved = estimate is not null;

        var predicted = estimate?.BitrateMbps ?? 0;

        // Stated uplink recovered from the estimate rather than read off the settings field,
        // so both figures on the bar come out of the arithmetic that produced the headroom.
        var capacity = estimate is null ? 0 : predicted + estimate.HeadroomMbps;
        var scale = Math.Max(Math.Max(predicted, capacity), 1) * Headroom;

        Bitrate = Copy.Fields.Megabits(predicted);
        UplinkCaption = capacity > 0 ? Copy.Cards.OfUplink(Copy.Fields.Megabits(capacity)) : Copy.Cards.NoUplink;

        FillShare = Share(predicted, scale);
        UplinkShare = Share(capacity, scale);
        IsOverUplink = capacity > 0 && predicted > capacity;

        Assert.That(FillShare is >= 0 and <= 1, "the bar's fill is a share of it", FillShare);
        Assert.That(UplinkShare is >= 0 and <= 1, "the uplink marker stands on the bar", UplinkShare);
        Assert.That(!IsOverUplink || UplinkShare > 0, "a bar past its uplink has an uplink to be past", UplinkShare);
    }

    private static double Share(double value, double scale)
        => scale > 0 ? Math.Clamp(value / scale, 0, 1) : 0;
}
