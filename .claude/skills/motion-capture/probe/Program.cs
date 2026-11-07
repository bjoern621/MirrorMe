using System.Diagnostics;
using Avalonia;
using Avalonia.Controls;
using Avalonia.Headless;
using Avalonia.Threading;
using Avalonia.VisualTree;
using MirrorMe.App;
using MirrorMe.App.Features.Viewer.Tile.View;
using MirrorMe.App.Features.Viewer.Tile.ViewModel;
using MirrorMe.App.Features.Viewer.View;
using MirrorMe.App.Tests;

AppBuilder.Configure<App>().UseSkia()
    .UseHeadless(new AvaloniaHeadlessPlatformOptions { UseHeadlessDrawing = false })
    .SetupWithoutStarting();

// The scenario: a viewer with two tiles, left and opened again as a tab switch does.
var (viewer, _) = Flows.Watching(new ViewerBackend("one", "two"));

var host = new ContentControl();
var window = new Window { Width = 1500, Height = 1000, Content = host };
window.Show();
Pump(300);

host.Content = new ViewerView { DataContext = viewer };
Sample("first mount", 700);

host.Content = new Border();
Pump(300);

host.Content = new ViewerView { DataContext = viewer };
Sample("second mount", 700);

void Pump(int milliseconds)
{
    var clock = Stopwatch.StartNew();
    while (clock.ElapsedMilliseconds < milliseconds)
    {
        Tick();
        Thread.Sleep(8);
    }
}

void Sample(string label, int milliseconds)
{
    Console.WriteLine($"== {label}");
    var clock = Stopwatch.StartNew();
    while (clock.ElapsedMilliseconds < milliseconds)
    {
        Tick();
        var cards = window.GetVisualDescendants().OfType<TileCard>().Select(Reading);
        Console.WriteLine($"{clock.ElapsedMilliseconds,4}  {string.Join("  |  ", cards)}");
        Thread.Sleep(20);
    }
}

static void Tick()
{
    AvaloniaHeadlessPlatform.ForceRenderTimerTick();
    Dispatcher.UIThread.RunJobs();
}

static string Reading(TileCard card)
{
    var frame = card.GetVisualDescendants().OfType<Border>().First(border => border.Classes.Contains("frame"));
    var name = (card.DataContext as TileViewModel)?.Name;
    return $"{name} card {card.Opacity:F2} sc {card.RenderTransform?.Value.M11:F2}"
        + $" frame {frame.Opacity:F2} sc {frame.RenderTransform?.Value.M11:F2}";
}
