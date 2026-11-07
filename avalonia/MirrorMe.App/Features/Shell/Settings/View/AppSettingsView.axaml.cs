using System.ComponentModel;
using Avalonia.Controls;
using Avalonia.Input;
using Avalonia.Threading;
using MirrorMe.App.Features.Shell.Settings.ViewModel;

namespace MirrorMe.App.Features.Shell.Settings.View;

/// <summary>
/// Two things the markup cannot state: the press on the dimmed ground and a section opening at its top.
/// Every setting the dialog draws writes through its own binding.
/// </summary>
public sealed partial class AppSettingsView : UserControl
{
    /// <summary>Dialog behind this view, held so its news is dropped when the binding moves.</summary>
    private AppSettingsViewModel? _settings;

    public AppSettingsView()
    {
        InitializeComponent();

        DataContextChanged += (_, _) => Follow(DataContext as AppSettingsViewModel);
    }

    /// <summary>
    /// Takes the dialog's news, dropping the last one's.
    /// The scroll follows the model rather than this control's own visibility:
    /// the Overlay carries whether the dialog is drawn, and this control is in the tree either way.
    /// </summary>
    private void Follow(AppSettingsViewModel? settings)
    {
        if (_settings is not null)
        {
            _settings.PropertyChanged -= Moved;
        }

        _settings = settings;

        if (_settings is not null)
        {
            _settings.PropertyChanged += Moved;
        }
    }

    private void Moved(object? sender, PropertyChangedEventArgs change)
    {
        if (_settings is not { IsOpen: true })
        {
            return;
        }

        // A section opens at its top, the offset otherwise carrying over from the last one shown.
        if (change.PropertyName is nameof(AppSettingsViewModel.IsOpen) or nameof(AppSettingsViewModel.Section))
        {
            Dispatcher.UIThread.Post(Sections.ScrollToHome, DispatcherPriority.Loaded);
        }
    }

    /// <summary>
    /// Press on the dimmed ground, which closes.
    /// The press has to have landed on the ground itself:
    /// one inside the dialog reaches this handler on its way out,
    /// and closing on it would dismiss the dialog whenever anything in it was clicked.
    /// </summary>
    private void ScrimPressed(object? sender, PointerPressedEventArgs e)
    {
        if (ReferenceEquals(e.Source, sender) && DataContext is AppSettingsViewModel settings)
        {
            settings.CloseCommand.Execute(null);
        }
    }
}
