using ScreenShare.Api.V1;
using ScreenShare.App.Copy;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// The short form a card prints for each statement the settings raise, the sentence standing in its tooltip.
/// The codes are the ones the form's per-settings diagnostics raise (backend/internal/form/diagnostics.go).
/// </summary>
public sealed class ChipsTests
{
    private static readonly TextCode[] SettingsCodes =
    [
        TextCode.DiscordNotLinked, TextCode.DiscordLinkRefused, TextCode.DiscordNoVoiceChannel,
        TextCode.GroupRequired, TextCode.GroupNameMissing,
        TextCode.NoUplinkStated, TextCode.UplinkBelowPrediction, TextCode.BurstAboveUplink,
        TextCode.CeilingHoldsQuality, TextCode.FpsAboveRefresh, TextCode.FpsMatchesRefresh,
        TextCode.EngineNotProbed, TextCode.DecodesOnCpu, TextCode.DecodesInHardwarePartly,
        TextCode.DecodesInHardwareEverywhere, TextCode.EncodesOnGpu, TextCode.JoinWait,
        TextCode.WatchDelay, TextCode.MonitorNotPriced, TextCode.NoPictureToPrice,
    ];

    private static Text Say(TextCode code, params TextArg[] args)
    {
        var text = new Text { Code = code };
        text.Args.Add(args);
        return text;
    }

    private static TextArg Num(TextArgName name, long value) => new() { Name = name, Number = value };

    private static TextArg Id(TextArgName name, string value) => new() { Name = name, Id = value };

    public static TheoryData<TextCode> Codes() => [.. SettingsCodes];

    /// <summary>A chip shares a card's width with others, so it stays a few words.</summary>
    [Theory]
    [MemberData(nameof(Codes))]
    public void EveryStatementTheSettingsRaiseHasAShortForm(TextCode code)
    {
        var chip = Chips.Of(Say(code));

        Assert.False(string.IsNullOrWhiteSpace(chip));
        Assert.True(chip.Split(' ').Length <= 5, $"{code} chip runs to \"{chip}\"");
        Assert.False(chip.EndsWith('.'), $"{code} chip ends on a period");
    }

    [Theory]
    [InlineData(TextCode.EncodesOnGpu, "Runs on the graphics card")]
    [InlineData(TextCode.DecodesInHardwareEverywhere, "Plays on every viewer")]
    [InlineData(TextCode.DecodesOnCpu, "Every viewer decodes on CPU")]
    [InlineData(TextCode.FpsMatchesRefresh, "Matches the screen")]
    public void TheMockupChipsReadAsDrawn(TextCode code, string expected)
        => Assert.Equal(expected, Chips.Of(Say(code)));

    [Theory]
    [InlineData(2000, "Viewers join within 2 s")]
    [InlineData(500, "Viewers join within 0.5 s")]
    public void TheJoinWaitReadsInSeconds(long ms, string expected)
        => Assert.Equal(expected, Chips.Of(Say(TextCode.JoinWait, Num(TextArgName.JoinWaitMs, ms))));

    [Theory]
    [InlineData(444, "~0.4 s behind")]
    [InlineData(2700, "~2.7 s behind")]
    [InlineData(32, "Under 0.1 s behind")]
    public void TheWatchDelayReadsInTenths(long ms, string expected)
        => Assert.Equal(expected, Chips.Of(Say(TextCode.WatchDelay, Num(TextArgName.DelayMs, ms))));

    /// <summary>The tooltip carries the sentence, so the two never read the same.</summary>
    [Theory]
    [MemberData(nameof(Codes))]
    public void TheSentenceSaysMoreThanTheChip(TextCode code)
    {
        var text = Say(code, Id(TextArgName.Engine, "gstreamer"), Id(TextArgName.Family, "nvenc"));

        Assert.NotEqual(Chips.Of(text), Statements.Of(text));
    }
}
