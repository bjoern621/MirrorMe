using Avalonia;
using Avalonia.Controls;
using Avalonia.Input;

using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Viewer.Tile.Model;

namespace ScreenShare.App.Features.Viewer.Tile.View;

/// <summary>
/// Turns the wheel, a pinch and a drag over a card's picture into zoom steps on the card's <see cref="TileZoom"/>.
///
/// A touch screen pinches through <see cref="PinchGestureRecognizer"/>, a macOS touchpad through its magnify gesture,
/// a Wayland touchpad through <see cref="WaylandTouchpad"/>.
/// Windows hands a touchpad pinch over as a wheel with Ctrl held, which the wheel path takes.
/// A two-finger scroll pans under Wayland alone, the one backend naming a scroll's source (<see cref="ScrollMeanings"/>).
/// Avalonia's X11 backend raises no touchpad pinch, and a touchpad's two-finger scroll zooms there.
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

    /// <summary>Card pixels a pan moves per unit of a finger scroll, which Wayland sends without the pointer's acceleration.</summary>
    private const double PanPerScrollUnit = 3;

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

    /// <summary>Pointer's last position over the frame, where a touchpad pinch zooms, its events carrying none.</summary>
    private Point? _hoveredAt;

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
        frame.AddHandler(InputElement.PointerEnteredEvent, OnEntered);

        // A static event holds its handlers, so a card leaving the window lets go of it.
        frame.AttachedToVisualTree += (_, _) => WaylandTouchpad.Pinched += OnTouchpadPinch;
        frame.DetachedFromVisualTree += (_, _) => WaylandTouchpad.Pinched -= OnTouchpadPinch;
    }

    private void OnEntered(object? sender, PointerEventArgs e)
    {
        _hoveredAt = e.GetPosition(_frame);
        if (TopLevel.GetTopLevel(_frame) is { } top)
        {
            WaylandTouchpad.Attach(top);
        }
    }

    /// <summary>
    /// A scroll the card cannot take goes on to whatever scrolls around it:
    /// one on a card that does not zoom, and a pan on a picture shown whole.
    /// Shift with a wheel keeps its meaning of a sideways scroll.
    /// </summary>
    private void OnWheel(object? sender, PointerWheelEventArgs e)
    {
        if (!_zoomable())
        {
            return;
        }

        var meaning = ScrollMeanings.Of(WaylandTouchpad.Source);
        switch (meaning)
        {
            case ScrollMeaning.Zoom:
                if (e.KeyModifiers.HasFlag(KeyModifiers.Shift) || e.Delta.Y == 0)
                {
                    return;
                }
                Zoom(e.GetPosition(_frame), Math.Pow(NotchFactor, e.Delta.Y));
                break;
            case ScrollMeaning.Pan:
                if (!_read().IsZoomed || Laid() is not { } size)
                {
                    return;
                }
                _write(_read().Panned(e.Delta * PanPerScrollUnit, size));
                break;
            default:
                Assert.Never("unexpected scroll meaning", (int)meaning);
                break;
        }
        e.Handled = true;
    }

    /// <summary>Zooms around the pointer and moves the picture with the fingers' middle, as a touch screen pinch does.</summary>
    private void OnTouchpadPinch(TouchpadPinch pinch)
    {
        if (!_zoomable() || !_frame.IsPointerOver || _hoveredAt is not { } at || Laid() is not { } size)
        {
            return;
        }

        _write(_read().At(at, size, pinch.Factor).Panned(pinch.Moved, size));
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
        _hoveredAt = e.GetPosition(_frame);

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
