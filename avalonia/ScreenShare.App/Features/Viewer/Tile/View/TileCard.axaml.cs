using Avalonia;
using Avalonia.Controls;
using Avalonia.Input;
using Avalonia.Interactivity;
using Avalonia.Markup.Xaml;
using Avalonia.Media;
using Avalonia.VisualTree;

using ScreenShare.App.Contracts;
using ScreenShare.App.Controls;
using ScreenShare.App.Features.Viewer.Model;
using ScreenShare.App.Features.Viewer.Tile.Model;
using ScreenShare.App.Features.Viewer.Tile.ViewModel;
using ScreenShare.App.Features.Viewer.View;

namespace ScreenShare.App.Features.Viewer.Tile.View;

/// <summary>
/// A tile's code-behind, for every host that draws one.
/// Covers what markup cannot state: the meter's bar is a pixel width rather than a proportion any panel offers,
/// the volume slider has to survive the menu's own click handling, a gesture is not a binding,
/// and a pointer holding still is a fact no selector carries.
/// </summary>
public partial class TileCard : UserControl
{
    public TileCard()
    {
        InitializeComponent();

        // Bar width follows the level and this control's width, and the overlay follows the host's answer,
        // so all three are watched.
        // A converter takes one input, so the arithmetic sits here.
        PropertyChanged += (_, change) =>
        {
            if (change.Property == BoundsProperty
                || change.Property == DataContextProperty
                || change.Property == PictureElsewhereProperty)
            {
                Draw();
            }
        };

        DataContextChanged += (_, _) => Watch();

        // Held by the handlers it puts on this card, so it lasts as long as the card draws.
        _ = new RestingWatch(this, Rested);

        var frame = this.FindControl<Panel>("Frame");
        Assert.NotNull(frame, "a card draws its picture inside a frame");
        _gestures = new ZoomGestures(frame, Zoomable, () => _zoom, ZoomTo);
    }

    /// <summary>
    /// The pointer while it rests over a picture.
    /// One for every card: a cursor is a platform handle, and one per tile would open a handle each to say
    /// the same thing.
    /// </summary>
    private static readonly Cursor Hidden = new(StandardCursorType.None);

    /// <summary>Pointer's answer as the watch last reported it, the one input to a pass that no owner can be asked for.</summary>
    private bool _resting;

    /// <summary>
    /// How far this card is zoomed into its picture, kept per card: a popped-out window and the grid slot
    /// it left zoom apart.
    /// </summary>
    private TileZoom _zoom = TileZoom.None;

    private readonly ZoomGestures _gestures;

    private void ZoomTo(TileZoom zoom)
    {
        _zoom = zoom;
        Draw();
    }

    /// <summary>
    /// Whether a gesture over this card zooms it.
    /// A strip tile under a focused one is too small to zoom into, so its wheel goes on to scroll the strip.
    /// </summary>
    private bool Zoomable() =>
        !PictureElsewhere
        && _watched is not null
        && (!InGrid || _watched.IsFocused || this.FindAncestorOfType<TileGrid>() is not { Mode: LayoutMode.Focus });

    /// <summary>Takes the pointer's answer and draws on it.</summary>
    private void Rested(bool resting)
    {
        _resting = resting;
        Draw();
    }

    /// <summary>
    /// Whether this stream's picture is in another window, which makes this card draw the plate saying so.
    ///
    /// The host's answer.
    /// A popped-out stream is drawn by two cards off one tile, the grid slot it keeps and the window it moved to,
    /// so a card reading this from the tile would show the plate twice and the picture nowhere.
    ///
    /// The plate subscribes to nothing: the style behind it clears the picture's source, ending the channel,
    /// giving the pool back and stopping a render size being asked for (<see cref="StreamTile"/>).
    /// </summary>
    public static readonly StyledProperty<bool> PictureElsewhereProperty =
        AvaloniaProperty.Register<TileCard, bool>(nameof(PictureElsewhere));

    public bool PictureElsewhere
    {
        get => GetValue(PictureElsewhereProperty);
        set => SetValue(PictureElsewhereProperty, value);
    }

    /// <summary>
    /// Whether this card is a slot in the arranged grid, which is what makes a click on it mean focus.
    ///
    /// The host's answer, as <see cref="PictureElsewhere"/> is.
    /// Focus arranges the grid's tiles.
    /// A click in the window a stream popped out to, or on the screen it fills,
    /// would move tiles nobody is looking at and leave the grid rearranged for whoever comes back.
    /// </summary>
    public static readonly StyledProperty<bool> InGridProperty =
        AvaloniaProperty.Register<TileCard, bool>(nameof(InGrid));

