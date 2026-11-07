using System.Reflection;
using Avalonia.Controls;
using NWayland.Protocols.XdgShell;

using MirrorMe.App.Contracts;

namespace MirrorMe.App.Features.Shell.Model;

/// <summary>
/// Names a window's application to a Wayland compositor, which Avalonia's Wayland backend leaves unnamed.
/// The backend sends no <c>xdg_toplevel.set_app_id</c>, so a window switcher matches the empty id against
/// whichever desktop entry it lands on and draws that entry's name and icon.
/// The id matches the desktop entry's file name and its <c>StartupWMClass</c>
/// (<c>packaging/linux/mirrorme.desktop</c>), and the X11 class the assembly name gives.
///
/// The toplevel lives past Avalonia's public surface and is reached by reflection into Avalonia.Wayland 12.1.1.
/// A member missing under another version asserts at the first window opened, since only an edit here fixes it.
/// </summary>
internal static class WaylandAppId
{
    private const string AppId = "mirrorme";

    /// <summary>
    /// Sends the id for the window's toplevel.
    /// Must be called on every open: a window shown again after a hide gets a toplevel of its own.
    /// A compositor applies its window rules at the first frame, and an open sends the id ahead of that frame.
    /// Nothing happens on another backend.
    /// </summary>
    public static void Name(Window window)
    {
        Assert.NotNull(window, "an application id names the application of a window");

        if (window.PlatformImpl is not { } impl || impl.GetType().Assembly.GetName().Name != "Avalonia.Wayland")
        {
            return;
        }

        var proxy = Assert.NotNull(Member(impl, "_surfaceProxy"), "an opened Wayland window holds its toplevel");
        var toplevel = Assert.NotNull(Member(proxy, "_target"), "a toplevel proxy wraps the worker's toplevel");
        var worker = Assert.NotNull(Member(Member(impl, "Client"), "_worker"), "a Wayland window runs on a worker");
        var post = Assert.NotNull(worker.GetType().GetMethod("PostOob"), "a Wayland worker takes callbacks out of band");
        post.Invoke(worker, [(Action)(() => Send(toplevel))]);
    }

    /// <summary>Runs on the Wayland thread, which owns the toplevel. A toplevel destroyed meanwhile takes nothing.</summary>
    private static void Send(object toplevel)
    {
        if (Member(toplevel, "_xdgTopLevel") is XdgToplevel xdg)
        {
            xdg.SetAppId(AppId);
        }
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
}
