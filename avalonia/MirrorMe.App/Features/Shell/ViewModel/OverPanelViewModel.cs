using MirrorMe.App.Contracts;
using MirrorMe.App.Features.Shell.Model;
using MirrorMe.App.Mvvm;

namespace MirrorMe.App.Features.Shell.ViewModel;

/// <summary>
/// One panel opened over a surface from the nav strip, read against the width the window has
/// (<c>docs/design-language.md</c>, "Narrow windows").
/// The surface under it keeps its whole width: a wide window floats the panel at its edge,
/// and a narrow one raises it as a sheet from the foot.
///
/// Owned here: whether the reader has the panel open.
/// </summary>
public sealed class OverPanelViewModel : Observable
{
    private readonly string _openTip;
    private readonly string _closeTip;

    /// <summary>Width the window last stated. Infinite until it states one, which floats the panel.</summary>
    private double _window = double.PositiveInfinity;

    private bool _opened;

    public OverPanelViewModel(string openTip, string closeTip)
    {
        Assert.That(openTip.Length > 0 && closeTip.Length > 0, "a toggle names both states it leads to", openTip, closeTip);

        _openTip = openTip;
        _closeTip = closeTip;

        Toggle = new DelegateCommand(() =>
        {
            _opened = !_opened;
            Apply();
        });
    }

    /// <summary>Opens the panel, or closes it again.</summary>
    public DelegateCommand Toggle { get; }

    public bool IsOpen => _opened;

    /// <summary>Whether the panel rises from the foot across the window rather than floating at its edge.</summary>
    public bool IsSheet => _window < NavWidth.LabelsMinWidth;

    public string ToggleTip => _opened ? _closeTip : _openTip;

    /// <summary>States the width the window has. The same width twice moves nothing.</summary>
    public void SetWindowWidth(double width)
    {
        Assert.That(width > 0, "a window states a width it has", width);

        _window = width;
        Apply();
    }

    /// <summary>Closes the panel. A panel already closed is the state the caller asked for.</summary>
    public void Close()
    {
        _opened = false;
        Apply();
    }

    /// <summary>One render function: every property here is derived, so the pass raises them all.</summary>
    public void Apply()
    {
        OnPropertyChanged(nameof(IsOpen));
        OnPropertyChanged(nameof(IsSheet));
        OnPropertyChanged(nameof(ToggleTip));
    }
}
