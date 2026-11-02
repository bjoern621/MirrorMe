namespace ScreenShare.App.Features.Shell.Model;

/// <summary>
/// Every side column this app draws, with the body each needs beside it.
/// One table read by the two destinations and by the tests, so a width is stated where the column is described
/// and nowhere else.
/// </summary>
public static class SideColumns
{
    /// <summary>
    /// Watch settings, opened from the nav strip on the viewer.
    /// Measured against a stage that keeps a stream watchable beside the panel.
    /// </summary>
    public static readonly SideColumn ViewerWatch = new(330, 540, ColumnEdge.Right, DrawnUnasked: false);
}
