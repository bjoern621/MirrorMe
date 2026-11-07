using MirrorMe.Api.V1;
using MirrorMe.App.Backend;
using MirrorMe.App.Features.Fields.ViewModel;
using MirrorMe.App.Features.Setup.Model;
using MirrorMe.App.Features.Setup.ViewModel;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>Every figure on the cost bar is the form's own estimate, never a number composed here.</summary>
public sealed class CostBarTests
{
    private static async Task<SetupViewModel> FlowAsync(SeededBackend backend)
    {
        var flow = Flows.Setup(backend);
        await flow.Settled;
        return flow;
    }

    /// <summary>Uplink control, reached by moving the flow to the step the fixture's form puts it on.</summary>
    private static FieldViewModel Uplink(SetupViewModel flow)
    {
        flow.CurrentStep = "network";
        return flow.CurrentGroup!.Fields.Single(field => field.Key == UplinkLayout.UplinkKey);
    }

    [Fact]
    public async Task TheHeadlineFigureIsTheFormsPrediction()
    {
        var backend = new SeededBackend("linux");
        var form = await backend.ResolveFormAsync(await backend.SettingsAsync());
        var flow = await FlowAsync(backend);

        var estimate = form.Summary.Estimate;
        var uplink = estimate.BitrateMbps + estimate.HeadroomMbps;

        Assert.True(flow.Cost.IsResolved);
        Assert.Contains(estimate.BitrateMbps.ToString("0.#"), flow.Cost.Bitrate);
        Assert.Contains(uplink.ToString("0"), flow.Cost.UplinkCaption);
        Assert.False(flow.Cost.IsOverUplink);
        Assert.InRange(flow.Cost.FillShare, 0, flow.Cost.UplinkShare);
    }

    [Fact]
    public async Task APredictionPastTheUplinkMarksTheBar()
    {
        var flow = await FlowAsync(new SeededBackend("linux"));

        Uplink(flow).Number = 4;
        await flow.Settled;

        Assert.True(flow.Cost.IsOverUplink);
    }

    /// <summary>Placement is this shell's and the form describes none, so the measurement rides on the field it writes.</summary>
    [Fact]
    public async Task TheMeasurementIsOfferedBesideTheControlItWrites()
    {
        var flow = await FlowAsync(new SeededBackend("linux"));

        Assert.True(Uplink(flow).HasAction);
        Assert.Equal("Measure", Uplink(flow).Action!.Label);

        foreach (var field in flow.CurrentGroup!.Fields.Where(field => field.Key != UplinkLayout.UplinkKey))
        {
            Assert.False(field.HasAction);
        }
    }

    /// <summary>A measured figure takes the path a typed one takes: one value, one re-resolve.</summary>
    [Fact]
    public async Task MeasuringTheUplinkWritesTheFigureIntoTheDraft()
    {
        var flow = await FlowAsync(new SeededBackend("linux"));

        Uplink(flow).Action!.Command.Execute(null);
        await flow.Settled;

        Assert.Equal((decimal)SeededBackend.MeasuredUplinkMbps, Uplink(flow).Number);
        Assert.Contains(SeededBackend.MeasuredUplinkMbps.ToString("0"), flow.Cost.UplinkCaption);
    }

    /// <summary>
    /// A live stream blocks the measurement and no field (<c>docs/field-availability.md</c>),
    /// so the figure stays editable.
    /// Lock is read through on every pass, so a stream that ended puts the button back.
    /// </summary>
    [Fact]
    public void AStreamOnTheAirGreysTheMeasurementAndSaysWhyBesideIt()
    {
        var backend = new PublishingBackend { Publish = Live("lab04") };
        var session = new Session(backend, static action => action());
        var flow = Flows.Setup(backend, session);

        Read(session, flow);

        Assert.False(Uplink(flow).Action!.Command.CanExecute(null));
        Assert.True(Uplink(flow).HasActionNotice);
        Assert.Contains("stream is publishing", Uplink(flow).ActionNotice);
        Assert.True(Uplink(flow).IsEnabled);

        backend.Publish = new PublishState();
        Read(session, flow);

        Assert.True(Uplink(flow).Action!.Command.CanExecute(null));
        Assert.False(Uplink(flow).HasActionNotice);
        Assert.Equal("", Uplink(flow).ActionNotice);
    }

    private static PublishState Live(string name) => new()
    {
        Live = new PublishState.Types.Live { Publish = new PublishSettings(), StreamName = name },
    };

    /// <summary>Reads the running state once, stops before the reconnect delay, then renders.</summary>
    private static void Read(Session session, SetupViewModel flow)
    {
        session.Start();
        session.Stop();
        flow.Apply();
    }
}
