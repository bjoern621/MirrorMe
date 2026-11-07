using System.Collections;
using System.Reflection;
using System.Runtime.CompilerServices;
using Avalonia;
using Avalonia.Controls;
using Avalonia.Logging;
using Avalonia.Threading;
using NWayland;
using NWayland.Interop;
using NWayland.Protocols.PointerGesturesUnstableV1;
using NWayland.Protocols.Wayland;

using MirrorMe.App.Contracts;

namespace MirrorMe.App.Controls;

/// <summary>One update of a touchpad pinch: the zoom since the last one, and how far the fingers' middle moved.</summary>
internal readonly record struct TouchpadPinch(double Factor, Vector Moved);

/// <summary>
/// Reads the two touchpad facts Avalonia's Wayland backend drops: a pinch, and what a scroll came from.
/// The backend binds no <c>zwp_pointer_gestures_v1</c> and discards <c>wl_pointer.axis_source</c>,
/// so this binds the pinch on Avalonia's own <c>wl_pointer</c> and taps its listener for the source.
///
/// Both live past Avalonia's public surface and are reached by reflection into Avalonia.Wayland 12.1.1.
/// A member missing under another version asserts at the first pointer over a scroll view or a card, since only an edit here fixes it.
/// A compositor serving no pointer gestures leaves the pinch out and logs once.
/// Every other backend names no source and raises no pinch here.
/// </summary>
internal static class WaylandTouchpad
{
    private const uint AxisSourceOpcode = 6;

    /// <summary>
    /// Source of the latest scroll frame, posted to the UI thread ahead of Avalonia's own post of the scroll.
    /// A libinput compositor names the source in every scroll frame, so a wheel after a touchpad reads as a wheel.
    /// </summary>
    public static ScrollSource Source { get; private set; }

    /// <summary>Raised on the UI thread for every update of a pinch on any card's window.</summary>
    public static event Action<TouchpadPinch>? Pinched;

    /// <summary>Each tapped pointer and its pinch, held as long as the pointer, Wayland thread only.</summary>
    private static readonly ConditionalWeakTable<WlPointer, object> Tapped = new();

    private static ZwpPointerGesturesV1? _gestures;
    private static bool _reportedNoGestures;

    /// <summary>
    /// Taps every pointer of the window's Wayland connection not tapped yet.
    /// Nothing happens on another backend, or for a pointer already tapped.
    /// </summary>
    public static void Attach(TopLevel top)
    {
        Assert.NotNull(top, "a touchpad tap reads the window a card sits in");

        if (top.PlatformImpl is not { } impl || impl.GetType().Assembly.GetName().Name != "Avalonia.Wayland")
        {
            return;
        }

        var worker = Assert.NotNull(Member(Member(impl, "Client"), "_worker"), "a Wayland window runs on a worker");
        var post = Assert.NotNull(worker.GetType().GetMethod("PostOob"), "a Wayland worker takes callbacks out of band");
        post.Invoke(worker, [(Action)(() => TapPointers(worker))]);
    }

    /// <summary>Runs on the Wayland thread, which owns every seat and proxy it reads.</summary>
    private static void TapPointers(object worker)
    {
        var globals = Assert.NotNull(Member(worker, "Globals"), "a worker holding a window has bound its globals");
        var seats = Assert.NotNull(Member(Member(globals, "InputDispatcher"), "_seats") as IDictionary, "the input dispatcher keeps its seats by name");

        foreach (var seat in seats.Values)
        {
            if (Member(seat!, "_pointerHandler") is not { } handler
                || Member(handler, "_pointer") is not WlPointer pointer
                || Tapped.TryGetValue(pointer, out _))
            {
                continue;
            }

            TapAxisSource(pointer);
            Tapped.Add(pointer, (object?)BindPinch(globals, pointer) ?? pointer);
        }
    }

    private static void TapAxisSource(WlPointer pointer)
    {
        var field = Assert.NotNull(
            typeof(WlProxy).GetField("_listener", BindingFlags.Instance | BindingFlags.NonPublic),
            "a Wayland proxy dispatches to one listener");
        var avalonia = Assert.NotNull(field.GetValue(pointer) as IWlEventsListener, "Avalonia listens on the pointer it bound");
        field.SetValue(pointer, new AxisSourceTap(avalonia));
    }

