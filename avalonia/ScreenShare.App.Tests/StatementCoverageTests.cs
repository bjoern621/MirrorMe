using ScreenShare.Api.V1;
using ScreenShare.App.Copy;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>Every code the backend can send reads as a sentence, so no screen shows a bare code name.</summary>
public sealed class StatementCoverageTests
{
    [Fact]
    public void EveryCodeHasASentence()
    {
        var bare = Enum.GetValues<TextCode>()
            .Where(code => code != TextCode.Unspecified)
            .Where(code => Statements.Of(new Text { Code = code }) == code.ToString())
            .ToList();

        Assert.Empty(bare);
    }

    [Fact]
    public void AFailureCarriesItsCauseAndTheThirdPartysWords()
    {
        var failure = new Text
        {
            Code = TextCode.AddressAnswered,
            Args =
            {
                new TextArg { Name = TextArgName.Address, Id = "https://relay.example" },
                new TextArg
                {
                    Name = TextArgName.Cause,
                    Text = new Text { Code = TextCode.RelayTokenUndrawn },
                },
            },
        };

        var shown = Failures.Of(failure, "403 Forbidden");

        Assert.Equal(
            "https://relay.example turned the request down. Could not get access to the relay for this group.\n403 Forbidden",
            shown);
    }
}
