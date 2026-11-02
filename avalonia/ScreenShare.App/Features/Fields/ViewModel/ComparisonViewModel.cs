using ScreenShare.App.Contracts;
using ScreenShare.App.Features.Fields.Model;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Fields.ViewModel;

/// <summary>One picture as the tip draws it: the entry's name, its figure, and whether the field holds it.</summary>
public sealed record PictureViewModel(string Asset, string Name, string Figure, bool IsPicked);

/// <summary>
/// A field's comparison pictures under a heading and the sentence saying where to look.
/// A record of records, so a pass over an unchanged field compares equal and the open tip is left alone.
/// </summary>
public sealed record ComparisonViewModel(string Title, string Sentence, ValueList<PictureViewModel> Pictures)
{
    /// <summary>
    /// Pictures for one field, null where the manifest names none.
    /// A picture takes its entry's price where the form sent one, so the tip and the cards print one figure.
    /// </summary>
    /// <param name="options">The field's entries as this pass drew them.</param>
    public static ComparisonViewModel? Of(string key, IReadOnlyList<OptionViewModel> options)
    {
        Assert.NotNull(options, "marking the picked picture needs the field's entries");

        var crops = Comparisons.Of(key);
        if (crops.Count == 0)
        {
            return null;
        }

        var copy = Copy.Pictures.Of(key)
            ?? Assert.Never<Copy.Pictures.Entry>("a field with pictures has a heading over them", key);

        var pictures = crops.Select(crop =>
        {
            var entry = options.FirstOrDefault(option => option.Value == crop.Value);
            return new PictureViewModel(
                crop.Asset,
                crop.Name,
                entry is { HasPrice: true } ? entry.Price : crop.Figure,
                entry is { IsSelected: true });
        });

        var comparison = new ComparisonViewModel(copy.Title, copy.Sentence, new ValueList<PictureViewModel>(pictures));
        Assert.That(
            comparison.Pictures.Count(picture => picture.IsPicked) <= 1,
            "one picture at most shows the entry the field holds", key);
        return comparison;
    }
}
