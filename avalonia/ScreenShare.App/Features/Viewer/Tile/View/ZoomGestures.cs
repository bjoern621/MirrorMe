using Avalonia;
using Avalonia.Controls;
using Avalonia.Input;

using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Viewer.Tile.Model;

namespace ScreenShare.App.Features.Viewer.Tile.View;

/// <summary>
/// Turns the wheel, a pinch and a drag over a card's picture into zoom steps on the card's <see cref="TileZoom"/>.
///
/// A touch screen pinches through <see cref="PinchGestureRecognizer"/>, a macOS touchpad through its magnify gesture.
/// Windows hands a touchpad pinch over as a wheel with Ctrl held, which the wheel path takes.
/// Avalonia's X11 and Wayland backends raise no touchpad pinch, and a touchpad's two-finger scroll zooms there instead.
///
/// The zoom stays the card's: this reads it and hands back the next one.
/// The state kept here belongs to one gesture in flight.
/// </summary>
internal sealed class ZoomGestures
{
    /// <summary>Zoom per wheel notch. A touchpad's smooth scroll sends fractions of a notch and zooms by the matching fraction.</summary>
    private const double NotchFactor = 1.2;

    /// <summary>Pointer travel, in card pixels, past which a press on a zoomed picture drags it rather than clicking it.</summary>
    private const double DragThreshold = 4;

    private readonly Control _frame;
    private readonly Func<bool> _zoomable;
    private readonly Func<TileZoom> _read;
    private readonly Action<TileZoom> _write;

    /// <summary>Where the left press went down, null with none held over a zoomed picture.</summary>
    private Point? _pressedAt;

    /// <summary>Last position a drag moved the picture to.</summary>
    private Point _draggedTo;

    /// <summary>Scale the pinch in flight has reported so far, its events carrying the total since it started.</summary>
    private double _pinched = 1;

    /// <summary>Whether the last press moved the picture, which keeps its release from reading as a click.</summary>
    public bool Dragged { get; private set; }

    /// <param name="frame">Unzoomed control around the picture, whose pixels every gesture is measured in.</param>
    /// <param name="zoomable">Whether this card zooms at all right now.</param>
    public ZoomGestures(Control frame, Func<bool> zoomable, Func<TileZoom> read, Action<TileZoom> write)
    {
        Assert.NotNull(frame, "zoom gestures land on a frame around the picture");
        Assert.NotNull(zoomable, "zoom gestures ask whether the card zooms");
        Assert.NotNull(read, "zoom gestures read the card's zoom");
        Assert.NotNull(write, "zoom gestures hand the next zoom back to the card");

        _frame = frame;
        _zoomable = zoomable;
        _read = read;
        _write = write;

        frame.GestureRecognizers.Add(new PinchGestureRecognizer());
        frame.AddHandler(InputElement.PinchEvent, OnPinch);
        frame.AddHandler(InputElement.PinchEndedEvent, (_, _) => _pinched = 1);
        frame.AddHandler(InputElement.PointerTouchPadGestureMagnifyEvent, OnMagnify);
        frame.AddHandler(InputElement.PointerWheelChangedEvent, OnWheel);
        frame.AddHandler(InputElement.PointerPressedEvent, OnPressed);
        frame.AddHandler(InputElement.PointerMovedEvent, OnMoved);
        frame.AddHandler(InputElement.PointerReleasedEvent, (_, _) => _pressedAt = null);
    }

    /// <summary>
    /// A wheel the card cannot zoom on goes on to whatever scrolls around it.
    /// Shift keeps its meaning of a sideways scroll.
    /// </summary>
    private void OnWheel(object? sender, PointerWheelEventArgs e)
    {
        if (!_zoomable() || e.KeyModifiers.HasFlag(KeyModifiers.Shift) || e.Delta.Y == 0)
        {
            return;
        }

        Zoom(e.GetPosition(_frame), Math.Pow(NotchFactor, e.Delta.Y));
        e.Handled = true;
    }

    private void OnPinch(object? sender, PinchEventArgs e)
    {
        if (!_zoomable() || e.Scale <= 0)
        {
            return;
        }

        Zoom(e.ScaleOrigin, e.Scale / _pinched);
        _pinched = e.Scale;
        e.Handled = true;
    }

    private void OnMagnify(object? sender, PointerDeltaEventArgs e)
    {
        var factor = 1 + e.Delta.X;
        if (!_zoomable() || factor <= 0)
        {
            return;
        }

        Zoom(e.GetPosition(_frame), factor);
        e.Handled = true;
    }

    /// <summary>
    /// Left unhandled, so a press that never moves stays the click it always was.
    /// A press the chrome handled never arrives here.
    /// </summary>
    private void OnPressed(object? sender, PointerPressedEventArgs e)
    {
        Dragged = false;
        _pressedAt = null;

        if (!_zoomable() || !_read().IsZoomed || !e.GetCurrentPoint(_frame).Properties.IsLeftButtonPressed)
        {
            return;
        }

        _pressedAt = _draggedTo = e.GetPosition(_frame);
    }

    private void OnMoved(object? sender, PointerEventArgs e)
    {
        if (_pressedAt is not { } pressedAt || !e.GetCurrentPoint(_frame).Properties.IsLeftButtonPressed)
        {
            _pressedAt = null;
            return;
        }

        var at = e.GetPosition(_frame);
        var travel = at - pressedAt;
        if (!Dragged && Math.Sqrt(travel.X * travel.X + travel.Y * travel.Y) < DragThreshold)
        {
            return;
        }

        Dragged = true;
        if (Laid() is { } size)
        {
            _write(_read().Panned(at - _draggedTo, size));
        }
        _draggedTo = at;
    }

    private void Zoom(Point anchor, double factor)
    {
        if (Laid() is { } size)
        {
            _write(_read().At(anchor, size, factor));
        }
    }

    /// <summary>Frame's size, null before its first layout, when there is nothing on screen to zoom.</summary>
    private Size? Laid() => _frame.Bounds.Size is { Width: > 0, Height: > 0 } size ? size : null;
}
