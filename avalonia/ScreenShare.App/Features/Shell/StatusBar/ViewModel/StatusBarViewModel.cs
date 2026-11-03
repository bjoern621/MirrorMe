using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Shell.Update.ViewModel;
using ScreenShare.App.Mvvm;
using UpdateCopy = ScreenShare.App.Copy.Updates;

namespace ScreenShare.App.Features.Shell.StatusBar.ViewModel;

/// <summary>
/// Bottom band: the build, or the release found beside it.
/// The band holds its height in every destination and says nothing where there is nothing to state.
/// </summary>
public sealed class StatusBarViewModel : Observable
{
    /// <param name="updates">
    /// What the app says about the release published beside this build, owned once for the window.
    /// The band puts its line where the build stands and presses it;
    /// the dialog behind that line reads the same view model
    /// (<c>Features/Shell/Update/ViewModel/UpdateViewModel.cs</c>).
    /// </param>
    public StatusBarViewModel(UpdateViewModel updates)
    {
        Assert.NotNull(updates, "a status band states what it knows about the published release");

        Updates = updates;

        // The release slot is composed out of what the update view model derived,
        // so a refusal or a check landing there re-renders the band without waiting for the shell's next pass.
        updates.Rendered += Render;
    }

    /// <summary>
    /// The published release, as the band states it: one control, and the failure that takes its place.
    /// Held rather than mirrored, so the band and the dialog read one answer.
    /// </summary>
    public UpdateViewModel Updates { get; }

    // --- What the shell says -------------------------------------------------------

    private string _build = "";

    /// <summary>
    /// Band's whole input.
    /// <paramref name="build"/> is the backend's own, off the handshake, and is empty until it settles.
    /// Idempotent.
    /// </summary>
    public void Show(string build)
    {
        Assert.NotNull(build, "a status band is told which build is running");

        _build = build;
        Apply();
    }

    // --- Outputs -------------------------------------------------------------------

    private string _release = "";
    private bool _showsRelease;
    private string _releaseHint = "";

    /// <summary>
    /// The band's one line about the release: what a check found, or the running build where it found nothing.
    /// The build is marked as a version.
    /// </summary>
    public string Release { get => _release; private set => Set(ref _release, value); }

    /// <summary>
    /// Whether that control is drawn.
    /// A failure takes its place as selectable text, and a build nothing has answered leaves the slot empty.
    /// </summary>
    public bool ShowsRelease { get => _showsRelease; private set => Set(ref _showsRelease, value); }

    /// <summary>
    /// What the control says on hover: what the press does, and the build behind a line that displaced it.
    /// </summary>
    public string ReleaseHint { get => _releaseHint; private set => Set(ref _releaseHint, value); }

    /// <summary>
    /// Renders the release the band composes from, then the band.
    /// Idempotent.
    /// </summary>
    public void Apply()
    {
        Updates.Apply();
        Render();
    }

    /// <summary>
    /// One render function.
    /// Every output on every pass.
    /// Reads the update view model rather than rendering it, so a pass driven by that model's own change
    /// does not re-enter it.
    /// </summary>
    private void Render()
    {
        // Every destination, the build being the app's rather than one screen's.
        var build = _build.Length > 0 ? "v" + _build : "";

        // One control for the release, so a found release displaces the build rather than standing beside it.
        // A failure takes the slot as selectable text and leaves no control at all.
        var line = Updates.IsFailure ? "" : Updates.Line;
        Release = line.Length > 0 ? line : build;
        ShowsRelease = !Updates.IsFailure && Release.Length > 0;
        ReleaseHint = line.Length > 0 ? UpdateCopy.Tip(build, Updates.PressHint) : Updates.PressHint;

        Assert.That(
            ShowsRelease == (Release.Length > 0 && !Updates.IsFailure),
            "the release control and its text agree", ShowsRelease, Release);
        Assert.That(
            !(ShowsRelease && Updates.ShowsPlainLine),
            "the band says one thing about the release", ShowsRelease, Updates.ShowsPlainLine);
    }
}
