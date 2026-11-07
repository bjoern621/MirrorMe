using System.Globalization;
using Avalonia.Data.Converters;
using Avalonia.Media.Imaging;
using Avalonia.Platform;
using MirrorMe.App.Contracts;

namespace MirrorMe.App.Controls;

/// <summary>
/// Loads a picture shipped inside the app from its <c>avares://</c> address.
/// Each address decodes once and is kept for the life of the process: the set is fixed at build time and small.
/// </summary>
public sealed class AssetBitmapConverter : IValueConverter
{
    public static readonly AssetBitmapConverter Instance = new();

    private readonly Dictionary<string, Bitmap> _loaded = [];

    public object Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        Assert.That(value is string { Length: > 0 }, "a shipped picture is read from its address", value);

        var address = (string)value!;
        if (!_loaded.TryGetValue(address, out var bitmap))
        {
            bitmap = new Bitmap(AssetLoader.Open(new Uri(address)));
            _loaded[address] = bitmap;
        }
        return bitmap;
    }

    public object ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture)
        => Assert.Never<object>("a shipped picture is never written back", value);
}