    public bool InGrid
    {
        get => GetValue(InGridProperty);
        set => SetValue(InGridProperty, value);
    }

    private TileViewModel? _watched;
    private TopLevel? _root;

    /// <summary>
    /// Subscribes to the tile's keys on the window this card sits in.
    ///
    /// The keys belong to the tile, and the pointer picks which tile a press means (<see cref="TileKeys"/>).
    /// A window-level shortcut would have to invent that rule,
    /// and every candidate for it, the focused tile or the last one touched, adds arrangement state to maintain.
    ///
    /// The handler goes on the window because keys arrive at whatever holds the keyboard, which a tile never does:
    /// taking focus on hover would steal it from whatever was being typed in.
    /// </summary>
    protected override void OnAttachedToVisualTree(VisualTreeAttachmentEventArgs e)
    {
        base.OnAttachedToVisualTree(e);

        _root = TopLevel.GetTopLevel(this);
        _root?.AddHandler(KeyDownEvent, OnKeyPressed, RoutingStrategies.Bubble);
    }

    protected override void OnDetachedFromVisualTree(VisualTreeAttachmentEventArgs e)
    {
        // The handler lives on the window, so leaving the tree takes it off again.
        // A window outlives its cards, and handlers for cards nobody draws would pile up for as long as it is open.
        _root?.RemoveHandler(KeyDownEvent, OnKeyPressed);
        _root = null;

        base.OnDetachedFromVisualTree(e);
    }

    /// <summary>
    /// Tracks the view model's notifications, so a landed level redraws the meter
    /// and an overlay turned on raises the chrome with it.
    /// Levels arrive on a path of their own fifteen times a second and move two properties.
    /// Redrawing resizes one border and writes the classes it already carries
    /// (<c>Backend/Session.cs</c>, <c>Metered</c>).
    /// </summary>
    private void Watch()
    {
        if (_watched is not null)
        {
            _watched.PropertyChanged -= OnTileChanged;
        }

        // Another stream on this card starts unzoomed.
        _zoom = TileZoom.None;
        _watched = DataContext as TileViewModel;
        if (_watched is not null)
        {
            _watched.PropertyChanged += OnTileChanged;
        }

        Draw();
    }

    private void OnTileChanged(object? sender, System.ComponentModel.PropertyChangedEventArgs change)
    {
        if (change.PropertyName is nameof(TileViewModel.Level)
            or nameof(TileViewModel.HasLevel)
            or nameof(TileViewModel.ShowStats)
            or nameof(TileViewModel.IsFocused))
        {
            Draw();
        }
    }

    /// <summary>
    /// Reports the size this card arranged at, in its own pixels.
    ///
    /// The pointer arrives as a fraction of the picture and is drawn in these, so the conversion needs both.
    /// The view model reads the picture's shape off the frames, and this is the only source for the card's size.
    /// </summary>
    protected override Size ArrangeOverride(Size size)
    {
        var arranged = base.ArrangeOverride(size);
        _watched?.SetPictureSize(arranged.Width, arranged.Height);
        return arranged;
    }

    /// <summary>
    /// Draws what markup cannot decide: the overlay, the plates and chrome under a resting pointer, the cursor
    /// and the meter's bar at the level in pixels of this card.
    ///
    /// The overlay opens and fades on its own state, the class is what the markup hides the plates and chrome on,
    /// and the cursor is this control's own property, inherited by everything drawn inside it.
    /// Every pass writes all of them, and each write is the state named rather than a change to it,
    /// so a pass per pointer move and a pass per level costs nothing.
    /// </summary>
    private void Draw()
    {
        var chrome = TileChrome.Of(_resting, _watched is { ShowStats: true }, PictureElsewhere);

        if (this.FindControl<Overlay>("Stats") is { } stats)
        {
            stats.IsOpen = chrome.Stats;
        }

        DrawZoom();

        PseudoClasses.Set(":resting", chrome.Resting);
        Cursor = chrome.HidesCursor ? Hidden : null;

        if (this.FindControl<Border>("Level") is not { } bar)
        {
            return;
        }

        bar.Width = _watched is { HasLevel: true } tile ? Math.Max(0, Bounds.Width * tile.Level) : 0;
    }

