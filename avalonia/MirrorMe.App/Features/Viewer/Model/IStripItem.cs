namespace MirrorMe.App.Features.Viewer.Model;

/// <summary>
/// Whatever stands in the strip under the stage and in the grid's cells:
/// a playing stream, a stopped one, a member sharing nothing, the Grid switch.
/// The panel places each at its shape and stages the focused one (<c>Features/Viewer/View/TileGrid.cs</c>).
/// </summary>
public interface IStripItem
{
    /// <summary>Width over height.</summary>
    double Aspect { get; }

    bool IsFocused { get; }
}

/// <summary>
/// A member of the group publishing nothing, standing where their stream would.
/// Record, so two passes over one group compare equal and the strip keeps the item.
/// </summary>
public sealed record MemberHint(string Line) : IStripItem
{
    public double Aspect => TileLayout.UnknownAspect;

    public bool IsFocused => false;
}
