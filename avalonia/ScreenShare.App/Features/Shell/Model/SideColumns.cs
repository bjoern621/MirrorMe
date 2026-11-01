namespace ScreenShare.App.Features.Shell.Model;

/// <summary>
/// Every side column this app draws, with the body each needs beside it.
/// One table read by the two destinations and by the tests, so a width is stated where the column is described
/// and nowhere else.
/// </summary>
public static class SideColumns
{
    /// <summary>
    /// Watch settings, opened from the viewer's rail.
    /// The body it measures against is the tile grid and the stream rail together, both standing on the same
    /// window: measured against the grid alone, two side columns would leave the tiles a strip between them.
    /// </summary>
    public static readonly SideColumn ViewerWatch = new(330, 540, ColumnEdge.Right, DrawnUnasked: false);

    /// <summary>
    /// Stream rail carrying the names.
    /// Below its threshold the rail keeps its entries and drops the names, which is what the reader's own collapse
    /// does (<c>Features/Viewer/ViewModel/ViewerViewModel.cs</c>).
    /// </summary>
    public static readonly SideColumn ViewerRail = new(240, 300, ColumnEdge.Left, DrawnUnasked: true);
}