    /// <summary>
    /// Writes the zoom onto the picture, onto the size it asks to be decoded at and onto the map over it.
    /// A card that stopped being zoomable, a focused tile gone back to the strip, draws unzoomed
    /// and keeps its zoom for its next turn on the stage.
    /// </summary>
    private void DrawZoom()
    {
        if (this.FindControl<Panel>("Frame") is not { } frame
            || this.FindControl<Panel>("Zoomed") is not { } zoomed
            || this.FindControl<StreamTile>("Picture") is not { } picture
            || this.FindControl<ZoomMap>("Map") is not { } map)
        {
            return;
        }

        var zoom = Zoomable() ? _zoom : TileZoom.None;
        var matrix = zoom.Transform(frame.Bounds.Size);
        if (zoomed.RenderTransform is not MatrixTransform { Matrix: var drawn } || drawn != matrix)
        {
            zoomed.RenderTransform = new MatrixTransform(matrix);
        }
        picture.Zoom = zoom.Scale;

        map.Zoom = zoom;
        if (frame.Bounds is { Width: > 0, Height: > 0 } laid)
        {
            map.Aspect = laid.Height / laid.Width;
        }
    }

    /// <summary>
    /// Makes the volume slider draggable inside the menu.
    /// A press on a menu item is a click and a click closes the flyout,
    /// so untouched the slider takes one press and the drag lands on a menu already dismissed.
    /// Handling the pointer events keeps them off the item,
    /// <c>StaysOpenOnClick</c> in the markup covering the press that still reaches it.
    /// </summary>
    private void OnVolumePressed(object? sender, PointerPressedEventArgs e) => e.Handled = true;

    /// <summary>
    /// A left click toggles focus, the state the menu's first row and its key name.
    ///
    /// The tile is what a reader points at to say which stream they want big,
    /// so the picture carries the gesture and there is nothing to find first.
    /// A click on the focused tile gives the space back, as a second press of the key does.
    ///
    /// Unhandled taps alone, which leaves the parts of a card a reader clicks to use with their own clicks
    /// (<see cref="OnClickTaken"/>).
    /// Right-click reaches the menu instead, a tap and a right tap being separate events.
    /// </summary>
    protected override void OnTapped(TappedEventArgs e)
    {
        base.OnTapped(e);

        // The release ending a drag of the picture.
        if (_gestures.Dragged)
        {
            e.Handled = true;
            return;
        }

        if (!InGrid || DataContext is not TileViewModel tile)
        {
            return;
        }

        tile.ToggleFocus.Execute(null);
        e.Handled = true;
    }

    /// <summary>
    /// Takes the click for the parts of a card a reader clicks to use:
    /// the error text they select and copy, and the stats overlay they read.
    /// Rearranging the grid under either loses them what they were pointing at.
    /// </summary>
    private void OnClickTaken(object? sender, TappedEventArgs e) => e.Handled = true;

    /// <summary>
    /// Applies what a key names to the tile under the pointer.
    /// The states the menu's rows name, on the keys those rows print beside them,
    /// so reading the menu once is enough.
    /// Each key but the two arrows names a state, so a second press is a round trip,
    /// the volume keys naming the level they want for the same reason.
    /// An arrow steps the stage, so a second press steps it again (<see cref="TileKeys"/>).
    /// </summary>
    private void OnKeyPressed(object? sender, KeyEventArgs e)
    {
        // The pointer selects the tile, so a press with it elsewhere belongs to another card.
        // Two cards can draw one stream, a grid slot beneath a window filled with the picture,
        // and only the one under the pointer hit-tests as over.
        if (!IsPointerOver || DataContext is not TileViewModel tile)
        {
            return;
        }

        // In a text box a letter is text.
        // Keys start at whatever holds the keyboard,
        // which separates somebody typing in the settings panel from somebody watching the tile under a pointer.
        if (e.Source is TextBox)
        {
            return;
        }

        if (TileKeys.Command(tile, e) is not { } command)
        {
            return;
        }

        // Refused wherever the row the key names is greyed:
        // a stream without a sound track has no level to move and nothing to silence.
        // The press stays unhandled, a key this tile cannot answer not being its own.
        if (!command.CanExecute(null))
        {
            return;
        }

        command.Execute(null);
        e.Handled = true;
    }

    private void InitializeComponent() => AvaloniaXamlLoader.Load(this);
}
