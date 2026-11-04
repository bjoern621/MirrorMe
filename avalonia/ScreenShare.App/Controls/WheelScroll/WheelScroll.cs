using Avalonia;
using ScreenShare.App.Contracts;

namespace ScreenShare.App.Controls;

/// <summary>
/// Where one wheel event leaves a scroll viewer, pure so it is tested without a window.
/// <see cref="WheelScrollPresenter"/> applies it to every scroll viewer in the app.
/// </summary>
public static class WheelScroll
{
    /// <summary>How far one wheel notch moves a view, in device-independent pixels.</summary>
    public const double NotchStep = 50;

    /// <summary>How far one unit of a touchpad's smooth scroll moves a view, in device-independent pixels.</summary>
    public const double SmoothStep = 25;

    /// <summary>
    /// Pixels one unit of a delta moves a view.
    /// A backend naming no source reports a wheel in whole notches and a touchpad in fractions of one.
    /// A wheel reporting fractions there moves as a touchpad does.
    /// </summary>
    public static double StepOf(ScrollSource source, Vector delta) => source switch
    {
        ScrollSource.Wheel or ScrollSource.WheelTilt => NotchStep,
        ScrollSource.Finger or ScrollSource.Continuous => SmoothStep,
        ScrollSource.Unknown => IsWhole(delta) ? NotchStep : SmoothStep,
        _ => Assert.Never<double>("unexpected scroll source", (int)source),
    };

    /// <summary>
    /// Offset after a wheel event, held between the start and the end of what there is to scroll.
    /// An axis whose content fits keeps its offset.
    /// </summary>
    /// <param name="delta">Notches turned, positive towards the top and the left edge.</param>
    /// <param name="shift">
    /// Whether Shift is held, which turns a delta along the vertical axis alone sideways.
    /// </param>
    public static Vector Offset(Vector offset, Vector delta, Size extent, Size viewport, bool shift, ScrollSource source)
    {
        Assert.That(offset.X >= 0 && offset.Y >= 0, "a scroll offset runs from the top left corner", offset);
        Assert.That(
            extent.Width >= 0 && extent.Height >= 0 && viewport.Width >= 0 && viewport.Height >= 0,
            "an extent and a viewport are sizes", extent, viewport);

        if (shift && delta.X == 0)
        {
            delta = new Vector(delta.Y, 0);
        }

        var step = StepOf(source, delta);
        var x = Moved(offset.X, delta.X * step, extent.Width - viewport.Width);
        var y = Moved(offset.Y, delta.Y * step, extent.Height - viewport.Height);

        return new Vector(x, y);
    }

    private static bool IsWhole(Vector delta) => delta.X == Math.Round(delta.X) && delta.Y == Math.Round(delta.Y);

    /// <param name="room">How far the axis scrolls, zero or less where its content fits.</param>
    private static double Moved(double offset, double distance, double room)
    {
        if (room <= 0)
        {
            return offset;
        }

        var moved = Math.Clamp(offset - distance, 0, room);

        Assert.That(moved >= 0 && moved <= room, "a wheel keeps the offset inside what there is to scroll", moved, room);
        return moved;
    }
}
