using Avalonia.Controls;
using ScreenShare.App.Features.Shell.Model;

namespace ScreenShare.App.Features.Insights.View;

/// <summary>
/// Markup apart from the narrow class.
/// Every control here binds to a command or a rendered value, so a handler would be a second definition
/// of the screen, one <c>InsightsViewModel.Apply</c> does not restore.
/// No window, no title bar and no nav strip: the shell supplies those and this is the body inside them.
/// </summary>
public sealed partial class InsightsView : UserControl
{
    public InsightsView()
    {
        InitializeComponent();

        // Below the strip's own threshold Edit in setup keeps its pencil and drops its words, as the strip does.
        SizeChanged += (_, change) => Classes.Set("narrow", change.NewSize.Width < NavWidth.LabelsMinWidth);
    }
}
