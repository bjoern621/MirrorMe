using ScreenShare.Api.V1;
using ScreenShare.App.Controls;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Fields.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>A diagnostic anchored on a control stands under that control, where the reader fixes it.</summary>
public sealed class FieldDiagnosticsTests
{
    private static FieldGroupViewModel Group(params Diagnostic[] diagnostics)
    {
        var group = new FieldGroupViewModel((_, _) => { });
        group.Apply(
            new FieldGroup
            {
                Key = "transport",
                Fields =
                {
                    new Field { Key = "publish.uplink_mbps", Control = ControlKind.Number, Visible = true, Enabled = true, Range = new NumericRange { Min = 1, Max = 100, Step = 1 }, Value = new FieldValue { Number = 4 } },
                    new Field { Key = "publish.publish_transport", Control = ControlKind.Select, Visible = true, Enabled = true, Value = new FieldValue { Text = "srt" }, Options = { new FieldOption { Value = "srt", Enabled = true } } },
                },
            },
            Vocabulary.Empty, null, diagnostics: diagnostics);
        return group;
    }

    private static readonly Diagnostic Short = new()
    {
        Severity = Severity.Warning,
        FieldKey = "publish.uplink_mbps",
        Text = new Text { Code = TextCode.UplinkBelowPrediction },
    };

    [Fact]
    public void ADiagnosticStandsUnderTheControlItIsAnchoredOn()
    {
        var group = Group(Short);

        var anchored = Assert.Single(group.Fields.Single(field => field.Key == "publish.uplink_mbps").Diagnostics);
        Assert.Equal(CheckState.Warned, anchored.State);
        Assert.Empty(group.Fields.Single(field => field.Key == "publish.publish_transport").Diagnostics);
    }

    [Fact]
    public void AControlNothingIsAnchoredOnCarriesNoLine()
    {
        Assert.All(Group().Fields, field => Assert.False(field.HasDiagnostics));
    }
}
