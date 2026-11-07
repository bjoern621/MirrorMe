using MirrorMe.Api.V1;
using MirrorMe.App.Backend;
using MirrorMe.App.Copy;
using MirrorMe.App.Features.Shell.StatusBar.ViewModel;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// The band's one release control: the build the window runs, and what a check found standing in its place.
///
/// The title bar is the app's on Windows and macOS alone.
/// On Linux the frame is the desktop's and nothing of the app is in it (<c>avalonia/README.md</c>),
/// so a build stated there reaches two platforms out of three.
///
/// Defects locked out: a bug report naming no build, a tester who cannot tell which one they run,
/// and a band drawing the build beside a second line about the same release.
/// </summary>
public sealed class StatusBarVersionTests
{
    private const string Build = "0.4.0";

    private static StatusBarViewModel Band(
        string version = Build, UpdateState? update = null)
    {
        var backend = new SeededBackend("linux");
        if (update is not null)
        {
            backend.Update = update;
        }

        var band = new StatusBarViewModel(Flows.Updates(backend));
        band.Show(version);
        return band;
    }

    [Fact]
    public void TheBandStatesTheBuild()
    {
        var band = Band();

        Assert.True(band.ShowsRelease);
        Assert.Contains(Build, band.Release);
    }

    /// <summary>
    /// A build nothing has answered yet is stated as nothing rather than as an empty version,
    /// the handshake being what carries it and a window opening before that.
    /// </summary>
    [Fact]
    public void ABuildNobodyAnsweredIsNotDrawn()
    {
        var band = Band(version: "");

        Assert.False(band.ShowsRelease);
        Assert.Equal("", band.Release);
    }

    /// <summary>
    /// A version reads as one at a glance.
    /// </summary>
    [Fact]
    public void TheBuildIsMarkedAsAVersion()
    {
        Assert.Equal("v" + Build, Band().Release);
    }

    /// <summary>
    /// A published release takes the build's place rather than standing beside it:
    /// one control in the band says one thing about the release.
    /// The build stays a hover away, that being what a bug report names.
    /// </summary>
    [Fact]
    public void AFoundReleaseTakesTheBuildsPlace()
    {
        var band = Band(update: new UpdateState
        {
            Stage = UpdateStage.Available,
            Running = Build,
            Latest = "v0.5.0",
        });

        Assert.True(band.ShowsRelease);
        Assert.Equal(Updates.Available, band.Release);
        Assert.Contains("v" + Build, band.ReleaseHint);
    }

    /// <summary>
    /// A check under way says so in the same control, so a press has an answer while it waits.
    /// </summary>
    [Fact]
    public void ACheckUnderWayTakesTheBuildsPlace()
    {
        var band = Band(update: new UpdateState
        {
            Stage = UpdateStage.Checking,
            Running = Build,
        });

        Assert.True(band.ShowsRelease);
        Assert.Equal(Updates.Checking, band.Release);
        Assert.True(band.Updates.IsChecking);
    }

    /// <summary>
    /// A build at the published release states that and nothing else.
    /// </summary>
    [Fact]
    public void ABuildAtThePublishedReleaseTakesTheBuildsPlace()
    {
        var band = Band(update: new UpdateState
        {
            Stage = UpdateStage.Current,
            Running = Build,
            Latest = "v" + Build,
        });

        Assert.Equal(Updates.Current, band.Release);
    }

    /// <summary>
    /// A failing check takes the slot as selectable text, that being the string a bug report carries
    /// (<c>CLAUDE.md</c>, "Every error message is selectable and copyable").
    /// The pressable control is gone while it stands, so the band still draws one thing.
    /// </summary>
    [Fact]
    public void AFailingCheckLeavesNoPressableControl()
    {
        var band = Band(update: new UpdateState
        {
            Stage = UpdateStage.Failed,
            Running = Build,
            Failure = new Text { Code = TextCode.UpdateServiceUnreadable },
        });

        Assert.False(band.ShowsRelease);
        Assert.True(band.Updates.ShowsPlainLine);
        Assert.True(band.Updates.IsFailure);
    }

    /// <summary>
    /// A check that failed and then staged the release moves the band from the plain failure line
    /// to the control, with no pass drawing both.
    /// The case is a start while a release is still uploading: the first read finds no asset,
    /// and a press after the upload finds it.
    /// </summary>
    [Fact]
    public void AFailureGivingWayToAStagedReleaseDrawsTheControl()
    {
        var backend = new SeededBackend("linux")
        {
            Update = new UpdateState
            {
                Stage = UpdateStage.Failed,
                Running = Build,
                Failure = new Text { Code = TextCode.UpdateServiceUnreadable },
            },
        };
        var session = new Session(backend, action => action());
        session.Start();
        session.Stop();
        var band = new StatusBarViewModel(Flows.Updates(backend, session));
        band.Show(Build);

        backend.Update = new UpdateState { Stage = UpdateStage.Ready, Running = Build, Latest = "v0.5.0" };
        session.Start();
        session.Stop();
        band.Updates.Apply();

        Assert.True(band.ShowsRelease);
        Assert.False(band.Updates.ShowsPlainLine);
    }
}
