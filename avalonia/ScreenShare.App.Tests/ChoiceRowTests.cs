using ScreenShare.App.Controls;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A question's choices sit side by side where each keeps a readable width, and wrap into rows where they cannot.
/// The widths are a column at a phone-sized tile, half a screen and a desktop window.
/// </summary>
public sealed class ChoiceRowTests
{
    [Theory]
    [InlineData(336, 3, 1)]
    [InlineData(576, 3, 3)]
    [InlineData(576, 4, 3)]
    [InlineData(1016, 4, 4)]
    [InlineData(1016, 7, 5)]
    [InlineData(1016, 1, 1)]
    public void ChoicesTakeAsManyColumnsAsKeepEachReadable(double width, int count, int columns)
    {
        Assert.Equal(columns, ChoiceRowLayout.Columns(width, count));
    }

    [Fact]
    public void NoChoicesTakeOneColumn()
    {
        Assert.Equal(1, ChoiceRowLayout.Columns(576, 0));
    }
}
