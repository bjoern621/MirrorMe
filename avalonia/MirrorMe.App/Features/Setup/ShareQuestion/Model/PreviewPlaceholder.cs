namespace MirrorMe.App.Features.Setup.ShareQuestion.Model;

/// <summary>
/// Why one screen has no picture, as the two states a chooser draws differently:
/// wanted and not up yet, or refused with the backend's reason.
/// Both carried out of one call, the branch between them belonging to whoever knows the refusals
/// (<c>Features/Setup/ShareQuestion/ViewModel/MonitorPreviews.cs</c>).
/// </summary>
/// <param name="Text">The sentence. Empty where there is a picture.</param>
/// <param name="Opening">Whether a picture is still on its way, which is what the wait beside the sentence says.</param>
public readonly record struct PreviewPlaceholder(string Text, bool Opening)
{
    /// <summary>Nothing to say, which is a screen whose picture arrived.</summary>
    public static readonly PreviewPlaceholder None = new("", false);
}
