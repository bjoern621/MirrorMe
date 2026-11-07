using Avalonia.Controls;
using Avalonia.Input;
using MirrorMe.App.Features.Setup.SharePicker.ViewModel;

namespace MirrorMe.App.Features.Setup.SharePicker.View;

/// <summary>
/// The press on the dimmed ground, which the markup cannot state and which dismisses.
/// Every control the dialog draws writes through its own binding.
/// </summary>
public sealed partial class SharePickerView : UserControl
{
    public SharePickerView()
    {
        InitializeComponent();
    }

    /// <summary>
    /// Press on the dimmed ground, which drops the press that opened this.
    /// The press has to have landed on the ground itself:
    /// one inside the dialog reaches this handler on its way out,
    /// and closing on it would dismiss the dialog whenever anything in it was clicked.
    /// </summary>
    private void ScrimPressed(object? sender, PointerPressedEventArgs e)
    {
        if (ReferenceEquals(e.Source, sender) && DataContext is SharePickerViewModel picker)
        {
            picker.CancelCommand.Execute(null);
        }
    }
}
