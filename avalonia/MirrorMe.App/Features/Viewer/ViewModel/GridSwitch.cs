using MirrorMe.App.Contracts;
using MirrorMe.App.Copy;
using MirrorMe.App.Features.Viewer.Model;
using MirrorMe.App.Mvvm;

namespace MirrorMe.App.Features.Viewer.ViewModel;

/// <summary>
/// Strip's last item while a stage stands over it, putting every stream in equal cells.
/// One instance per viewer, so the strip keeps its container across passes.
/// </summary>
public sealed class GridSwitch : IStripItem
{
    public GridSwitch(DelegateCommand toggle)
    {
        Toggle = Assert.NotNull(toggle, "the switch runs the viewer's own grid toggle");
    }

    public DelegateCommand Toggle { get; }

    public string Tip => Cards.GridSwitchTip;

    public double Aspect => TileLayout.UnknownAspect;

    public bool IsFocused => false;
}
