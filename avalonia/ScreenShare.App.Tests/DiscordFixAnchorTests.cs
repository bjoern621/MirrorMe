using ScreenShare.App.Backend;
using ScreenShare.App.Controls;
using ScreenShare.App.Features.Setup.Model;
using ScreenShare.App.Features.Setup.ViewModel;
using ScreenShare.App.Features.Shell.Settings.Model;
using ScreenShare.App.Features.Shell.Settings.ViewModel;
using ScreenShare.App.Features.Shell.Update.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A group that follows the voice channel needs a Discord link,
/// and the button making one stands in the settings dialog.
/// The check line saying so leads there, at the heading the button stands under.
/// The step drawing the group source is where the fault is reported and nowhere it can be fixed.
/// </summary>
public sealed class DiscordFixAnchorTests
{
    private static readonly Action<Action> Inline = action => action();

    /// <summary>The wizard and the dialog over one session and one draft, as the window builds them.</summary>
    private static async Task<(SetupViewModel Flow, AppSettingsViewModel Settings)> WindowAsync(
        SeededBackend backend)
    {
        var session = new Session(backend, Inline);
        var form = new FormSession(backend, session, Inline);
        var updates = new UpdateViewModel(backend, session, Inline, Flows.Held);
        var settings = new AppSettingsViewModel(backend, form, session, updates, Inline);
        var flow = new SetupViewModel(
            backend, form, session, Flows.Picker(backend, form, session), settings.OpenAtDiscordCommand, Inline);

        await flow.Settled;
        return (flow, settings);
    }

    /// <summary>Follows the voice channel with no account linked, which is what the backend blocks on.</summary>
    private static SeededBackend Unlinked() => new("linux") { FollowDiscord = true };

    [Fact]
    public async Task TheLineAboutTheLinkLeadsToTheSettingsDialog()
    {
        var (flow, settings) = await WindowAsync(Unlinked());

        var blocking = flow.Checks.Lines.Single(check => check.State == CheckState.Blocking);

        Assert.Equal("", blocking.Anchor.StepKey);
        Assert.Equal("Go to Settings · Discord", blocking.Anchor.Label);

        blocking.Anchor.Fix!.Execute(null);

        Assert.True(settings.IsOpen);
        Assert.Equal(SettingsSection.Discord, settings.Section);
    }

    /// <summary>No chip names a screen outside the flow, so the line greys the commit itself.</summary>
    [Fact]
    public async Task TheLineStillBlocksTheCommit()
    {
        var (flow, _) = await WindowAsync(Unlinked());

        Assert.False(flow.Commit.CanStartSharing);
        Assert.DoesNotContain(flow.Steps, chip => chip.IsBlocked);
    }

    /// <summary>A press on the strip's own settings button opens the dialog at its first section.</summary>
    [Fact]
    public async Task OpeningTheDialogPlainlyStandsAtTheFirstSection()
    {
        var (_, settings) = await WindowAsync(Unlinked());

        settings.OpenCommand.Execute(null);

        Assert.True(settings.IsOpen);
        Assert.Equal(SettingsSection.Window, settings.Section);
    }
}
