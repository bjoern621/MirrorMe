using Avalonia.Controls;
using Avalonia.Input;
using MirrorMe.App.Features.Shell.Consent.ViewModel;

namespace MirrorMe.App.Features.Shell.Consent.View;

/// <summary>
/// The press on the dimmed ground, which the markup cannot state and which answers.
/// The toggle the question is answered on writes through its own binding.
/// </summary>
public sealed partial class ConsentView : UserControl
{
    public ConsentView()
    {
        InitializeComponent();
    }

    /// <summary>
    /// Press on the dimmed ground, which answers and closes.
    /// The press has to have landed on the ground itself:
    /// one inside the dialog reaches this handler on its way out,
    /// and answering on it would close the question whenever anything in it was clicked.
    /// </summary>
    private void ScrimPressed(object? sender, PointerPressedEventArgs e)
    {
        if (ReferenceEquals(e.Source, sender) && DataContext is ConsentViewModel consent)
        {
            consent.CloseCommand.Execute(null);
        }
    }
}
