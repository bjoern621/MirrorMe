using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Fields.ViewModel;
using ScreenShare.App.Features.Setup.Model;
using ScreenShare.App.Features.Setup.QualityStep.ViewModel;
using ScreenShare.App.Features.Setup.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// Encode step end to end: backend describing a group, flow adopting the draft it answers with, controls drawn from it.
/// Asserted through the properties the markup binds and against no value this module wrote:
/// entries, prices, impacts and greying all arrive on the resolved form,
/// so a shell inventing a list fails here rather than on a screen (<c>docs/ipc-api.md</c>, "The rule").
/// Every test awaits, the boundary being a round trip whose answer lands on a later pass.
/// </summary>
public sealed class QualityStepTests
{
    private static async Task<SetupViewModel> FlowAsync()
    {
        var backend = new SeededBackend("linux");
        var flow = Flows.Setup(backend);
        await flow.Settled;
        return flow;
    }

    private static FieldViewModel Question(SetupViewModel flow, string key)
        => flow.Quality.Questions.Single(question => question.Field.Key == key).Field;

    private static async Task ChooseAsync(SetupViewModel flow, OptionViewModel option)
    {
        option.Choose.Execute(null);
        await flow.Settled;
    }

    /// <summary>Which controls are asked is this shell's placement, and their order the form's.</summary>
    [Fact]
    public async Task TheStepAsksItsQuestionsInTheFormsOrder()
    {
        var backend = new SeededBackend("linux");
        var form = await backend.ResolveFormAsync(await backend.SettingsAsync());
        var flow = await FlowAsync();

        var asked = form.Groups.Single(group => group.Key == QualityLayout.GroupKey).Fields
            .Select(field => field.Key)
            .Where(QualityLayout.Questions.Contains);

        Assert.True(flow.Quality.IsResolved);
        Assert.Equal(["publish.fps", "publish.output_resolution", "publish.cq"], asked);
        Assert.Equal(asked, flow.Quality.Questions.Select(question => question.Field.Key));
        Assert.All(flow.Quality.Questions, question => Assert.NotEmpty(question.Heading));
    }

    /// <summary>Between the questions and the fold, every control the group offers is drawn exactly once.</summary>
    [Fact]
    public async Task EveryOtherControlOfTheGroupSitsInTheExpertFold()
    {
        var backend = new SeededBackend("linux");
        var form = await backend.ResolveFormAsync(await backend.SettingsAsync());
        var flow = await FlowAsync();

        var sent = form.Groups.Single(group => group.Key == QualityLayout.GroupKey).Fields
            .Where(field => field.Visible)
            .Select(field => field.Key)
            .Where(key => !QualityLayout.Questions.Contains(key));

        Assert.Equal(sent, flow.Quality.Expert.Rows.Select(row => row.Field.Key));
    }

    [Fact]
    public async Task EveryQuestionOffersNamedChoices()
    {
        var flow = await FlowAsync();

        Assert.All(flow.Quality.Questions, question =>
        {
            Assert.NotEmpty(question.Field.Options);
            Assert.All(question.Field.Options, option => Assert.NotEmpty(option.Label));
        });
    }

    [Fact]
    public async Task TheOutputResolutionOffersTheSourceAndTheScalesBelowIt()
    {
        var resolution = Question(await FlowAsync(), "publish.output_resolution");

        Assert.Contains(resolution.Options, option => option.Value == "1920x1080");
        Assert.Contains(resolution.Options, option => option.Value == "2560x1440");
        Assert.DoesNotContain(resolution.Options, option => option.Value == "3840x2160");
    }

    /// <summary>An entry the form priced no differently shows what it was derived from in the figure's place.</summary>
    [Fact]
    public async Task AnUnpricedScaledResolutionSaysWhatItWasScaledFrom()
    {
        var scaled = Question(await FlowAsync(), "publish.output_resolution").Options
            .Single(option => option.Value == "1920x1080");

        Assert.Contains("2560", scaled.Figure);
    }

    [Fact]
    public async Task APricedChoiceShowsWhatPickingItCosts()
    {
        var rate = Question(await FlowAsync(), "publish.fps").Options.Single(option => option.Value == "30");

        Assert.True(rate.HasPrice);
        Assert.Equal(rate.Price, rate.Figure);
    }

    [Fact]
    public async Task PickingAResolutionMovesTheDraftAndTheQuestionShowsIt()
    {
        var flow = await FlowAsync();
        var picked = Question(flow, "publish.output_resolution").Options.Single(option => option.Value == "1280x720");

        await ChooseAsync(flow, picked);

        var after = Question(flow, "publish.output_resolution");
        Assert.Equal("1280x720", after.Picked!.Value);
        Assert.Single(after.Options, option => option.IsSelected);
    }

    /// <summary>Frame rate is a choice over a number, so a string-only write marks the entry while storing zero.</summary>
    [Fact]
    public async Task PickingAFramerateStoresTheNumberAndNotAZero()
    {
        var flow = await FlowAsync();

        await ChooseAsync(flow, Question(flow, "publish.fps").Options.Single(option => option.Value == "30"));

        var after = Question(flow, "publish.fps");
        Assert.Equal("30", after.Picked!.Value);
        Assert.Equal("30", after.Number.ToString());
    }

    /// <summary>
    /// Above the screen's refresh rate the encoder codes repeats.
    /// Legal to ask for, so the entry stays and carries what picking it raises.
    /// </summary>
    [Fact]
    public async Task AFramerateAboveTheSourceIsOfferedWithWhatItRaises()
    {
        var above = Question(await FlowAsync(), "publish.fps").Options.Single(option => option.Value == "120");

        Assert.True(above.IsEnabled);
        Assert.True(above.HasImpacts);
    }

