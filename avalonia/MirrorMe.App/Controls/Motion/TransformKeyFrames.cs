using Avalonia.Animation;
using Avalonia.Media;
using Avalonia.Media.Transformation;
using MirrorMe.App.Contracts;

namespace MirrorMe.App.Controls;

/// <summary>
/// Lets a keyframe animate <c>RenderTransform</c> between two transform strings, such as <c>scale(0.96)</c> to <c>none</c>.
/// Avalonia interpolates those for a transition and registers nothing for a keyframe,
/// so a style animation on the property throws as it starts.
/// </summary>
public sealed class TransformKeyFrames : InterpolatingAnimator<ITransform>
{
    private static bool _registered;

    /// <summary>Registers the animator for every <c>ITransform</c> property. A second call changes nothing.</summary>
    public static void Register()
    {
        if (_registered)
        {
            return;
        }

        _registered = true;
        Animation.RegisterCustomAnimator<ITransform, TransformKeyFrames>();
    }

    /// <summary>An unset transform reads as none.</summary>
    public override ITransform Interpolate(double progress, ITransform oldValue, ITransform newValue)
        => TransformOperations.Interpolate(Operations(oldValue), Operations(newValue), progress);

    private static TransformOperations Operations(ITransform? transform)
    {
        if (transform is null)
        {
            return TransformOperations.Identity;
        }

        return Assert.NotNull(
            transform as TransformOperations,
            "a keyframe states RenderTransform as a transform string, which parses to TransformOperations");
    }
}
