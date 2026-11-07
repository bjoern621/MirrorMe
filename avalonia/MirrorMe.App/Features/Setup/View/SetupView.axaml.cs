using Avalonia.Controls;
using MirrorMe.App.Features.Shell.Model;

namespace MirrorMe.App.Features.Setup.View;

/// <summary>
/// Setup flow as the shell embeds it: a <see cref="UserControl"/> over <c>SetupViewModel</c>, owning no window,
/// title bar or nav strip.
/// Markup apart from the narrow class, so every write is a command or a bound input
/// and the view model's one render function still restores a correct view.
/// </summary>
public sealed partial class SetupView : UserControl
{
    public SetupView()
    {
        InitializeComponent();

        // Below the strip's own threshold the action bar keeps Back's arrow and drops its word, as the strip does.
        SizeChanged += (_, change) => Classes.Set("narrow", change.NewSize.Width < NavWidth.LabelsMinWidth);
    }
}
