using System.Runtime.CompilerServices;
using Avalonia;
using Avalonia.Controls;
using Avalonia.Threading;
using Avalonia.VisualTree;

namespace MirrorMe.App.Controls;

/// <summary>
/// Edges a box green with a tick for a moment each time the value in it was taken
/// (<see cref="Features.Fields.ViewModel.FieldViewModel.AcceptedMarks"/>).
/// A value written on losing focus shows nothing on screen otherwise,
/// and a box answering a value with nothing reads as one that dropped it.
/// The mark is a pseudo-class the text box theme draws, and it clears itself (<c>Design/Inputs.axaml</c>).
/// </summary>
public static class Accepted
{
    private const string PseudoClass = ":accepted";

    /// <summary>How long the mark stands.</summary>
    private static readonly TimeSpan Marked = TimeSpan.FromSeconds(1.5);

    private static readonly ConditionalWeakTable<TextBox, DispatcherTimer> Clears = new();

    /// <summary>Counts the acceptances, each new count marking the box once.</summary>
    public static readonly AttachedProperty<int> MarksProperty =
        AvaloniaProperty.RegisterAttached<Control, int>("Marks", typeof(Accepted));

    static Accepted()
        => MarksProperty.Changed.AddClassHandler<Control>((control, _) => Mark(control));

    public static int GetMarks(Control control) => control.GetValue(MarksProperty);

    public static void SetMarks(Control control, int value) => control.SetValue(MarksProperty, value);

    /// <summary>
    /// A control not yet loaded takes its count from a binding catching up, a box drawn for a value accepted earlier,
    /// so it stays quiet.
    /// A number box carries its text box in its template, and the mark lands on that one.
    /// </summary>
    private static void Mark(Control control)
    {
        if (!control.IsLoaded)
        {
            return;
        }

        var box = control as TextBox ?? control.GetVisualDescendants().OfType<TextBox>().FirstOrDefault();
        if (box is null)
        {
            return;
        }

        ((IPseudoClasses)box.Classes).Set(PseudoClass, true);

        var clear = Clears.GetValue(box, Clearing);
        clear.Stop();
        clear.Start();
    }

    private static DispatcherTimer Clearing(TextBox box)
    {
        var clear = new DispatcherTimer { Interval = Marked };
        clear.Tick += (_, _) =>
        {
            clear.Stop();
            ((IPseudoClasses)box.Classes).Set(PseudoClass, false);
        };
        return clear;
    }
}