    /// <summary>Null on a compositor serving no pointer gestures.</summary>
    private static ZwpPointerGesturePinchV1? BindPinch(object globals, WlPointer pointer)
    {
        if (_gestures is null)
        {
            var known = Assert.NotNull(Member(globals, "_knownGlobals") as IDictionary<string, (uint Name, uint Version)>, "the globals keep every name the registry announced");
            if (!known.TryGetValue(ZwpPointerGesturesV1.ProxyType.Interface.Name, out var global))
            {
                if (!_reportedNoGestures)
                {
                    _reportedNoGestures = true;
                    Logger.TryGet(LogEventLevel.Warning, "Zoom")?.Log(null, "a touchpad pinch zooms nothing: the compositor serves no zwp_pointer_gestures_v1");
                }
                return null;
            }

            var registry = Assert.NotNull(Member(globals, "Registry") as WlRegistry, "the globals hold the registry they bound from");
            _gestures = ZwpPointerGesturesV1.Bind(registry, global.Name, 1);
        }

        return _gestures.GetPinchGesture(pointer, new PinchRelay());
    }

    /// <summary>Reads a field or property, public or not, declared on the owner's type or any base.</summary>
    private static object? Member(object? owner, string name)
    {
        Assert.NotNull(owner, "a member is read off an object of Avalonia's Wayland backend");

        const BindingFlags flags = BindingFlags.Instance | BindingFlags.Public | BindingFlags.NonPublic | BindingFlags.DeclaredOnly;
        for (var type = owner.GetType(); type is not null; type = type.BaseType)
        {
            if (type.GetField(name, flags) is { } field)
            {
                return field.GetValue(owner);
            }
            if (type.GetProperty(name, flags) is { } property)
            {
                return property.GetValue(owner);
            }
        }
        return Assert.Never<object?>("missing member of Avalonia.Wayland 12.1.1", owner.GetType().FullName, name);
    }

    /// <summary>Hands every event on to Avalonia, posting the source of each scroll frame first.</summary>
    private sealed class AxisSourceTap(IWlEventsListener avalonia) : IWlEventsListener
    {
        public void DispatchEvent(WlEventArgs args)
        {
            if (args.Opcode == AxisSourceOpcode)
            {
                var source = (WlPointer.AxisSourceEnum)args.GetUInt32(0) switch
                {
                    WlPointer.AxisSourceEnum.Wheel => ScrollSource.Wheel,
                    WlPointer.AxisSourceEnum.Finger => ScrollSource.Finger,
                    WlPointer.AxisSourceEnum.Continuous => ScrollSource.Continuous,
                    WlPointer.AxisSourceEnum.WheelTilt => ScrollSource.WheelTilt,
                    _ => ScrollSource.Unknown,
                };
                Dispatcher.UIThread.Post(() => Source = source);
            }

            avalonia.DispatchEvent(args);
        }
    }

    /// <summary>Turns the scale since the pinch began, which Wayland sends, into a step since the last update.</summary>
    private sealed class PinchRelay : ZwpPointerGesturePinchV1.Listener
    {
        private double _scale = 1;

        protected override void Begin(ZwpPointerGesturePinchV1 eventSender, uint serial, uint time, WlSurface? surface, uint fingers) => _scale = 1;

        protected override void Update(ZwpPointerGesturePinchV1 eventSender, uint time, WlFixed dx, WlFixed dy, WlFixed scale, WlFixed rotation)
        {
            var total = (double)scale;
            if (total <= 0)
            {
                return;
            }

            var step = new TouchpadPinch(total / _scale, new Vector((double)dx, (double)dy));
            _scale = total;
            Dispatcher.UIThread.Post(() => Pinched?.Invoke(step));
        }

        protected override void End(ZwpPointerGesturePinchV1 eventSender, uint serial, uint time, int cancelled) => _scale = 1;
    }
}
