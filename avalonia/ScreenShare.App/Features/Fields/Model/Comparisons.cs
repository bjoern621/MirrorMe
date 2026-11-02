using System.Reflection;
using System.Text.Json;
using ScreenShare.App.Contracts;
using ScreenShare.App.Mvvm;

namespace ScreenShare.App.Features.Fields.Model;

/// <summary>One picture of a comparison: the entry it shows, and its file under <c>Assets/Crops/</c>.</summary>
public sealed record Crop
{
    /// <summary>Entry's plain name: "Sharper".</summary>
    public required string Name { get; init; }

    /// <summary>What the picture was encoded at, where the entry carries no price of its own: "CRF 18".</summary>
    public required string Figure { get; init; }

    /// <summary>Settings value of the entry the picture shows, as the form offers it: "18".</summary>
    public required string Value { get; init; }

    public required string File { get; init; }

    /// <summary>Address the view loads the picture from.</summary>
    public string Asset => $"avares://mirrorme/Assets/Crops/{File}";
}

/// <summary>
/// Comparison crops per settings field, read from <c>Assets/Crops/manifest.json</c>.
/// The same manifest drives <c>tools/crops.sh</c>, which renders every crop through the encoder its entry names,
/// so the list of pictures and the pictures cannot drift apart.
/// Display data alone: the crops illustrate a choice and decide nothing about it.
/// </summary>
public static class Comparisons
{
    private const string ManifestResource = "ScreenShare.App.Assets.Crops.manifest.json";

    private static readonly IReadOnlyDictionary<string, ValueList<Crop>> ByField = Load();

    /// <summary>Crops comparing the entries of one field, empty for a field the manifest does not name.</summary>
    public static ValueList<Crop> Of(string fieldKey)
    {
        Assert.That(fieldKey.Length > 0, "a comparison lookup names the field it looks for");

        return ByField.TryGetValue(fieldKey, out var crops) ? crops : ValueList<Crop>.Empty;
    }

    private static IReadOnlyDictionary<string, ValueList<Crop>> Load()
    {
        using var stream = Assembly.GetExecutingAssembly().GetManifestResourceStream(ManifestResource)
            ?? Assert.Never<Stream>("the crop manifest is embedded in the app", ManifestResource);

        using var document = JsonDocument.Parse(stream);
        var byField = new Dictionary<string, ValueList<Crop>>();

        foreach (var field in document.RootElement.GetProperty("comparisons").EnumerateObject())
        {
            var crops = field.Value.EnumerateArray()
                .Select(entry => new Crop
                {
                    Name = entry.GetProperty("name").GetString() ?? "",
                    Figure = entry.GetProperty("figure").GetString() ?? "",
                    Value = entry.GetProperty("value").GetString() ?? "",
                    File = entry.GetProperty("file").GetString() ?? "",
                })
                .ToList();

            Assert.That(crops.Count >= 2, "a comparison sets at least two pictures side by side", field.Name, crops.Count);
            Assert.That(
                crops.All(crop => crop.Name.Length > 0 && crop.Figure.Length > 0 && crop.File.Length > 0),
                "every crop names what it shows, its figure and its file", field.Name);

            byField[field.Name] = new ValueList<Crop>(crops);
        }

        return byField;
    }
}
