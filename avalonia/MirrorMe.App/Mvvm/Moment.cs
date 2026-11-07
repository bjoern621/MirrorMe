using MirrorMe.App.Contracts;

namespace MirrorMe.App.Mvvm;

/// <summary>
/// A statement that stands for a moment and then stops standing.
///
/// What a control says in answer to a press outlives the press by a moment, and the slot goes back
/// to whatever holds it the rest of the time.
/// A line still standing an hour on says nothing about the press that put it there.
///
/// Nothing is drawn here.
/// <see cref="Standing"/> is what a render pass reads,
/// and a moment ending calls back so the owner of that pass runs another.
/// </summary>
public sealed class Moment
{
    private readonly TimeSpan _lasts;
    private readonly Action<TimeSpan, Action> _after;
    private readonly Action _passed;

    // The last answer <see cref="Show"/> was given, which is the edge a moment starts on.
    // Held apart from _standing, whose end comes while that answer still holds:
    // a moment keyed off the standing would start itself again the moment it ended.
    private bool _asked;
    private bool _standing;

    // Which moment an end belongs to. An end carrying an older number is one its moment outlived.
    private int _started;

    /// <param name="lasts">How long a moment stands.</param>
    /// <param name="after">Runs an end once <paramref name="lasts"/> has passed, on the UI loop.</param>
    /// <param name="passed">Called where a moment ended, for the render pass that follows it.</param>
    public Moment(TimeSpan lasts, Action<TimeSpan, Action> after, Action passed)
    {
        Assert.That(lasts > TimeSpan.Zero, "a moment lasts a while", lasts);
        Assert.NotNull(after, "a moment is ended by something");
        Assert.NotNull(passed, "a moment ending is worth a render pass");

        _lasts = lasts;
        _after = after;
        _passed = passed;
    }

    /// <summary>Whether the moment is standing, which is what the render pass draws off.</summary>
    public bool Standing => _standing;

    /// <summary>
    /// States whether there is something to stand for a moment.
    /// The same answer twice leaves the moment where it is, so a re-render does not extend one.
    /// Idempotent.
    /// </summary>
    public void Show(bool something)
    {
        if (something == _asked)
        {
            return;
        }

        _asked = something;
        _standing = something;
        _started++;

        if (something)
        {
            var started = _started;
            _after(_lasts, () => Pass(started));
        }

        Assert.That(!_standing || _asked, "a moment stands for something", _standing, _asked);
    }

    /// <summary>Ends one moment, and nothing where the moment it belongs to is over.</summary>
    private void Pass(int started)
    {
        if (started != _started)
        {
            return;
        }

        _standing = false;
        _passed();
    }
}
