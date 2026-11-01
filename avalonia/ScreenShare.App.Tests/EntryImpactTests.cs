using ScreenShare.Api.V1;
using ScreenShare.App.Controls;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Fields.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// What picking an entry costs and raises arrives on the entry (form.proto, FieldOption.estimate and impacts),
/// and a row prints it as sent.
/// </summary>
public sealed class EntryImpactTests
{
    private static FieldViewModel Rates() => Rendered(new FieldViewModel("publish.fps", (_, _) => { }));

    private static FieldViewModel Rendered(FieldViewModel field)
    {
        field.Apply(
            new Field
            {
                Key = "publish.fps",
                Control = ControlKind.NumberSelect,
                Visible = true,
                Enabled = true,
                Range = new NumericRange { Min = 1, Max = 240, Step = 1 },
                Value = new FieldValue { Number = 60 },
                Options =
                {
                    new FieldOption { Value = "30", Enabled = true, Estimate = new Estimate { BitrateMbps = 4.32 } },
                    new FieldOption { Value = "60", Enabled = true, Estimate = new Estimate { BitrateMbps = 8.7 } },
                    new FieldOption
                    {
                        Value = "120",
                        Enabled = true,
                        Estimate = new Estimate { BitrateMbps = 17.4 },
                        Impacts =
                        {
                            new Diagnostic
                            {
                                Severity = Severity.Warning,
                                FieldKey = "publish.fps",
                                Text = new Text
                                {
                                    Code = TextCode.FpsAboveRefresh,
                                    Args =
                                    {
                                        new TextArg { Name = TextArgName.Fps, Number = 120 },
                                        new TextArg { Name = TextArgName.RefreshHz, Number = 60 },
                                    },
                                },
                            },
                        },
                    },
                    new FieldOption { Value = "240", Enabled = false },
                },
            },
            Vocabulary.Empty);
        return field;
    }

    [Fact]
    public void AnEntryPrintsItsPriceAndAnUnpricedOneNone()
    {
        var rows = Rates().Options;

        Assert.Equal("4.3 Mb/s", rows.Single(row => row.Value == "30").Price);
        Assert.Equal("17.4 Mb/s", rows.Single(row => row.Value == "120").Price);
        Assert.Equal("", rows.Single(row => row.Value == "240").Price);
    }

    [Fact]
    public void AnEntryCarriesTheStatementsPickingItRaisesAtTheirRank()
    {
        var rows = Rates().Options;

        var raised = Assert.Single(rows.Single(row => row.Value == "120").Impacts);
        Assert.Equal(CheckState.Warned, raised.State);
        Assert.Contains("120", raised.Text);
        Assert.Empty(rows.Single(row => row.Value == "60").Impacts);
    }

    /// <summary>An unchanged pass builds rows equal to the last, so the bound list is left alone.</summary>
    [Fact]
    public void ASecondPassOverAnUnchangedFieldKeepsEveryRow()
    {
        var field = Rates();
        var before = field.Options.ToList();

        Rendered(field);

        Assert.All(before.Zip(field.Options), pair => Assert.Same(pair.First, pair.Second));
    }
}
