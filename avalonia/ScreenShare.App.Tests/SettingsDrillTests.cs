using ScreenShare.App.Backend;
using ScreenShare.App.Features.Shell.Settings.Model;
using ScreenShare.App.Features.Shell.Settings.ViewModel;
using ScreenShare.App.Features.Shell.Update.ViewModel;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A window too narrow for the list beside a section fills with the dialog and shows one of the two.
/// The list opens first, a section replaces it, and the arrow returns to it.
/// </summary>
public sealed class SettingsDrillTests
{
    private static readonly Action<Action> Inline = action => action();

    private static AppSettingsViewModel Settings(bool compact)
    {
        var backend = new SeededBackend("linux");
        var session = new Session(backend, Inline);
        var form = new FormSession(backend, session, Inline);
        var updates = new UpdateViewModel(backend, session, Inline, Flows.Held);
        var settings = new AppSettingsViewModel(backend, form, session, updates, Inline);
        settings.SetCompact(compact);
        return settings;
    }

    private static SettingsTab Tab(AppSettingsViewModel settings, SettingsSection section)
        => settings.Tabs.Single(tab => tab.Section == section);

    [Fact]
    public void ANarrowWindowOpensOnTheList()
    {
        var settings = Settings(compact: true);

        settings.OpenCommand.Execute(null);

        Assert.True(settings.ShowsList);
        Assert.False(settings.ShowsSection);
        Assert.False(settings.ShowsBack);
    }

    [Fact]
    public void PickingASectionReplacesTheList()
    {
        var settings = Settings(compact: true);
        settings.OpenCommand.Execute(null);

        settings.SelectedTab = Tab(settings, SettingsSection.Discord);

        Assert.False(settings.ShowsList);
        Assert.True(settings.ShowsSection);
        Assert.True(settings.ShowsBack);
        Assert.Equal(AppSettingsCopy.DiscordHeading, settings.Heading);
    }

    [Fact]
    public void TheArrowReturnsToTheList()
    {
        var settings = Settings(compact: true);
        settings.OpenCommand.Execute(null);
        settings.SelectedTab = Tab(settings, SettingsSection.Discord);

        settings.BackCommand.Execute(null);

        Assert.True(settings.ShowsList);
        Assert.False(settings.ShowsSection);
        Assert.Equal(AppSettingsCopy.Title, settings.Heading);
    }

    /// <summary>A press that comes to fix one thing lands on the section holding the control that fixes it.</summary>
    [Fact]
    public void AnOpenAtASectionSkipsTheList()
    {
        var settings = Settings(compact: true);

        settings.OpenAtDiscordCommand.Execute(null);

        Assert.True(settings.ShowsSection);
        Assert.Equal(SettingsSection.Discord, settings.Section);
    }

    /// <summary>A section opens on the question it answers, drawn above whatever controls it holds.</summary>
    [Fact]
    public void EverySectionOpensOnItsQuestion()
    {
        var settings = Settings(compact: false);

        foreach (var tab in settings.Tabs)
        {
            settings.Show(true, tab.Section);

            Assert.EndsWith("?", settings.Question);
            Assert.NotEmpty(settings.Lead);
            Assert.Equal(tab.Question, settings.Question);
        }
    }

    /// <summary>Development stands under the others, set apart, its toggles being for testing the app itself.</summary>
    [Fact]
    public void DevelopmentStandsApartFromTheOtherSections()
    {
        var settings = Settings(compact: false);

        Assert.Equal([SettingsSection.Development], settings.ApartTabs.Select(tab => tab.Section));
        Assert.DoesNotContain(settings.MainTabs, tab => tab.Section == SettingsSection.Development);
        Assert.Equal(settings.Tabs.Count, settings.MainTabs.Count + settings.ApartTabs.Count);
    }

    [Fact]
    public void AWideWindowShowsTheListBesideTheSection()
    {
        var settings = Settings(compact: false);

        settings.OpenCommand.Execute(null);

        Assert.True(settings.ShowsList);
        Assert.True(settings.ShowsSection);
        Assert.False(settings.ShowsBack);
        Assert.Equal(AppSettingsCopy.Title, settings.Heading);
    }
}
