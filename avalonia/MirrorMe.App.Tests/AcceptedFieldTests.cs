using MirrorMe.Api.V1;
using MirrorMe.App.Backend;
using MirrorMe.App.Controls;
using MirrorMe.App.Copy;
using MirrorMe.App.Features.Fields.Model;
using MirrorMe.App.Features.Fields.ViewModel;
using MirrorMe.App.Features.Setup.Model;
using MirrorMe.App.Features.Setup.ViewModel;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// A value written on losing focus shows nothing on screen by itself,
/// so a box marks the moment the value it carried away came back taken.
/// The group key and the name also carry what the group service made of them.
/// </summary>
public sealed class AcceptedFieldTests
{
    private const string HostKey = "relay.host";
    private const string NameKey = "relay.display_name";

    private static Field Text(string key, string value, bool enabled = true) => new()
    {
        Key = key,
        Control = ControlKind.Text,
        Visible = true,
        Enabled = enabled,
        Reason = enabled ? null : new Text { Code = TextCode.GroupFollowsDiscord },
        Value = new FieldValue { Text = value },
    };

    private static FieldViewModel Rendered(string key, string value)
    {
        var field = new FieldViewModel(key, (_, _) => { });
        field.Apply(Text(key, value), Vocabulary.Empty);
        return field;
    }

    private static MembersState Joined(params string[] others)
    {
        var members = new MembersState { Joined = true };
        members.Members.Add(new Member { MemberId = "self", DisplayName = "Kim", Self = true });
        foreach (var name in others)
        {
            members.Members.Add(new Member { MemberId = name, DisplayName = name });
        }

        return members;
    }

    [Fact]
    public void ATypedValueAnsweredUnchangedMarksTheBox()
    {
        var field = Rendered(HostKey, "old.example.org");

        field.Text = "relay.example.org";
        field.Apply(Text(HostKey, "relay.example.org"), Vocabulary.Empty);

        Assert.Equal(1, field.AcceptedMarks);
    }

    [Fact]
    public void ARepairedValueLeavesTheBoxUnmarked()
    {
        var field = Rendered(HostKey, "old.example.org");

        field.Text = " relay.example.org ";
        field.Apply(Text(HostKey, "relay.example.org"), Vocabulary.Empty);

        Assert.Equal(0, field.AcceptedMarks);
    }

    [Fact]
    public void AValueWaitsForTheAnswerAboutIt()
    {
        var field = Rendered(HostKey, "old.example.org");

        field.Text = "relay.example.org";
        field.Apply(Text(HostKey, "old.example.org"), Vocabulary.Empty, answered: false);
        Assert.Equal(0, field.AcceptedMarks);

        field.Apply(Text(HostKey, "relay.example.org"), Vocabulary.Empty);
        Assert.Equal(1, field.AcceptedMarks);
    }

    [Fact]
    public void AValueTheFormRaisesALineAgainstLeavesTheBoxUnmarked()
    {
        var field = Rendered(RelayLayout.GroupKeyKey, "");
        var line = new Diagnostic
        {
            Severity = Severity.Warning,
            FieldKey = RelayLayout.GroupKeyKey,
            Text = new Text { Code = TextCode.GroupNameMissing },
        };

        field.Text = "a-key";
        field.Apply(Text(RelayLayout.GroupKeyKey, "a-key"), Vocabulary.Empty, diagnostics: [line]);

        Assert.Equal(0, field.AcceptedMarks);
    }

    /// <summary>A pass answering for a value nobody typed marks nothing, however often it runs.</summary>
    [Fact]
    public void AnAdoptedValueLeavesTheBoxUnmarked()
    {
        var field = Rendered(HostKey, "old.example.org");

        field.Apply(Text(HostKey, "relay.example.org"), Vocabulary.Empty);
        field.Apply(Text(HostKey, "relay.example.org"), Vocabulary.Empty);

        Assert.Equal(0, field.AcceptedMarks);
    }

