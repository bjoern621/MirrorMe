namespace MirrorMe.App.Copy;

/// <summary>Figure in a tile's lower right.</summary>
public static class TileFigure
{
    /// <summary>"1080p · 60 fps", or "1080p" before a rate is measured.</summary>
    /// <param name="fps">Rounded to a whole frame, the measured rate wandering around the one sent.</param>
    public static string Of(int height, double? fps)
        => fps is { } rate ? $"{height}p · {Math.Round(rate):0} fps" : $"{height}p";
}