    /// <summary>The picked choice's own statements stand under the question, once.</summary>
    [Fact]
    public async Task ThePickedChoiceCarriesItsImpactsUnderTheQuestion()
    {
        var flow = await FlowAsync();

        await ChooseAsync(flow, Question(flow, "publish.fps").Options.Single(option => option.Value == "120"));

        Assert.True(Question(flow, "publish.fps").Picked!.HasImpacts);
    }

    /// <summary>The number of a detail step means another step on another codec's scale, so the step's own name labels it.</summary>
    [Fact]
    public async Task TheDetailChoicesAreNamedByTheirStep()
    {
        var detail = Question(await FlowAsync(), "publish.cq");

        Assert.Equal(
            ["Lighter", "Clean", "Sharper"],
            detail.Options.Where(option => option.Value is "28" or "23" or "18").Select(option => option.Label));
    }

    /// <summary>Nothing here knows a mode and a quantizer are related: the backend states the greying.</summary>
    [Fact]
    public async Task TheDetailQuestionFollowsTheRateControlMode()
    {
        var flow = await FlowAsync();
        var modes = flow.Quality.Expert.Rows.Single(row => row.Field.Key == "publish.mode").Field;

        await ChooseAsync(flow, modes.Options.Single(option => option.Value == "crf"));
        Assert.True(Question(flow, "publish.cq").IsEnabled);

        await ChooseAsync(flow, modes.Options.Single(option => option.Value == "cbr"));
        Assert.False(Question(flow, "publish.cq").IsEnabled);
        Assert.NotEmpty(Question(flow, "publish.cq").Reason);
    }

    [Fact]
    public async Task TheStepChipRepeatsTheGroupsOwnSummary()
    {
        var flow = await FlowAsync();
        var chip = flow.Steps.Single(step => step.Key == QualityLayout.GroupKey);

        Assert.Equal(flow.Quality.Summary, chip.Value);
    }

    /// <summary>
    /// Equal rows over an unchanged draft let an open tooltip survive a pass
    /// (<c>docs/development-principles.md</c>, "Idempotency").
    /// </summary>
    [Fact]
    public async Task ASecondRenderPassLeavesTheQuestionsAndTheirChoicesAlone()
    {
        var flow = await FlowAsync();
        var questions = flow.Quality.Questions.ToList();
        var choices = Question(flow, "publish.output_resolution").Options.ToList();

        flow.Apply();
        await flow.Settled;

        Assert.Equal(questions, flow.Quality.Questions);
        Assert.Equal(choices, Question(flow, "publish.output_resolution").Options);
    }
}

/// <summary>The fold under the questions: every expert control as a closed row, one row open at a time.</summary>
public sealed class ExpertFoldTests
{
    private static async Task<SetupViewModel> FlowAsync()
    {
        var flow = Flows.Setup(new SeededBackend("linux"));
        await flow.Settled;
        return flow;
    }

    private static ExpertRow Row(SetupViewModel flow, string key)
        => flow.Quality.Expert.Rows.Single(row => row.Field.Key == key);

    [Fact]
    public async Task EveryRowStartsClosed()
    {
        var flow = await FlowAsync();

        Assert.NotEmpty(flow.Quality.Expert.Rows);
        Assert.All(flow.Quality.Expert.Rows, row => Assert.False(row.IsOpen));
    }

    [Fact]
    public async Task OpeningARowClosesTheOneOpenBefore()
    {
        var flow = await FlowAsync();

        Row(flow, "publish.mode").Toggle.Execute(null);
        Row(flow, "publish.gop").Toggle.Execute(null);

        Assert.Equal(["publish.gop"], flow.Quality.Expert.Rows.Where(row => row.IsOpen).Select(row => row.Field.Key));
    }

    /// <summary>A replaced row rebuilds its container, and the fold under it then snaps to its end state.</summary>
    [Fact]
    public async Task OpeningARowKeepsEveryRowItsContainer()
    {
        var flow = await FlowAsync();
        var drawn = flow.Quality.Expert.Rows.ToList();
        var edits = 0;
        flow.Quality.Expert.Rows.CollectionChanged += (_, _) => edits++;

        drawn.First(row => row.Field.IsEnabled).Toggle.Execute(null);

        Assert.Equal(0, edits);
        Assert.True(drawn.First(row => row.Field.IsEnabled).IsOpen);
    }

    [Fact]
    public async Task PressingTheOpenRowClosesIt()
    {
        var flow = await FlowAsync();

        Row(flow, "publish.gop").Toggle.Execute(null);
        Row(flow, "publish.gop").Toggle.Execute(null);

        Assert.DoesNotContain(flow.Quality.Expert.Rows, row => row.IsOpen);
    }

    /// <summary>The open row is the reader's, so a pass the form answers keeps it open.</summary>
    [Fact]
    public async Task TheOpenRowSurvivesAResolve()
    {
        var flow = await FlowAsync();

        Row(flow, "publish.gop").Toggle.Execute(null);
        flow.Apply();
        await flow.Settled;

        Assert.True(Row(flow, "publish.gop").IsOpen);
    }

    /// <summary>A greyed field states its reason on the closed row and opens onto nothing.</summary>
    [Fact]
    public async Task AGreyedRowDoesNotOpen()
    {
        var flow = await FlowAsync();
        var greyed = flow.Quality.Expert.Rows.FirstOrDefault(row => !row.Field.IsEnabled);
        Assert.NotNull(greyed);

        greyed.Toggle.Execute(null);

        Assert.False(Row(flow, greyed.Field.Key).IsOpen);
        Assert.NotEmpty(greyed.Field.Reason);
    }
}