    [Fact]
    public void APassingVerdictArrivingMarksTheBoxOnce()
    {
        var field = Rendered(RelayLayout.GroupKeyKey, "a-key");
        var joined = MembershipVerdict.Of(RelayLayout.GroupKeyKey, Joined());

        field.Apply(Text(RelayLayout.GroupKeyKey, "a-key"), Vocabulary.Empty, verdict: joined);
        field.Apply(Text(RelayLayout.GroupKeyKey, "a-key"), Vocabulary.Empty, verdict: joined);

        Assert.Equal(1, field.AcceptedMarks);
        Assert.Equal(CheckState.Passed, field.VerdictState);
        Assert.Equal(Fields.Joined(0), field.Verdict);
    }

    /// <summary>A box opening on a group joined long ago states the verdict and stays quiet.</summary>
    [Fact]
    public void AVerdictStandingOnTheFirstPassMarksNothing()
    {
        var field = new FieldViewModel(RelayLayout.GroupKeyKey, (_, _) => { });

        field.Apply(Text(RelayLayout.GroupKeyKey, "a-key"), Vocabulary.Empty,
            verdict: MembershipVerdict.Of(RelayLayout.GroupKeyKey, Joined()));

        Assert.True(field.HasVerdict);
        Assert.Equal(0, field.AcceptedMarks);
    }

    [Fact]
    public void AGreyedControlCarriesNoVerdict()
    {
        var field = new FieldViewModel(RelayLayout.GroupKeyKey, (_, _) => { });

        field.Apply(Text(RelayLayout.GroupKeyKey, "a-key", enabled: false), Vocabulary.Empty,
            verdict: MembershipVerdict.Of(RelayLayout.GroupKeyKey, Joined()));

        Assert.False(field.HasVerdict);
    }

    [Fact]
    public void AJoinedGroupCountsTheOtherMembersUnderTheKey()
    {
        var verdict = MembershipVerdict.Of(RelayLayout.GroupKeyKey, Joined("Ana", "Bo"));

        Assert.Equal(new FieldVerdict(Fields.Joined(2), CheckState.Passed), verdict);
        Assert.Equal(new FieldVerdict(Fields.NameClaimed, CheckState.Passed), MembershipVerdict.Of(NameKey, Joined()));
        Assert.Null(MembershipVerdict.Of(HostKey, Joined()));
    }

    [Fact]
    public void ATakenNameStandsUnderTheNameAlone()
    {
        var members = new MembersState { Joined = true, Refusal = new Text { Code = TextCode.GroupNameTaken } };

        Assert.Equal(CheckState.Blocking, MembershipVerdict.Of(NameKey, members)?.State);
        Assert.Null(MembershipVerdict.Of(RelayLayout.GroupKeyKey, members));
    }

    /// <summary>The form anchors a missing name on the name itself, so the membership states it nowhere.</summary>
    [Fact]
    public void AMissingNameIsLeftToTheForm()
    {
        var members = new MembersState { Refusal = new Text { Code = TextCode.GroupNameMissing } };

        Assert.Null(MembershipVerdict.Of(NameKey, members));
        Assert.Null(MembershipVerdict.Of(RelayLayout.GroupKeyKey, members));
    }

    [Fact]
    public void AGroupWithoutThisMachineInItSaysNothing()
    {
        Assert.Null(MembershipVerdict.Of(RelayLayout.GroupKeyKey, new MembersState { Joined = true }));
        Assert.Null(MembershipVerdict.Of(RelayLayout.GroupKeyKey, null));
    }

    private static FieldViewModel Relay(SetupViewModel flow, string key)
    {
        flow.CurrentStep = RelayLayout.GroupKey;
        return flow.CurrentGroup!.Fields.Single(field => field.Key == key);
    }

    [Fact]
    public async Task TheRelayStepMarksAnAddressTheBackendTook()
    {
        var flow = Flows.Setup(new SeededBackend("linux"));
        await flow.Settled;

        Relay(flow, HostKey).Text = "relay.example.org";
        await flow.Settled;

        Assert.Equal("relay.example.org", Relay(flow, HostKey).Text);
        Assert.Equal(1, Relay(flow, HostKey).AcceptedMarks);
    }

    [Fact]
    public async Task TheRelayStepStatesTheGroupThisMachineJoined()
    {
        var backend = new SeededBackend("linux") { Members = Joined("Ana") };
        var session = new Session(backend, action => action());
        _ = session.Start();
        var flow = Flows.Setup(backend, session);
        await flow.Settled;

        Assert.Equal(Fields.Joined(1), Relay(flow, RelayLayout.GroupKeyKey).Verdict);
    }
}
