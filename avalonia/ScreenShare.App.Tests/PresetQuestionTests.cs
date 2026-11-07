using ScreenShare.App.Features.Setup.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>The built-in presets asked as the encode step's first question, each choice priced as the others are.</summary>
public sealed class PresetQuestionTests
{
    private static async Task<SetupViewModel> FlowAsync()
    {
        var flow = Flows.Setup(new SeededBackend("linux"));
        await flow.Settled;
        return flow;
    }

    /// <summary>The figure is the form's estimate for the settings the preset resolves to here.</summary>
    [Fact]
    public async Task AReachablePresetShowsWhatItsSettingsCostHere()
    {
        var backend = new SeededBackend("linux");
        var form = await backend.ResolveFormAsync(await backend.SettingsAsync());
        var flow = Flows.Setup(backend);
        await flow.Settled;

        var gaming = form.Presets.Single(preset => preset.Key == "gaming");
        var row = flow.Presets.Builtin.Single(row => row.Key == "gaming");

        Assert.True(row.HasPrice);
        Assert.Contains(gaming.Estimate.BitrateMbps.ToString("0.#"), row.Price);
    }

    /// <summary>The saved presets and the save box stand behind a disclosure, closed on every launch.</summary>
    [Fact]
    public async Task TheSavedPresetsStartFolded()
    {
        var flow = await FlowAsync();

        Assert.False(flow.Presets.SavedShown);

        flow.Presets.ToggleSaved.Execute(null);

        Assert.True(flow.Presets.SavedShown);
    }
}
