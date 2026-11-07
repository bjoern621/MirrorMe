using Avalonia;
using Avalonia.Controls;

namespace MirrorMe.App.Controls;

/// <summary>Fades a text in each time its words change, for a label that swaps on a press.</summary>
public static class TextFade
{
    public static readonly AttachedProperty<bool> IsOnProperty =
        AvaloniaProperty.RegisterAttached<TextBlock, bool>("IsOn", typeof(TextFade));

    static TextFade()
    {
        TextBlock.TextProperty.Changed.AddClassHandler<TextBlock>((text, _) =>
        {
            if (GetIsOn(text) && text.IsLoaded)
            {
                ArrangeMotion.Arrive(text);
            }
        });
    }

    public static bool GetIsOn(TextBlock text) => text.GetValue(IsOnProperty);

    public static void SetIsOn(TextBlock text, bool value) => text.SetValue(IsOnProperty, value);
}
