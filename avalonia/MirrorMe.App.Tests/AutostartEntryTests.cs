using MirrorMe.App.Features.Shell.Model;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// The sign-in entry: the environment variable over the setting, and the XDG autostart file a Linux session reads.
/// </summary>
public sealed class AutostartEntryTests
{
    /// <summary><c>MIRRORME_AUTOSTART=0</c> leaves the entry alone, and every other value leaves the setting deciding.</summary>
    [Fact]
    public void OnlyZeroLeavesTheEntryAlone()
    {
        Assert.False(AutostartEntry.IsManaged("0"));
        Assert.True(AutostartEntry.IsManaged(null));
        Assert.True(AutostartEntry.IsManaged(""));
        Assert.True(AutostartEntry.IsManaged("1"));
        Assert.True(AutostartEntry.IsManaged("false"));
    }

    [Fact]
    public void AnInstallStartsTheExecutableRunningNow()
    {
        Assert.Equal(
            ("/usr/lib/mirrorme/mirrorme", "/usr/lib/mirrorme/mirrorme"),
            AutostartEntry.LinuxCommand("/usr/lib/mirrorme/mirrorme", flatpakId: null));
    }

    /// <summary>The store path changes with every update, so the entry names the program the profile carries.</summary>
    [Fact]
    public void ANixInstallStartsThroughItsProfile()
    {
        Assert.Equal(
            ("mirrorme", "mirrorme"),
            AutostartEntry.LinuxCommand("/nix/store/abc-mirrorme-0.13.1/lib/mirrorme/mirrorme", flatpakId: null));
    }

    [Fact]
    public void AFlatpakStartsThroughFlatpakRun()
    {
        Assert.Equal(
            ("flatpak run de.bjoernblessin.MirrorMe", null),
            AutostartEntry.LinuxCommand("/app/lib/mirrorme/mirrorme", "de.bjoernblessin.MirrorMe"));
    }

    /// <summary>A path holding a space goes in quotes, and its backslash is escaped for the quote and for the string.</summary>
    [Fact]
    public void AnExecArgumentIsQuotedWhereTheKeyReservesACharacter()
    {
        Assert.Equal("/opt/mirrorme/mirrorme", AutostartEntry.ExecArgument("/opt/mirrorme/mirrorme"));
        Assert.Equal("\"/home/a b/mirrorme\"", AutostartEntry.ExecArgument("/home/a b/mirrorme"));
        Assert.Equal("\"/home/a\\\\$b/mirrorme\"", AutostartEntry.ExecArgument("/home/a$b/mirrorme"));
        Assert.Equal("/home/100%%/mirrorme", AutostartEntry.ExecArgument("/home/100%/mirrorme"));
    }

    /// <summary>A Flatpak names no program on the host, so its entry carries no TryExec.</summary>
    [Fact]
    public void TheEntrySkipsTryExecWhereThereIsNoHostProgram()
    {
        Assert.Contains("TryExec=/usr/bin/mirrorme\n", AutostartEntry.DesktopEntry("/usr/bin/mirrorme", "/usr/bin/mirrorme"));
        Assert.DoesNotContain("TryExec=", AutostartEntry.DesktopEntry("flatpak run de.bjoernblessin.MirrorMe", null));
    }

    [Fact]
    public void TheEntryLandsInTheSessionsConfigDirectory()
    {
        Assert.Equal("/cfg/autostart", AutostartEntry.AutostartDirectory(null, null, "/cfg", "/home/a"));
        Assert.Equal("/home/a/.config/autostart", AutostartEntry.AutostartDirectory(null, null, null, "/home/a"));
        Assert.Equal("/home/a/.config/autostart", AutostartEntry.AutostartDirectory(null, null, "relative", "/home/a"));
    }

    /// <summary>Inside a Flatpak the sandbox's own config directory reaches no session, so the host's takes the entry.</summary>
    [Fact]
    public void AFlatpakWritesIntoTheHostsConfigDirectory()
    {
        Assert.Equal(
            "/home/a/.config/autostart",
            AutostartEntry.AutostartDirectory("de.bjoernblessin.MirrorMe", null, "/home/a/.var/app/x/config", "/home/a"));
        Assert.Equal(
            "/cfg/autostart",
            AutostartEntry.AutostartDirectory("de.bjoernblessin.MirrorMe", "/cfg", "/home/a/.var/app/x/config", "/home/a"));
    }
}
