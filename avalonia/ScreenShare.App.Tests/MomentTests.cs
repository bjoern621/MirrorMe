using ScreenShare.App.Mvvm;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// A statement that stands for a moment, which is how a control answers a press
/// without holding the slot it borrowed.
///
/// Defects locked out: a re-render handing the reader another moment, an ended moment starting
/// itself again off the answer that is still true, and one moment's end cutting the next one short.
/// </summary>
public sealed class MomentTests
{
    /// <summary>Holds the ends handed over, so a test runs them rather than waiting them out.</summary>
    private sealed class Ends
    {
        private readonly List<Action> _due = [];

        public int Standing => _due.Count;

        public TimeSpan Lasts { get; private set; }

        public Action<TimeSpan, Action> After => (lasts, end) =>
        {
            Lasts = lasts;
            _due.Add(end);
        };

        public void Pass()
        {
            var due = _due.ToArray();
            _due.Clear();
            foreach (var end in due)
            {
                end();
            }
        }

        /// <summary>Runs the end handed over first, which a later moment may have outlived.</summary>
        public void PassOldest()
        {
            var end = _due[0];
            _due.RemoveAt(0);
            end();
        }
    }

    private static readonly TimeSpan Lasts = TimeSpan.FromSeconds(5);

    /// <summary>Something to state stands until its moment ends, and the end is worth a render pass.</summary>
    [Fact]
    public void SomethingToStateStandsUntilItsMomentEnds()
    {
        var ends = new Ends();
        var passes = 0;
        var moment = new Moment(Lasts, ends.After, () => passes++);

        moment.Show(true);

        Assert.True(moment.Standing);
        Assert.Equal(Lasts, ends.Lasts);
        Assert.Equal(0, passes);

        ends.Pass();

        Assert.False(moment.Standing);
        Assert.Equal(1, passes);
    }

    /// <summary>
    /// The same answer twice leaves the moment where it is, a render pass being run for anything.
    /// </summary>
    [Fact]
    public void TheSameAnswerLeavesTheMomentWhereItIs()
    {
        var ends = new Ends();
        var moment = new Moment(Lasts, ends.After, () => { });

        moment.Show(true);
        moment.Show(true);
        moment.Show(true);

        Assert.Equal(1, ends.Standing);
    }

    /// <summary>
    /// An answer that outlives its own moment starts no second one:
    /// the state saying "up to date" holds long after the line about it has gone.
    /// </summary>
    [Fact]
    public void AnAnswerOutlivingItsMomentStartsNoSecondOne()
    {
        var ends = new Ends();
        var moment = new Moment(Lasts, ends.After, () => { });

        moment.Show(true);
        ends.Pass();
        moment.Show(true);

        Assert.False(moment.Standing);
        Assert.Equal(0, ends.Standing);
    }

    /// <summary>
    /// An answer going away and coming back is a moment of its own, which is a second check
    /// saying what it found.
    /// </summary>
    [Fact]
    public void AnAnswerComingBackGetsAMomentOfItsOwn()
    {
        var ends = new Ends();
        var moment = new Moment(Lasts, ends.After, () => { });

        moment.Show(true);
        moment.Show(false);

        Assert.False(moment.Standing);

        moment.Show(true);

        Assert.True(moment.Standing);
        Assert.Equal(2, ends.Standing);
    }

    /// <summary>
    /// An end its moment outlived stops nothing:
    /// a check landing inside the last one's five seconds gets the whole five of its own.
    /// </summary>
    [Fact]
    public void AnEndItsMomentOutlivedStopsNothing()
    {
        var ends = new Ends();
        var passes = 0;
        var moment = new Moment(Lasts, ends.After, () => passes++);

        moment.Show(true);
        moment.Show(false);
        moment.Show(true);

        ends.PassOldest();

        Assert.True(moment.Standing);
        Assert.Equal(0, passes);
    }
}
