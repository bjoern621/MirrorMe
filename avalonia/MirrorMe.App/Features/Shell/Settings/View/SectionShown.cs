using System.Globalization;
using Avalonia.Data;
using Avalonia.Data.Converters;

namespace MirrorMe.App.Features.Shell.Settings.View;

/// <summary>Whether the section on screen is the one a panel draws, named in the binding's parameter.</summary>
public sealed class SectionShown : IValueConverter
{
    public static readonly SectionShown Instance = new();

    public object? Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
        => Equals(value, parameter);

    public object? ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture)
        => BindingOperations.DoNothing;
}
