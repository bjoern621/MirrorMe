using Avalonia.Controls;
using ScreenShare.App.Features.Shell.Model;

namespace ScreenShare.App.Features.Insights.Chart.View;

public sealed partial class ChartView : UserControl
{
    public ChartView()
    {
        InitializeComponent();

        // The figures and the range picker share a row down to the strip's own threshold, then stack.
        SizeChanged += (_, change) => Classes.Set("narrow", change.NewSize.Width < NavWidth.LabelsMinWidth);
    }
}
