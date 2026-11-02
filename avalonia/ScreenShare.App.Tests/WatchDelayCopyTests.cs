using ScreenShare.Api.V1;
using ScreenShare.App.Copy;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>The figure a watch protocol's card prints, in tenths of a second.</summary>
public sealed class WatchDelayCopyTests
{
    private static string Behind(long ms)
    {
        var delay = new Text { Code = TextCode.WatchDelay };
        delay.Args.Add(new TextArg { Name = TextArgName.DelayMs, Number = ms });
        return Statements.Of(delay);
    }

    [Theory]
    [InlineData(444, "~0.4 s behind")]
    [InlineData(542, "~0.5 s behind")]
    [InlineData(2700, "~2.7 s behind")]
    public void ADelayReadsInTenthsOfASecond(long ms, string expected) => Assert.Equal(expected, Behind(ms));

    /// <summary>A figure rounding to nothing would read as no delay at all.</summary>
    [Fact]
    public void ADelayUnderATenthSaysSo() => Assert.Equal("Under 0.1 s behind", Behind(32));
}
